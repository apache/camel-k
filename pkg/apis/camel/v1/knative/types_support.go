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
)

// BuildCamelServiceDefinition creates a CamelServiceDefinition from a given URL
func BuildCamelServiceDefinition(name string, endpointKind CamelEndpointKind, serviceType CamelServiceType,
	serviceURL url.URL, apiVersion, kind string) (CamelServiceDefinition, error) {

	definition := CamelServiceDefinition{
		Name:        name,
		URL:         serviceURL.String(),
		ServiceType: serviceType,
		Metadata: map[string]string{
			CamelMetaEndpointKind:      string(endpointKind),
			CamelMetaKnativeAPIVersion: apiVersion,
			CamelMetaKnativeKind:       kind,
		},
	}

	return definition, nil
}

// Serialize serializes a CamelEnvironment
func (env *CamelEnvironment) Serialize() (string, error) {
	res, err := json.Marshal(env)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

// Deserialize deserializes a camel environment into this struct
func (env *CamelEnvironment) Deserialize(str string) error {
	if err := json.Unmarshal([]byte(str), env); err != nil {
		return err
	}
	return nil
}

// ContainsService tells if the environment contains a service with the given name and type
func (env *CamelEnvironment) ContainsService(name string, endpointKind CamelEndpointKind, serviceType CamelServiceType, apiVersion, kind string) bool {
	return env.FindService(name, endpointKind, serviceType, apiVersion, kind) != nil
}

// FindService --
func (env *CamelEnvironment) FindService(name string, endpointKind CamelEndpointKind, serviceType CamelServiceType, apiVersion, kind string) *CamelServiceDefinition {
	for _, svc := range env.Services {
		svc := svc
		if svc.Name == name &&
			svc.Metadata[CamelMetaEndpointKind] == string(endpointKind) &&
			svc.ServiceType == serviceType &&
			(apiVersion == "" || svc.Metadata[CamelMetaKnativeAPIVersion] == apiVersion) &&
			(kind == "" || svc.Metadata[CamelMetaKnativeKind] == kind) {
			return &svc
		}
	}
	return nil
}
