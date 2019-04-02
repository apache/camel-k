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
	"github.com/apache/camel-k/pkg/apis"
	"github.com/apache/camel-k/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	clientscheme "k8s.io/client-go/kubernetes/scheme"
	controller "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewFakeClient ---
func NewFakeClient(initObjs ...runtime.Object) (client.Client, error) {
	scheme := clientscheme.Scheme

	// Setup Scheme for all resources
	if err := apis.AddToScheme(scheme); err != nil {
		return nil, err
	}

	c := fake.NewFakeClientWithScheme(scheme, initObjs...)

	return &FakeClient{
		Client:    c,
		Interface: nil,
	}, nil

}

// FakeClient ---
type FakeClient struct {
	controller.Client
	kubernetes.Interface
}

// GetScheme ---
func (c *FakeClient) GetScheme() *runtime.Scheme {
	return nil
}
