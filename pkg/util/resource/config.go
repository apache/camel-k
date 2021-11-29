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

package resource

import (
	"context"
	"crypto/sha1" //nolint
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

var invalidPaths = []string{"/etc/camel", "/deployments/dependencies"}

// Config represents a config option.
type Config struct {
	storageType     StorageType
	contentType     ContentType
	resourceName    string
	resourceKey     string
	destinationPath string
}

// DestinationPath is the location where the resource will be stored on destination.
func (config *Config) DestinationPath() string {
	return config.destinationPath
}

// StorageType is the type of storage used for the configuration.
func (config *Config) StorageType() StorageType {
	return config.storageType
}

// ContentType is the type of content used for the configuration.
func (config *Config) ContentType() ContentType {
	return config.contentType
}

// Name is the name of the resource.
func (config *Config) Name() string {
	return config.resourceName
}

// Key is the key specified for the resource.
func (config *Config) Key() string {
	return config.resourceKey
}

// Validate checks if the DestinationPath is correctly configured.
func (config *Config) Validate() error {
	if config.destinationPath == "" {
		return nil
	}

	// Check for invalid path
	for _, invalidPath := range invalidPaths {
		if config.destinationPath == invalidPath || strings.HasPrefix(config.destinationPath, invalidPath+"/") {
			return fmt.Errorf("you cannot mount a file under %s path", invalidPath)
		}
	}
	return nil
}

// StorageType represent a resource/config type such as configmap, secret or local file.
type StorageType string

const (
	// StorageTypeConfigmap --.
	StorageTypeConfigmap StorageType = "configmap"
	// StorageTypeSecret --.
	StorageTypeSecret StorageType = "secret"
	// StorageTypeFile --.
	StorageTypeFile StorageType = "file"
)

// ContentType represent what kind of a content is, either data or purely text configuration.
type ContentType string

const (
	// ContentTypeData can contain binary content, won't be parsed to look for user properties.
	ContentTypeData ContentType = "data"
	// ContentTypeText can't contain binary content, will be parsed to look for user properties.
	ContentTypeText ContentType = "text"
)

var (
	validConfigSecretRegexp = regexp.MustCompile(`^(configmap|secret)\:([\w\.\-\_\:\/@]+)$`)
	validFileRegexp         = regexp.MustCompile(`^file\:([\w\.\-\_\:\/@" ]+)$`)
	validResourceRegexp     = regexp.MustCompile(`^([\w\.\-\_\:]+)(\/([\w\.\-\_\:]+))?(\@([\w\.\-\_\:\/]+))?$`)
)

func newConfig(storageType StorageType, contentType ContentType, value string) *Config {
	rn, mk, mp := parseResourceValue(storageType, value)
	return &Config{
		storageType:     storageType,
		contentType:     contentType,
		resourceName:    rn,
		resourceKey:     mk,
		destinationPath: mp,
	}
}

func parseResourceValue(storageType StorageType, value string) (resource string, maybeKey string, maybeDestinationPath string) {
	if storageType == StorageTypeFile {
		resource, maybeDestinationPath = ParseFileValue(value)
		return resource, "", maybeDestinationPath
	}

	return parseCMOrSecretValue(value)
}

// ParseFileValue will parse a file resource/config option to return the local path and the
// destination path expected.
func ParseFileValue(value string) (localPath string, maybeDestinationPath string) {
	split := strings.SplitN(value, "@", 2)
	if len(split) == 2 {
		return split[0], split[1]
	}

	return value, ""
}

func parseCMOrSecretValue(value string) (resource string, maybeKey string, maybeDestinationPath string) {
	if !validResourceRegexp.MatchString(value) {
		return value, "", ""
	}
	// Must have 3 values
	groups := validResourceRegexp.FindStringSubmatch(value)

	return groups[1], groups[3], groups[5]
}

// ParseResource will parse and return a Config.
func ParseResource(item string) (*Config, error) {
	// Deprecated: ensure backward compatibility with `--resource filename` format until version 1.5.x
	// then replace with parse() func directly
	resource, err := parse(item, ContentTypeData)
	if err != nil {
		if strings.HasPrefix(err.Error(), "could not match config, secret or file configuration") {
			fmt.Printf("Warn: --resource %s has been deprecated. You should use --resource file:%s instead.\n", item, item)
			return parse("file:"+item, ContentTypeData)
		}
		return nil, err
	}

	return resource, nil
}

// ParseConfig will parse and return a Config.
func ParseConfig(item string) (*Config, error) {
	return parse(item, ContentTypeText)
}

func parse(item string, contentType ContentType) (*Config, error) {
	var cot StorageType
	var value string
	switch {
	case validConfigSecretRegexp.MatchString(item):
		// parse as secret/configmap
		groups := validConfigSecretRegexp.FindStringSubmatch(item)
		switch groups[1] {
		case "configmap":
			cot = StorageTypeConfigmap
		case "secret":
			cot = StorageTypeSecret
		}
		value = groups[2]
	case validFileRegexp.MatchString(item):
		// parse as file
		groups := validFileRegexp.FindStringSubmatch(item)
		cot = StorageTypeFile
		value = groups[1]
	default:
		return nil, fmt.Errorf("could not match config, secret or file configuration as %s", item)
	}

	configurationOption := newConfig(cot, contentType, value)
	if err := configurationOption.Validate(); err != nil {
		return nil, err
	}
	return configurationOption, nil
}

// ConvertFileToConfigmap convert a local file resource type in a configmap type
// taking care to create the Configmap on the cluster. The method will change the value of config parameter
// to reflect the conversion applied transparently.
func ConvertFileToConfigmap(ctx context.Context, c client.Client, resourceSpec v1.ResourceSpec, config *Config,
	namespace string, resourceType v1.ResourceType) (*corev1.ConfigMap, error) {
	if config.DestinationPath() == "" {
		config.resourceKey = filepath.Base(config.Name())
		// As we are changing the resource to a configmap type
		// we need to declare the mount path not to use the
		// default behavior of a configmap (which include a subdirectory with the configmap name)
		if resourceType == v1.ResourceTypeData {
			config.destinationPath = camel.ResourcesDefaultMountPath
		} else {
			config.destinationPath = camel.ConfigResourcesMountPath
		}
	} else {
		config.resourceKey = filepath.Base(config.DestinationPath())
		config.destinationPath = filepath.Dir(config.DestinationPath())
	}
	genCmName := fmt.Sprintf("cm-%s", hashFrom([]byte(resourceSpec.Content), resourceSpec.RawContent))
	cm := kubernetes.NewConfigmap(namespace, genCmName, filepath.Base(config.Name()), config.Key(), resourceSpec.Content, resourceSpec.RawContent)
	err := c.Create(ctx, cm)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// We'll reuse it, as is
		} else {
			return cm, err
		}
	}
	config.storageType = StorageTypeConfigmap
	config.resourceName = cm.Name

	return cm, nil
}

//nolint
func hashFrom(contents ...[]byte) string {
	// SHA1 because we need to limit the length to less than 64 chars
	hash := sha1.New()
	for _, c := range contents {
		hash.Write(c)
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}
