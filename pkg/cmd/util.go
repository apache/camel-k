/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/apache/camel-k/pkg/util/gzip"
	"github.com/pkg/errors"

	"github.com/mitchellh/mapstructure"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	p "github.com/gertd/go-pluralize"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	offlineCommandLabel = "camel.apache.org/cmd.offline"

	// Supported source schemes
	gistScheme   = "gist"
	githubScheme = "github"
	httpScheme   = "http"
	httpsScheme  = "https"
)

// DeleteIntegration --
func DeleteIntegration(ctx context.Context, c client.Client, name string, namespace string) error {
	integration := v1.Integration{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.IntegrationKind,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	return c.Delete(ctx, &integration)
}

func bindPFlagsHierarchy(cmd *cobra.Command) error {
	for _, c := range cmd.Commands() {
		if err := bindPFlags(c); err != nil {
			return err
		}

		if err := bindPFlagsHierarchy(c); err != nil {
			return err
		}
	}

	return nil
}

func bindPFlags(cmd *cobra.Command) error {
	prefix := pathToRoot(cmd)
	pl := p.NewClient()

	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		name := flag.Name
		name = strings.ReplaceAll(name, "_", "-")
		name = strings.ReplaceAll(name, ".", "-")

		if err := viper.BindPFlag(prefix+"."+name, flag); err != nil {
			log.Printf("error binding flag %s with prefix %s to viper: %v", flag.Name, prefix, err)
		}

		// this is a little bit of an hack to register plural version of properties
		// based on the naming conventions used by the flag type because it is not
		// possible to know what is the type of a flag
		flagType := strings.ToUpper(flag.Value.Type())
		if strings.Contains(flagType, "SLICE") || strings.Contains(flagType, "ARRAY") {
			if err := viper.BindPFlag(prefix+"."+pl.Plural(name), flag); err != nil {
				log.Printf("error binding plural flag %s with prefix %s to viper: %v", flag.Name, prefix, err)
			}
		}
	})

	return nil
}

func pathToRoot(cmd *cobra.Command) string {
	path := cmd.Name()

	for current := cmd.Parent(); current != nil; current = current.Parent() {
		name := current.Name()
		name = strings.ReplaceAll(name, "_", "-")
		name = strings.ReplaceAll(name, ".", "-")
		path = name + "." + path
	}

	return path
}

func decodeKey(target interface{}, key string) error {
	nodes := strings.Split(key, ".")
	settings := viper.AllSettings()

	for _, node := range nodes {
		v := settings[node]

		if v == nil {
			return nil
		}

		if m, ok := v.(map[string]interface{}); ok {
			settings = m
		} else {
			return fmt.Errorf("unable to find node %s", node)
		}
	}

	c := mapstructure.DecoderConfig{
		Result:           target,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToIPNetHookFunc(),
			mapstructure.StringToTimeDurationHookFunc(),
			stringToSliceHookFunc(','),
		),
	}

	decoder, err := mapstructure.NewDecoder(&c)
	if err != nil {
		return err
	}

	err = decoder.Decode(settings)
	if err != nil {
		return err
	}

	return nil
}

func decode(target interface{}) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		path := pathToRoot(cmd)
		if err := decodeKey(target, path); err != nil {
			return err
		}

		return nil
	}
}

func stringToSliceHookFunc(comma rune) mapstructure.DecodeHookFunc {
	return func(f reflect.Kind, t reflect.Kind, data interface{}) (interface{}, error) {
		if f != reflect.String || t != reflect.Slice {
			return data, nil
		}

		s, ok := data.(string)
		if !ok {
			return []string{}, nil
		}
		s = strings.TrimPrefix(s, "[")
		s = strings.TrimSuffix(s, "]")

		if s == "" {
			return []string{}, nil
		}

		stringReader := strings.NewReader(s)
		csvReader := csv.NewReader(stringReader)
		csvReader.Comma = comma
		csvReader.LazyQuotes = true

		return csvReader.Read()
	}
}

func cmdOnly(cmd *cobra.Command, options interface{}) *cobra.Command {
	return cmd
}

func isOfflineCommand(cmd *cobra.Command) bool {
	return cmd.Annotations[offlineCommandLabel] == "true"
}

func clone(dst interface{}, src interface{}) error {
	if dst == nil {
		return fmt.Errorf("dst cannot be nil")
	}
	if src == nil {
		return fmt.Errorf("src cannot be nil")
	}

	data, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("unable to marshal src: %w", err)
	}

	err = json.Unmarshal(data, dst)
	if err != nil {
		return fmt.Errorf("unable to unmarshal into dst: %w", err)
	}
	return nil
}

func fieldByMapstructureTagName(target reflect.Value, tagName string) (reflect.StructField, bool) {
	pl := p.NewClient()

	for i := 0; i < target.Type().NumField(); i++ {
		f := target.Type().Field(i)

		tag, ok := f.Tag.Lookup(MapstructureTagName)
		if !ok {
			continue
		}

		if tag == tagName {
			return f, true
		} else if tag == pl.Plural(tagName) {
			return f, true
		}
	}

	return reflect.StructField{}, false
}

func compressToString(content []byte) (string, error) {
	bytes, err := gzip.CompressBase64(content)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func isLocalAndFileExists(uri string) (bool, error) {
	if hasSupportedScheme(uri) {
		// it's not a local file as it matches one of the supporting schemes
		return false, nil
	}
	info, err := os.Stat(uri)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		// If it is a different error (ie, permission denied) we should report it back
		return false, errors.Wrap(err, fmt.Sprintf("file system error while looking for %s", uri))
	}
	return !info.IsDir(), nil
}

func hasSupportedScheme(uri string) bool {
	if strings.HasPrefix(strings.ToLower(uri), gistScheme+":") ||
		strings.HasPrefix(strings.ToLower(uri), githubScheme+":") ||
		strings.HasPrefix(strings.ToLower(uri), httpScheme+":") ||
		strings.HasPrefix(strings.ToLower(uri), httpsScheme+":") {
		return true
	}

	return false
}
