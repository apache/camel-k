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

package keda

import (
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
)

// ScaleMapper defines the interface for mapping Camel component URIs to KEDA triggers.
type ScaleMapper interface {
	Component() string
	Map(pathValue string, params map[string]string) (kedaType string, metadata map[string]string)
}

var registry = map[string]ScaleMapper{}

func Register(m ScaleMapper) {
	registry[m.Component()] = m
}

func GetMapper(component string) (ScaleMapper, bool) {
	m, ok := registry[component]

	return m, ok
}

func MapToKedaTrigger(rawURI string) (*traitv1.KedaTrigger, error) {
	scheme, pathValue, params, err := ParseComponentURI(rawURI)
	if err != nil {
		return nil, err
	}
	if scheme == "" {
		return nil, nil
	}
	mapper, found := GetMapper(scheme)
	if !found {
		return nil, nil
	}
	kedaType, metadata := mapper.Map(pathValue, params)

	return &traitv1.KedaTrigger{
		Type:     kedaType,
		Metadata: metadata,
	}, nil
}
