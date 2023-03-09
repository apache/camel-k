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

package bindings

import (
	"encoding/json"
	"net/url"

	knativeapis "github.com/apache/camel-k/pkg/apis/camel/v1/knative"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

func asEndpointProperties(props map[string]string) *v1alpha1.EndpointProperties {
	serialized, err := json.Marshal(props)
	if err != nil {
		panic(err)
	}
	return &v1alpha1.EndpointProperties{
		RawMessage: serialized,
	}
}

func asKnativeConfig(endpointURL string) string {
	serviceURL, err := url.Parse(endpointURL)
	if err != nil {
		panic(err)
	}
	def, err := knativeapis.BuildCamelServiceDefinition("sink", knativeapis.CamelEndpointKindSink, knativeapis.CamelServiceTypeEndpoint, *serviceURL, "", "")
	if err != nil {
		panic(err)
	}
	env := knativeapis.NewCamelEnvironment()
	env.Services = append(env.Services, def)
	serialized, err := json.Marshal(env)
	if err != nil {
		panic(err)
	}
	return string(serialized)
}
