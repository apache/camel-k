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
	"crypto/sha1" //nolint
	"fmt"
	"path"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/resource"
	"github.com/magiconair/properties"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

//nolint
func hashFrom(contents ...[]byte) string {
	// SHA1 because we need to limit the length to less than 64 chars
	hash := sha1.New()
	for _, c := range contents {
		hash.Write(c)
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}

func parseConfigAndGenCm(ctx context.Context, cmd *cobra.Command, c client.Client, config *resource.Config, integration *v1.Integration, enableCompression bool) (*corev1.ConfigMap, error) {
	switch config.StorageType() {
	case resource.StorageTypeConfigmap:
		cm := kubernetes.LookupConfigmap(ctx, c, integration.Namespace, config.Name())
		if cm == nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warn: %s Configmap not found in %s namespace, make sure to provide it before the Integration can run\n",
				config.Name(), integration.Namespace)
		} else if config.ContentType() != resource.ContentTypeData && cm.BinaryData != nil {
			return nil, fmt.Errorf("you cannot provide a binary config, use a text file instead")
		}
	case resource.StorageTypeSecret:
		secret := kubernetes.LookupSecret(ctx, c, integration.Namespace, config.Name())
		if secret == nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warn: %s Secret not found in %s namespace, make sure to provide it before the Integration can run\n",
				config.Name(), integration.Namespace)
		}
	case resource.StorageTypeFile:
		// Don't allow a binary non compressed resource
		rawData, contentType, err := loadRawContent(ctx, config.Name())
		if err != nil {
			return nil, err
		}
		if config.ContentType() != resource.ContentTypeData && !enableCompression && isBinary(contentType) {
			return nil, fmt.Errorf("you cannot provide a binary config, use a text file or check --resource flag instead")
		}
		resourceType := v1.ResourceTypeData
		if config.ContentType() == resource.ContentTypeText {
			resourceType = v1.ResourceTypeConfig
		}
		resourceSpec, err := binaryOrTextResource(path.Base(config.Name()), rawData, contentType, enableCompression, resourceType, config.DestinationPath())
		if err != nil {
			return nil, err
		}

		return resource.ConvertFileToConfigmap(ctx, c, config, integration.Namespace, integration.Name, resourceSpec.Content, resourceSpec.RawContent)
	default:
		// Should never reach this
		return nil, fmt.Errorf("invalid option type %s", config.StorageType())
	}

	return nil, nil
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
			localPath, _ := resource.ParseFileValue(strings.Replace(option, "file:", "", 1))
			filteredOptions = append(filteredOptions, localPath)
		}
	}
	return filteredOptions
}

func mergePropertiesWithPrecedence(items []string) (*properties.Properties, error) {
	loPrecedenceProps := properties.NewProperties()
	hiPrecedenceProps := properties.NewProperties()
	for _, item := range items {
		prop, err := extractProperties(item)
		if err != nil {
			return nil, err
		}
		// We consider file props to have a lower priority versus single properties
		if strings.HasPrefix(item, "file:") {
			loPrecedenceProps.Merge(prop)
		} else {
			hiPrecedenceProps.Merge(prop)
		}
	}
	// Any property contained in both collections will be merged
	// giving precedence to the ones in hiPrecedenceProps
	loPrecedenceProps.Merge(hiPrecedenceProps)
	return loPrecedenceProps, nil
}

// The function parse the value and if it is a file (file:/path/), it will parse as property file
// otherwise return a single property built from the item passed as `key=value`.
func extractProperties(value string) (*properties.Properties, error) {
	if !strings.HasPrefix(value, "file:") {
		return keyValueProps(value)
	}
	// we already validated the existence of files during validate()
	return loadPropertyFile(strings.Replace(value, "file:", "", 1))
}

func keyValueProps(value string) (*properties.Properties, error) {
	return properties.Load([]byte(value), properties.UTF8)
}
