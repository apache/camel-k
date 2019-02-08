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
	"sync"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewRuntime --
func NewRuntime() Runtime {
	return Runtime{
		catalogs: make(map[string]RuntimeCatalog),
	}
}

// Runtime --
type Runtime struct {
	catalogs map[string]RuntimeCatalog
	lock     sync.Mutex
}

// LoadCatalog --
func (r *Runtime) LoadCatalog(ctx context.Context, client client.Client, namespace string, version string) (*RuntimeCatalog, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if c, ok := r.catalogs[version]; ok {
		return &c, nil
	}

	var c *RuntimeCatalog
	var err error

	// try with the exact match
	c, err = r.doLoadCatalog(ctx, client, namespace, version)
	if err != nil {
		return nil, err
	}
	if c != nil {
		r.catalogs[version] = *c
		return c, nil
	}

	return nil, fmt.Errorf("unable to find a camel catalog for version: %s", version)
}

func (r *Runtime) doLoadCatalog(ctx context.Context, client client.Client, namespace string, version string) (*RuntimeCatalog, error) {
	list := v1alpha1.NewCamelCatalogList()
	err := client.List(ctx, &k8sclient.ListOptions{Namespace: namespace}, &list)
	if err != nil {
		return nil, err
	}

	for _, c := range list.Items {
		if c.Spec.Version == version {
			return NewRuntimeCatalog(c.Spec), nil
		}
	}

	return nil, nil
}
