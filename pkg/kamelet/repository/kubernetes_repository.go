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

	"github.com/apache/camel-k/v2/pkg/client"
	kameletsv1 "github.com/apache/camel-kamelets/crds/pkg/apis/camel/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type kubernetesKameletRepository struct {
	c         client.Client
	namespace string
}

func newKubernetesKameletRepository(c client.Client, namespace string) KameletRepository {
	return &kubernetesKameletRepository{
		c:         c,
		namespace: namespace,
	}
}

// Enforce type.
var _ KameletRepository = &kubernetesKameletRepository{}

func (c *kubernetesKameletRepository) List(ctx context.Context) ([]string, error) {
	list, err := c.c.KameletsCamelV1().Kamelets(c.namespace).List(ctx, v1.ListOptions{})
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

func (c *kubernetesKameletRepository) Get(ctx context.Context, name string) (*kameletsv1.Kamelet, error) {
	kamelet, err := c.c.KameletsCamelV1().Kamelets(c.namespace).Get(ctx, name, v1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		// return nil if not found, so other repositories can try to find it
		return nil, nil
	}
	return kamelet, err
}

func (c *kubernetesKameletRepository) String() string {
	return fmt.Sprintf("Kubernetes[namespace=%s]", c.namespace)
}
