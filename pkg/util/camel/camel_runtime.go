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
	"strings"

	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/maven"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
)

// CreateCatalog --.
func CreateCatalog(ctx context.Context, client client.Client, namespace string, platform *v1.IntegrationPlatform, runtime v1.RuntimeSpec) (*RuntimeCatalog, error) {
	ctx, cancel := context.WithTimeout(ctx, platform.Status.Build.GetTimeout().Duration)
	defer cancel()
	catalog, err := GenerateCatalog(ctx, client, namespace, platform.Status.Build.Maven, runtime, []maven.Dependency{})
	if err != nil {
		return nil, err
	}

	// sanitize catalog name
	catalogName := "camel-catalog-" + strings.ToLower(runtime.Version)

	cx := v1.NewCamelCatalogWithSpecs(namespace, catalogName, catalog.CamelCatalogSpec)
	cx.Labels = make(map[string]string)
	cx.Labels["app"] = "camel-k"
	cx.Labels["camel.apache.org/runtime.version"] = runtime.Version
	cx.Labels["camel.apache.org/runtime.provider"] = string(runtime.Provider)
	cx.Labels["camel.apache.org/catalog.generated"] = "true"

	if err := client.Create(ctx, &cx); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// It's still possible that catalog wasn't yet found at the time of loading
			// but then created in the background before the client tries to create it.
			// In this case, simply try loading again and reuse the existing catalog.
			catalog, err = LoadCatalog(ctx, client, namespace, runtime)
			if err != nil {
				// unexpected error
				return nil, fmt.Errorf("catalog %q already exists but unable to load: %w", catalogName, err)
			}
		} else {
			return nil, fmt.Errorf("unable to create catalog runtime=%s, provider=%s, name=%s: %w",
				runtime.Version,
				runtime.Provider,
				catalogName, err)

		}
	}

	// verify that the catalog has been generated
	ct, err := kubernetes.GetUnstructured(
		ctx,
		client,
		schema.GroupVersionKind{Group: "camel.apache.org", Version: "v1", Kind: "CamelCatalog"},
		catalogName,
		namespace,
	)
	if ct == nil || err != nil {
		return nil, fmt.Errorf("unable to create catalog runtime=%s, provider=%s, name=%s: %w",
			runtime.Version,
			runtime.Provider,
			catalogName, err)
	}

	return catalog, nil
}

// LoadCatalog --.
func LoadCatalog(ctx context.Context, client client.Client, namespace string, runtime v1.RuntimeSpec) (*RuntimeCatalog, error) {
	options := []k8sclient.ListOption{
		k8sclient.InNamespace(namespace),
	}

	list := v1.NewCamelCatalogList()
	err := client.List(ctx, &list, options...)
	if err != nil {
		return nil, err
	}

	catalog, err := findBestMatch(list.Items, runtime)
	if err != nil {
		return nil, err
	}

	return catalog, nil
}
