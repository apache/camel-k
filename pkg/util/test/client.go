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

package test

import (
	"strings"

	"github.com/apache/camel-k/pkg/apis"
	"github.com/apache/camel-k/pkg/client"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	clientscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	controller "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// NewFakeClient ---
func NewFakeClient(initObjs ...runtime.Object) (client.Client, error) {
	scheme := clientscheme.Scheme

	// Setup Scheme for all resources
	if err := apis.AddToScheme(scheme); err != nil {
		return nil, err
	}

	c := fake.NewFakeClientWithScheme(scheme, initObjs...)
	filtered := make([]runtime.Object, 0, len(initObjs))
	skipList := []string{"camel", "knative"}
	for _, o := range initObjs {
		kinds, _, _ := scheme.ObjectKinds(o)
		allow := true
		for _, k := range kinds {
			for _, skip := range skipList {
				if strings.Contains(k.Group, skip) {
					allow = false
					break
				}
			}
		}
		if allow {
			filtered = append(filtered, o)
		}
	}
	clientset := fakeclientset.NewSimpleClientset(filtered...)

	return &FakeClient{
		Client:    c,
		Interface: clientset,
	}, nil
}

// FakeClient ---
type FakeClient struct {
	controller.Client
	kubernetes.Interface
}

// GetScheme ---
func (c *FakeClient) GetScheme() *runtime.Scheme {
	return clientscheme.Scheme
}

func (c *FakeClient) GetConfig() *rest.Config {
	return nil
}

func (c *FakeClient) GetCurrentNamespace(kubeConfig string) (string, error) {
	return "", nil
}
