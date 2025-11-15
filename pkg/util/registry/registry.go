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

package registry

import (
	"context"
	"os"
	"path/filepath"

	"github.com/apache/camel-k/v2/pkg/util/io"

	"github.com/apache/camel-k/v2/pkg/client"
	"go.uber.org/multierr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MountSecretRegistryConfig write a file containing the secret registry config in a temporary folder.
func MountSecretRegistryConfig(ctx context.Context, c client.Client, namespace, prefix, name string) (string, error) {
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return "", err
	}

	secret, err := c.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if removeErr := os.RemoveAll(dir); removeErr != nil {
			err = multierr.Append(err, removeErr)
		}

		return "", err
	}

	for file, content := range secret.Data {
		if err := os.WriteFile(filepath.Join(dir, remap(file)), content, io.FilePerm600); err != nil {
			if removeErr := os.RemoveAll(dir); removeErr != nil {
				err = multierr.Append(err, removeErr)
			}

			return "", err
		}
	}

	return dir, nil
}

func remap(name string) string {
	if name == ".dockerconfigjson" {
		return "config.json"
	}

	return name
}
