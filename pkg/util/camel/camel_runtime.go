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

package camel

import (
	"context"
	"fmt"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
)

// LoadCatalog --
func LoadCatalog(ctx context.Context, client client.Client, namespace string, camelVersion string, runtimeVersion string, provider interface{}) (*RuntimeCatalog, error) {
	options := []k8sclient.ListOption{
		k8sclient.InNamespace(namespace),
	}

	if _, ok := provider.(v1alpha1.QuarkusRuntimeProvider); ok {
		options = append(options, k8sclient.MatchingLabels{
			"camel.apache.org/runtime.provider": "quarkus",
		})
	}

	list := v1alpha1.NewCamelCatalogList()
	err := client.List(ctx, &list, options...)
	if err != nil {
		return nil, err
	}

	catalog, err := findBestMatch(list.Items, camelVersion, runtimeVersion, provider)
	if err != nil {
		return nil, err
	}

	if catalog != nil {
		return catalog, nil
	}

	switch provider := provider.(type) {
	case v1alpha1.QuarkusRuntimeProvider:
		return nil, fmt.Errorf("unable to find catalog matching version requirement: camel=%s, runtime=%s, camel-quarkus=%s, quarkus=%s",
			camelVersion, runtimeVersion, provider.CamelQuarkusVersion, provider.QuarkusVersion)
	default:
		return nil, fmt.Errorf("unable to find catalog matching version requirement: camel=%s, runtime=%s",
			camelVersion, runtimeVersion)
	}
}
