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

package repository

import (
	"context"
	"fmt"
	"sort"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client/camel/clientset/versioned"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type kubernetesKameletRepository struct {
	client    versioned.Interface
	namespace string
}

func newKubernetesKameletRepository(client versioned.Interface, namespace string) KameletRepository {
	return &kubernetesKameletRepository{
		client:    client,
		namespace: namespace,
	}
}

// Enforce type
var _ KameletRepository = &kubernetesKameletRepository{}

func (c *kubernetesKameletRepository) List(ctx context.Context) ([]string, error) {
	list, err := c.client.CamelV1alpha1().Kamelets(c.namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	res := make([]string, 0, len(list.Items))
	for _, item := range list.Items {
		res = append(res, item.Name)
	}
	sort.Strings(res)
	return res, nil
}

func (c *kubernetesKameletRepository) Get(ctx context.Context, name string) (*v1alpha1.Kamelet, error) {
	kamelet, err := c.client.CamelV1alpha1().Kamelets(c.namespace).Get(ctx, name, v1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		// return nil if not found, so other repositories can try to find it
		return nil, nil
	}
	return kamelet, err
}

func (c *kubernetesKameletRepository) String() string {
	return fmt.Sprintf("Kubernetes[namespace=%s]", c.namespace)
}
