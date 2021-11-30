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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// ApplyConfigOption will set the proper --config option behavior to the IntegrationSpec.
func ApplyConfigOption(ctx context.Context, config *resource.Config, integration *v1.Integration, c client.Client,
	namespace string, enableCompression bool) (*corev1.ConfigMap, error) {
	// A config option cannot specify destination path
	if config.DestinationPath() != "" {
		return nil, fmt.Errorf("cannot specify a destination path for this option type")
	}
	return applyOption(ctx, config, integration, c, namespace, enableCompression, v1.ResourceTypeConfig)
}

// ApplyResourceOption will set the proper --resource option behavior to the IntegrationSpec.
func ApplyResourceOption(ctx context.Context, config *resource.Config, integration *v1.Integration, c client.Client,
	namespace string, enableCompression bool) (*corev1.ConfigMap, error) {
	return applyOption(ctx, config, integration, c, namespace, enableCompression, v1.ResourceTypeData)
}

func applyOption(ctx context.Context, config *resource.Config, integration *v1.Integration,
	c client.Client, namespace string, enableCompression bool, resourceType v1.ResourceType) (*corev1.ConfigMap, error) {
	var maybeGenCm *corev1.ConfigMap
	switch config.StorageType() {
	case resource.StorageTypeConfigmap:
		cm := kubernetes.LookupConfigmap(ctx, c, namespace, config.Name())
		if cm == nil {
			fmt.Printf("Warn: %s Configmap not found in %s namespace, make sure to provide it before the Integration can run\n",
				config.Name(), namespace)
		} else if resourceType != v1.ResourceTypeData && cm.BinaryData != nil {
			return maybeGenCm, fmt.Errorf("you cannot provide a binary config, use a text file instead")
		}
	case resource.StorageTypeSecret:
		secret := kubernetes.LookupSecret(ctx, c, namespace, config.Name())
		if secret == nil {
			fmt.Printf("Warn: %s Secret not found in %s namespace, make sure to provide it before the Integration can run\n",
				config.Name(), namespace)
		}
	case resource.StorageTypeFile:
		// Don't allow a binary non compressed resource
		rawData, contentType, err := loadRawContent(ctx, config.Name())
		if err != nil {
			return maybeGenCm, err
		}
		if resourceType != v1.ResourceTypeData && !enableCompression && isBinary(contentType) {
			return maybeGenCm, fmt.Errorf("you cannot provide a binary config, use a text file or check --resource flag instead")
		}
		resourceSpec, err := binaryOrTextResource(path.Base(config.Name()), rawData, contentType, enableCompression, resourceType, config.DestinationPath())
		if err != nil {
			return maybeGenCm, err
		}
		maybeGenCm, err = resource.ConvertFileToConfigmap(ctx, c, resourceSpec, config, integration.Namespace, integration.Name, resourceType)
		if err != nil {
			return maybeGenCm, err
		}
	default:
		// Should never reach this
		return maybeGenCm, fmt.Errorf("invalid option type %s", config.StorageType())
	}

	integration.Spec.AddConfigurationAsResource(string(config.StorageType()), config.Name(), string(resourceType), config.DestinationPath(), config.Key())

	return maybeGenCm, nil
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

func bindGeneratedConfigmapsToIntegration(ctx context.Context, c client.Client, i *v1.Integration, configmaps []*corev1.ConfigMap) error {
	controller := true
	blockOwnerDeletion := true
	for _, cm := range configmaps {
		cm.ObjectMeta.Labels[v1.IntegrationLabel] = i.Name
		cm.ObjectMeta.Labels["camel.apache.org/autogenerated"] = "true"
		// set owner references
		cm.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
			{
				Kind:               v1.IntegrationKind,
				APIVersion:         v1.SchemeGroupVersion.String(),
				Name:               i.Name,
				UID:                i.UID,
				Controller:         &controller,
				BlockOwnerDeletion: &blockOwnerDeletion,
			},
		}
		err := c.Update(ctx, cm)
		if err != nil {
			return err
		}
	}

	return nil
}
