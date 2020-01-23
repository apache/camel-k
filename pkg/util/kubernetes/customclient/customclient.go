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

package customclient

import (
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// GetClientFor returns a RESTClient for the given group and version
func GetClientFor(c kubernetes.Interface, group string, version string) (*rest.RESTClient, error) {
	inConfig, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	conf := rest.CopyConfig(inConfig)
	conf.GroupVersion = &schema.GroupVersion{
		Group:   group,
		Version: version,
	}
	conf.APIPath = "/apis"
	conf.AcceptContentTypes = "application/json"
	conf.ContentType = "application/json"

	// this gets used for discovery and error handling types
	conf.NegotiatedSerializer = basicNegotiatedSerializer{}
	if conf.UserAgent == "" {
		conf.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return rest.RESTClientFor(conf)
}

// GetDynamicClientFor returns a dynamic client for a given kind
func GetDynamicClientFor(group string, version string, kind string, namespace string) (dynamic.ResourceInterface, error) {
	conf, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(conf)
	if err != nil {
		return nil, err
	}
	return dynamicClient.Resource(schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: kind,
	}).Namespace(namespace), nil
}

// GetDefaultDynamicClientFor returns a dynamic client for a given kind
func GetDefaultDynamicClientFor(kind string, namespace string) (dynamic.ResourceInterface, error) {
	conf, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(conf)
	if err != nil {
		return nil, err
	}
	return dynamicClient.Resource(schema.GroupVersionResource{
		Group:    v1.SchemeGroupVersion.Group,
		Version:  v1.SchemeGroupVersion.Version,
		Resource: kind,
	}).Namespace(namespace), nil
}
