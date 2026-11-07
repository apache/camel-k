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
	"strings"

	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/source"
)

// ScaleMapper defines the interface for mapping Camel component URIs to KEDA triggers.
type ScaleMapper interface {
	Component() string
	Map(pathValue string, params map[string]string, props map[string]string) (kedaType string, metadata map[string]string)
}

var registry = map[string]ScaleMapper{}

func Register(m ScaleMapper) {
	registry[m.Component()] = m
}

func GetMapper(component string) (ScaleMapper, bool) {
	m, ok := registry[component]

	return m, ok
}

// BeanAwareScaleMapper is an optional extension for mappings that can use
// directly declared bean configuration. Existing ScaleMapper implementations
// do not need to change.
type BeanAwareScaleMapper interface {
	ScaleMapper
	MapWithBeans(pathValue string, params map[string]string, props map[string]string, beans []source.BeanDefinition) (kedaType string, metadata map[string]string)
}

// ResolveBeanReference resolves the simple #beanName form against the bean
// definitions extracted from the Integration sources. Ambiguous and dynamic
// references are deliberately left unresolved.
func ResolveBeanReference(reference string, beans []source.BeanDefinition) (source.BeanDefinition, bool) {
	if !strings.HasPrefix(reference, "#") {
		return source.BeanDefinition{}, false
	}
	name := strings.TrimPrefix(reference, "#")
	if name == "" || strings.ContainsAny(name, " \t\r\n#{}$") {
		return source.BeanDefinition{}, false
	}

	var resolved source.BeanDefinition
	found := false
	for _, bean := range beans {
		if bean.Name == name {
			if found {
				return source.BeanDefinition{}, false
			}
			resolved = bean
			found = true
		}
	}

	return resolved, found
}

// MapToKedaTrigger preserves the original URI/property-only mapping entry point.
func MapToKedaTrigger(rawURI string, props map[string]string) (*traitv1.KedaTrigger, error) {
	return MapToKedaTriggerWithBeans(rawURI, props, nil)
}

// MapToKedaTriggerWithBeans also supplies statically extracted bean definitions
// to mappings that explicitly opt in through BeanAwareScaleMapper.
func MapToKedaTriggerWithBeans(rawURI string, props map[string]string, beans []source.BeanDefinition) (*traitv1.KedaTrigger, error) {
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

	var kedaType string
	var metadata map[string]string
	if beanMapper, ok := mapper.(BeanAwareScaleMapper); ok {
		kedaType, metadata = beanMapper.MapWithBeans(pathValue, params, props, beans)
	} else {
		kedaType, metadata = mapper.Map(pathValue, params, props)
	}

	if kedaType == "" {
		return nil, nil
	}

	return &traitv1.KedaTrigger{
		Type:     kedaType,
		Metadata: metadata,
	}, nil
}
