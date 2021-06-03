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

package digest

import (
	// nolint: gosec
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/flow"
)

// ComputeForIntegration a digest of the fields that are relevant for the deployment
// Produces a digest that can be used as docker image tag
func ComputeForIntegration(integration *v1.Integration) (string, error) {
	hash := sha256.New()
	// Integration version is relevant
	if _, err := hash.Write([]byte(integration.Status.Version)); err != nil {
		return "", err
	}
	// Integration Kit is relevant
	if integration.Spec.IntegrationKit != nil {
		if _, err := hash.Write([]byte(fmt.Sprintf("%s/%s", integration.Spec.IntegrationKit.Namespace, integration.Spec.IntegrationKit.Name))); err != nil {
			return "", err
		}
	}
	// Profile is relevant
	if _, err := hash.Write([]byte(integration.Spec.Profile)); err != nil {
		return "", err
	}

	// Integration code
	for _, s := range integration.Spec.Sources {
		if s.Content != "" {
			if _, err := hash.Write([]byte(s.Content)); err != nil {
				return "", err
			}
		}
	}

	// Integration resources
	for _, item := range integration.Spec.Resources {
		if _, err := hash.Write([]byte(item.Content)); err != nil {
			return "", err
		}
	}

	// Integration flows
	if len(integration.Spec.Flows) > 0 {
		flows, err := flow.ToYamlDSL(integration.Spec.Flows)
		if err != nil {
			return "", err
		}
		if _, err := hash.Write(flows); err != nil {
			return "", err
		}
	}

	// Integration dependencies
	for _, item := range integration.Spec.Dependencies {
		if _, err := hash.Write([]byte(item)); err != nil {
			return "", err
		}
	}

	// Integration configuration
	for _, item := range integration.Spec.Configuration {
		if _, err := hash.Write([]byte(item.String())); err != nil {
			return "", err
		}
	}

	// Integration traits
	for _, name := range sortedTraitSpecMapKeys(integration.Spec.Traits) {
		if _, err := hash.Write([]byte(name + "[")); err != nil {
			return "", err
		}
		spec, err := json.Marshal(integration.Spec.Traits[name].Configuration)
		if err != nil {
			return "", err
		}
		trait := make(map[string]interface{})
		err = json.Unmarshal(spec, &trait)
		if err != nil {
			return "", err
		}
		for _, prop := range util.SortedMapKeys(trait) {
			val := trait[prop]
			if _, err := hash.Write([]byte(fmt.Sprintf("%s=%v,", prop, val))); err != nil {
				return "", err
			}
		}
		if _, err := hash.Write([]byte("]")); err != nil {
			return "", err
		}
	}
	// Integration traits as annotations
	for _, k := range sortedTraitAnnotationsKeys(integration) {
		v := integration.Annotations[k]
		if _, err := hash.Write([]byte(fmt.Sprintf("%s=%v,", k, v))); err != nil {
			return "", err
		}
	}

	// Add a letter at the beginning and use URL safe encoding
	digest := "v" + base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
	return digest, nil
}

// ComputeForIntegrationKit a digest of the fields that are relevant for the deployment
// Produces a digest that can be used as docker image tag
func ComputeForIntegrationKit(kit *v1.IntegrationKit) (string, error) {
	hash := sha256.New()
	// Kit version is relevant
	if _, err := hash.Write([]byte(kit.Status.Version)); err != nil {
		return "", err
	}

	for _, item := range kit.Spec.Dependencies {
		if _, err := hash.Write([]byte(item)); err != nil {
			return "", err
		}
	}
	for _, item := range kit.Spec.Configuration {
		if _, err := hash.Write([]byte(item.String())); err != nil {
			return "", err
		}
	}

	// Add a letter at the beginning and use URL safe encoding
	digest := "v" + base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
	return digest, nil
}

// ComputeForResource returns a digest for the specific resource
func ComputeForResource(res v1.ResourceSpec) (string, error) {
	hash := sha256.New()
	// Operator version is relevant
	if _, err := hash.Write([]byte(defaults.Version)); err != nil {
		return "", err
	}

	if _, err := hash.Write([]byte(res.Content)); err != nil {
		return "", err
	}
	if _, err := hash.Write([]byte(res.Name)); err != nil {
		return "", err
	}
	if _, err := hash.Write([]byte(res.Type)); err != nil {
		return "", err
	}
	if _, err := hash.Write([]byte(res.ContentKey)); err != nil {
		return "", err
	}
	if _, err := hash.Write([]byte(res.ContentRef)); err != nil {
		return "", err
	}
	if _, err := hash.Write([]byte(res.MountPath)); err != nil {
		return "", err
	}
	if _, err := hash.Write([]byte(strconv.FormatBool(res.Compression))); err != nil {
		return "", err
	}

	// Add a letter at the beginning and use URL safe encoding
	digest := "v" + base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
	return digest, nil
}

// ComputeForSource returns a digest for the specific source
func ComputeForSource(s v1.SourceSpec) (string, error) {
	hash := sha256.New()
	// Operator version is relevant
	if _, err := hash.Write([]byte(defaults.Version)); err != nil {
		return "", err
	}

	if _, err := hash.Write([]byte(s.Content)); err != nil {
		return "", err
	}
	if _, err := hash.Write([]byte(s.Name)); err != nil {
		return "", err
	}
	if _, err := hash.Write([]byte(s.Type)); err != nil {
		return "", err
	}
	if _, err := hash.Write([]byte(s.Language)); err != nil {
		return "", err
	}
	if _, err := hash.Write([]byte(s.ContentKey)); err != nil {
		return "", err
	}
	if _, err := hash.Write([]byte(s.ContentRef)); err != nil {
		return "", err
	}
	if _, err := hash.Write([]byte(s.Loader)); err != nil {
		return "", err
	}
	for _, i := range s.Interceptors {
		if _, err := hash.Write([]byte(i)); err != nil {
			return "", err
		}
	}

	if _, err := hash.Write([]byte(strconv.FormatBool(s.Compression))); err != nil {
		return "", err
	}

	// Add a letter at the beginning and use URL safe encoding
	digest := "v" + base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
	return digest, nil
}

func sortedTraitSpecMapKeys(m map[string]v1.TraitSpec) []string {
	res := make([]string, len(m))
	i := 0
	for k := range m {
		res[i] = k
		i++
	}
	sort.Strings(res)
	return res
}

func sortedTraitAnnotationsKeys(it *v1.Integration) []string {
	res := make([]string, 0, len(it.Annotations))
	for k := range it.Annotations {
		if strings.HasPrefix(k, v1.TraitAnnotationPrefix) {
			res = append(res, k)
		}
	}
	sort.Strings(res)
	return res
}

func ComputeSHA1(elem ...string) (string, error) {
	file := path.Join(elem...)

	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// nolint: gosec
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}
