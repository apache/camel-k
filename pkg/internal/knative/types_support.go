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
	"fmt"
	"net/url"
)

// BuildCamelServiceDefinition creates a CamelServiceDefinition from a given URL.
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
			CamelMetaKnativeName:       name,
		},
	}

	return definition, nil
}

// SetSinkBinding marks one of the service as SinkBinding.
func (env *CamelEnvironment) SetSinkBinding(name string, endpointKind CamelEndpointKind, serviceType CamelServiceType, apiVersion, kind string) {
	for i, svc := range env.Services {
		if svc.Name == name &&
			svc.Metadata[CamelMetaEndpointKind] == string(endpointKind) &&
			svc.ServiceType == serviceType &&
			(apiVersion == "" || svc.Metadata[CamelMetaKnativeAPIVersion] == apiVersion) &&
			(kind == "" || svc.Metadata[CamelMetaKnativeKind] == kind) {
			svc.SinkBinding = true
			env.Services[i] = svc
		}
	}
}

// ToCamelProperties returns the application properties representation of the services.
func (env *CamelEnvironment) ToCamelProperties() map[string]string {
	mappedServices := make(map[string]string)
	for i, service := range env.Services {
		resource := fmt.Sprintf("camel.component.knative.environment.resources[%d]", i)
		mappedServices[fmt.Sprintf("%s.name", resource)] = service.Name
		mappedServices[fmt.Sprintf("%s.type", resource)] = string(service.ServiceType)
		mappedServices[fmt.Sprintf("%s.objectKind", resource)] = service.Metadata[CamelMetaKnativeKind]
		mappedServices[fmt.Sprintf("%s.objectApiVersion", resource)] = service.Metadata[CamelMetaKnativeAPIVersion]
		mappedServices[fmt.Sprintf("%s.endpointKind", resource)] = service.Metadata[CamelMetaEndpointKind]
		mappedServices[fmt.Sprintf("%s.reply", resource)] = service.Metadata[CamelMetaKnativeReply]
		if service.ServiceType == CamelServiceTypeEvent {
			mappedServices[fmt.Sprintf("%s.objectName", resource)] = service.Metadata[CamelMetaKnativeName]
		}
		if service.SinkBinding {
			mappedServices[fmt.Sprintf("%s.url", resource)] = "${K_SINK}"
			mappedServices[fmt.Sprintf("%s.ceOverrides", resource)] = "${K_CE_OVERRIDES}"
		} else {
			mappedServices[fmt.Sprintf("%s.url", resource)] = service.URL
			mappedServices[fmt.Sprintf("%s.path", resource)] = service.Path
		}
	}

	return mappedServices
}

// Serialize serializes a CamelEnvironment.
func (env *CamelEnvironment) Serialize() (string, error) {
	res, err := json.Marshal(env)
	if err != nil {
		return "", err
	}

	return string(res), nil
}

// Deserialize deserializes a camel environment into this struct.
func (env *CamelEnvironment) Deserialize(str string) error {
	if err := json.Unmarshal([]byte(str), env); err != nil {
		return err
	}

	return nil
}

// ContainsService tells if the environment contains a service with the given name and type.
func (env *CamelEnvironment) ContainsService(name string, endpointKind CamelEndpointKind, serviceType CamelServiceType, apiVersion, kind string) bool {
	return env.FindService(name, endpointKind, serviceType, apiVersion, kind) != nil
}

// FindService -- .
func (env *CamelEnvironment) FindService(name string, endpointKind CamelEndpointKind, serviceType CamelServiceType, apiVersion, kind string) *CamelServiceDefinition {
	for _, svc := range env.Services {
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
