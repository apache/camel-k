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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// DataTypeSpecApplyConfiguration represents a declarative configuration of the DataTypeSpec type for use
// with apply.
type DataTypeSpecApplyConfiguration struct {
	Scheme       *string                                 `json:"scheme,omitempty"`
	Format       *string                                 `json:"format,omitempty"`
	Description  *string                                 `json:"description,omitempty"`
	MediaType    *string                                 `json:"mediaType,omitempty"`
	Dependencies []string                                `json:"dependencies,omitempty"`
	Headers      map[string]HeaderSpecApplyConfiguration `json:"headers,omitempty"`
	Schema       *JSONSchemaPropsApplyConfiguration      `json:"schema,omitempty"`
}

// DataTypeSpecApplyConfiguration constructs a declarative configuration of the DataTypeSpec type for use with
// apply.
func DataTypeSpec() *DataTypeSpecApplyConfiguration {
	return &DataTypeSpecApplyConfiguration{}
}

// WithScheme sets the Scheme field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Scheme field is set to the value of the last call.
func (b *DataTypeSpecApplyConfiguration) WithScheme(value string) *DataTypeSpecApplyConfiguration {
	b.Scheme = &value
	return b
}

// WithFormat sets the Format field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Format field is set to the value of the last call.
func (b *DataTypeSpecApplyConfiguration) WithFormat(value string) *DataTypeSpecApplyConfiguration {
	b.Format = &value
	return b
}

// WithDescription sets the Description field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Description field is set to the value of the last call.
func (b *DataTypeSpecApplyConfiguration) WithDescription(value string) *DataTypeSpecApplyConfiguration {
	b.Description = &value
	return b
}

// WithMediaType sets the MediaType field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the MediaType field is set to the value of the last call.
func (b *DataTypeSpecApplyConfiguration) WithMediaType(value string) *DataTypeSpecApplyConfiguration {
	b.MediaType = &value
	return b
}

// WithDependencies adds the given value to the Dependencies field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Dependencies field.
func (b *DataTypeSpecApplyConfiguration) WithDependencies(values ...string) *DataTypeSpecApplyConfiguration {
	for i := range values {
		b.Dependencies = append(b.Dependencies, values[i])
	}
	return b
}

// WithHeaders puts the entries into the Headers field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the Headers field,
// overwriting an existing map entries in Headers field with the same key.
func (b *DataTypeSpecApplyConfiguration) WithHeaders(entries map[string]HeaderSpecApplyConfiguration) *DataTypeSpecApplyConfiguration {
	if b.Headers == nil && len(entries) > 0 {
		b.Headers = make(map[string]HeaderSpecApplyConfiguration, len(entries))
	}
	for k, v := range entries {
		b.Headers[k] = v
	}
	return b
}

// WithSchema sets the Schema field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Schema field is set to the value of the last call.
func (b *DataTypeSpecApplyConfiguration) WithSchema(value *JSONSchemaPropsApplyConfiguration) *DataTypeSpecApplyConfiguration {
	b.Schema = value
	return b
}
