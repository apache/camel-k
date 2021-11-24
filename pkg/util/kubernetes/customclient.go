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

package kubernetes

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"

	"github.com/apache/camel-k/pkg/client"
)

// GetClientFor returns a RESTClient for the given group and version.
func GetClientFor(c client.Client, group string, version string) (*rest.RESTClient, error) {
	conf := rest.CopyConfig(c.GetConfig())
	conf.NegotiatedSerializer = serializer.NewCodecFactory(c.GetScheme())
	conf.UserAgent = rest.DefaultKubernetesUserAgent()
	conf.APIPath = "/apis"
	conf.GroupVersion = &schema.GroupVersion{
		Group:   group,
		Version: version,
	}
	return rest.RESTClientFor(conf)
}
