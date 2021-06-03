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
	"fmt"
	"path"
	"regexp"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

var invalidPaths = []string{"/etc/camel", "/deployments/dependencies"}

// RunConfigOption represents a config option
type RunConfigOption struct {
	ConfigType      configOptionType
	Value           string
	destinationPath string
}

// DestinationPath is the location where the resource will be stored on destination
func (runConfigOption *RunConfigOption) DestinationPath() string {
	return runConfigOption.destinationPath
}

// Validate checks if the DestinationPath exists and in case if it's a valid path
func (runConfigOption *RunConfigOption) Validate() error {
	if runConfigOption.destinationPath == "" {
		return nil
	}
	for _, invalidPath := range invalidPaths {
		if runConfigOption.destinationPath == invalidPath || strings.HasPrefix(runConfigOption.destinationPath, invalidPath+"/") {
			return fmt.Errorf("you cannot mount a file under %s path", invalidPath)
		}
	}
	return nil
}

type configOptionType string

const (
	// ConfigOptionTypeConfigmap --
	ConfigOptionTypeConfigmap configOptionType = "configmap"
	// ConfigOptionTypeSecret --
	ConfigOptionTypeSecret configOptionType = "secret"
	// ConfigOptionTypeFile --
	ConfigOptionTypeFile configOptionType = "file"
)

var validConfigRegexp = regexp.MustCompile(`^(configmap|secret|file)\:([\w\.\-\_\:\/@]+)$`)

func newRunConfigOption(configType configOptionType, value string) *RunConfigOption {
	optionValue, maybeDestinationPath := parseFileValue(value)
	return &RunConfigOption{
		ConfigType:      configType,
		Value:           optionValue,
		destinationPath: maybeDestinationPath,
	}
}

func parseFileValue(value string) (string, string) {
	split := strings.SplitN(value, "@", 2)
	if len(split) == 2 {
		return split[0], split[1]
	}
	return value, ""
}

// ParseResourceOption will parse and return a runConfigOption
func ParseResourceOption(item string) (*RunConfigOption, error) {
	// Deprecated: ensure backward compatibility with `--resource filename` format until version 1.5.x
	// then replace with parseOption() func directly
	option, err := parseOption(item)
	if err != nil {
		if strings.HasPrefix(err.Error(), "could not match configuration") {
			fmt.Printf("Warn: --resource %s has been deprecated. You should use --resource file:%s instead.\n", item, item)
			return parseOption("file:" + item)
		}
		return nil, err
	}
	return option, nil
}

// ParseConfigOption will parse and return a runConfigOption
func ParseConfigOption(item string) (*RunConfigOption, error) {
	return parseOption(item)
}

func parseOption(item string) (*RunConfigOption, error) {
	if !validConfigRegexp.MatchString(item) {
		return nil, fmt.Errorf("could not match configuration %s, must match %v regular expression", item, validConfigRegexp)
	}
	// Parse the regexp groups
	groups := validConfigRegexp.FindStringSubmatch(item)
	var cot configOptionType
	switch groups[1] {
	case "configmap":
		cot = ConfigOptionTypeConfigmap
	case "secret":
		cot = ConfigOptionTypeSecret
	case "file":
		cot = ConfigOptionTypeFile
	default:
		// Should never reach this
		return nil, fmt.Errorf("invalid config option type %s", groups[1])
	}
	configurationOption := newRunConfigOption(cot, groups[2])
	if err := configurationOption.Validate(); err != nil {
		return nil, err
	}
	return configurationOption, nil
}

func applyOption(config *RunConfigOption, integrationSpec *v1.IntegrationSpec,
	c client.Client, namespace string, enableCompression bool, resourceType v1.ResourceType) error {
	switch config.ConfigType {
	case ConfigOptionTypeConfigmap:
		cm := kubernetes.LookupConfigmap(context.Background(), c, namespace, config.Value)
		if cm == nil {
			fmt.Printf("Warn: %s Configmap not found in %s namespace, make sure to provide it before the Integration can run\n",
				config.Value, namespace)
		} else if resourceType != v1.ResourceTypeData && cm.BinaryData != nil {
			return fmt.Errorf("you cannot provide a binary config, use a text file instead")
		}
		integrationSpec.AddConfigurationAsResource(string(config.ConfigType), config.Value, string(resourceType), config.DestinationPath())
	case ConfigOptionTypeSecret:
		secret := kubernetes.LookupSecret(context.Background(), c, namespace, config.Value)
		if secret == nil {
			fmt.Printf("Warn: %s Secret not found in %s namespace, make sure to provide it before the Integration can run\n",
				config.Value, namespace)
		}
		integrationSpec.AddConfigurationAsResource(string(config.ConfigType), config.Value, string(resourceType), config.DestinationPath())
	case ConfigOptionTypeFile:
		// Don't allow a file size longer than 1 MiB
		fileSize, err := fileSize(config.Value)
		printSize := fmt.Sprintf("%.2f", float64(fileSize)/Megabyte)
		if err != nil {
			return err
		} else if fileSize > Megabyte {
			return fmt.Errorf("you cannot provide a file larger than 1 MB (it was %s MB), check configmap option or --volume instead", printSize)
		}
		// Don't allow a binary non compressed resource
		rawData, contentType, err := loadRawContent(config.Value)
		if err != nil {
			return err
		}
		if resourceType != v1.ResourceTypeData && !enableCompression && isBinary(contentType) {
			return fmt.Errorf("you cannot provide a binary config, use a text file or check --resource flag instead")
		}
		resourceSpec, err := binaryOrTextResource(path.Base(config.Value), rawData, contentType, enableCompression, resourceType, config.DestinationPath())
		if err != nil {
			return err
		}
		integrationSpec.AddResources(resourceSpec)
	default:
		// Should never reach this
		return fmt.Errorf("invalid option type %s", config.ConfigType)
	}

	return nil
}

// ApplyConfigOption will set the proper --config option behavior to the IntegrationSpec
func ApplyConfigOption(config *RunConfigOption, integrationSpec *v1.IntegrationSpec, c client.Client, namespace string, enableCompression bool) error {
	return applyOption(config, integrationSpec, c, namespace, enableCompression, v1.ResourceTypeConfig)
}

// ApplyResourceOption will set the proper --resource option behavior to the IntegrationSpec
func ApplyResourceOption(config *RunConfigOption, integrationSpec *v1.IntegrationSpec, c client.Client, namespace string, enableCompression bool) error {
	return applyOption(config, integrationSpec, c, namespace, enableCompression, v1.ResourceTypeData)
}

func binaryOrTextResource(fileName string, data []byte, contentType string, base64Compression bool, resourceType v1.ResourceType, destinationPath string) (v1.ResourceSpec, error) {
	resourceSpec := v1.ResourceSpec{
		DataSpec: v1.DataSpec{
			Name:        fileName,
			Path:        destinationPath,
			ContentKey:  fileName,
			ContentType: contentType,
			Compression: false,
		},
		Type: resourceType,
	}

	if !base64Compression && isBinary(contentType) {
		resourceSpec.RawContent = data
		return resourceSpec, nil
	}
	// either is a text resource or base64 compression is enabled
	if base64Compression {
		content, err := compressToString(data)
		if err != nil {
			return resourceSpec, err
		}
		resourceSpec.Content = content
		resourceSpec.Compression = true
	} else {
		resourceSpec.Content = string(data)
	}
	return resourceSpec, nil
}

func filterFileLocation(maybeFileLocations []string) []string {
	filteredOptions := make([]string, 0)
	for _, option := range maybeFileLocations {
		if strings.HasPrefix(option, "file:") {
			localPath, _ := parseFileValue(strings.Replace(option, "file:", "", 1))
			filteredOptions = append(filteredOptions, localPath)
		}
	}
	return filteredOptions
}
