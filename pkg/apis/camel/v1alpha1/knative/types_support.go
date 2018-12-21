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

package knative

import (
	"encoding/json"
	"net/url"
	"strconv"
)

// BuildCamelServiceDefinition creates a CamelServiceDefinition from a given URL
func BuildCamelServiceDefinition(name string, serviceType CamelServiceType, rawurl string) (*CamelServiceDefinition, error) {
	serviceURL, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	definition := CamelServiceDefinition{
		Name:        name,
		Host:        serviceURL.Host,
		Port:        -1,
		ServiceType: serviceType,
		Protocol:    CamelProtocol(serviceURL.Scheme),
		Metadata:    make(map[string]string),
	}
	portStr := serviceURL.Port()
	if portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, err
		}
		definition.Port = port
	}
	path := serviceURL.Path
	if path != "" {
		definition.Metadata[CamelMetaServicePath] = path
	} else {
		definition.Metadata[CamelMetaServicePath] = "/"
	}
	return &definition, nil
}


// Serialize serializes a CamelEnvironment
func (env CamelEnvironment) Serialize() (string, error) {
	res, err := json.Marshal(env)
	if err != nil {
		return "", err
	}
	return string(res), nil
}
