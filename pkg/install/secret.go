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

package install

import (
	"context"

	"github.com/apache/camel-k/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/registry"
)

const registrySecretName = "camel-k-registry-secret"

// RegistrySecretOrCollect generates a secret from auth settings and creates it on the cluster
// (or appends it to the collection).
func RegistrySecretOrCollect(ctx context.Context, c client.Client, namespace string, auth registry.Auth,
	collection *kubernetes.Collection, force bool) (string, error) {
	secretData, err := auth.GenerateDockerConfig()
	if err != nil {
		return "", err
	}

	return registrySecretFromDataOrCollect(ctx, c, namespace, secretData, collection, force)
}

// RegistrySecretFromFileOrCollect generates a secret from a docker-config.json file and creates it on the cluster
// (or appends it to the collection).
func RegistrySecretFromFileOrCollect(ctx context.Context, c client.Client, namespace string, file string,
	collection *kubernetes.Collection, force bool) (string, error) {
	secretData, err := util.ReadFile(file)
	if err != nil {
		return "", err
	}

	return registrySecretFromDataOrCollect(ctx, c, namespace, secretData, collection, force)
}

// registrySecretFromDataOrCollect generates a secret from a docker config file content file and creates it
// on the cluster (or appends it to the collection).
func registrySecretFromDataOrCollect(ctx context.Context, c client.Client, namespace string, secretData []byte,
	collection *kubernetes.Collection, force bool) (string, error) {
	registrySecret := v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      registrySecretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "camel-k",
			},
		},
		Type: v1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			v1.DockerConfigJsonKey: secretData,
		},
	}

	if err := ObjectOrCollect(ctx, c, namespace, collection, force, &registrySecret); err != nil {
		return "", err
	}
	return registrySecretName, nil
}
