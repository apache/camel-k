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

// TaskApplyConfiguration represents a declarative configuration of the Task type for use
// with apply.
type TaskApplyConfiguration struct {
	Builder  *BuilderTaskApplyConfiguration  `json:"builder,omitempty"`
	Custom   *UserTaskApplyConfiguration     `json:"custom,omitempty"`
	Package  *BuilderTaskApplyConfiguration  `json:"package,omitempty"`
	Buildah  *BuildahTaskApplyConfiguration  `json:"buildah,omitempty"`
	Kaniko   *KanikoTaskApplyConfiguration   `json:"kaniko,omitempty"`
	Spectrum *SpectrumTaskApplyConfiguration `json:"spectrum,omitempty"`
	S2i      *S2iTaskApplyConfiguration      `json:"s2i,omitempty"`
	Jib      *JibTaskApplyConfiguration      `json:"jib,omitempty"`
}

// TaskApplyConfiguration constructs a declarative configuration of the Task type for use with
// apply.
func Task() *TaskApplyConfiguration {
	return &TaskApplyConfiguration{}
}

// WithBuilder sets the Builder field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Builder field is set to the value of the last call.
func (b *TaskApplyConfiguration) WithBuilder(value *BuilderTaskApplyConfiguration) *TaskApplyConfiguration {
	b.Builder = value
	return b
}

// WithCustom sets the Custom field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Custom field is set to the value of the last call.
func (b *TaskApplyConfiguration) WithCustom(value *UserTaskApplyConfiguration) *TaskApplyConfiguration {
	b.Custom = value
	return b
}

// WithPackage sets the Package field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Package field is set to the value of the last call.
func (b *TaskApplyConfiguration) WithPackage(value *BuilderTaskApplyConfiguration) *TaskApplyConfiguration {
	b.Package = value
	return b
}

// WithBuildah sets the Buildah field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Buildah field is set to the value of the last call.
func (b *TaskApplyConfiguration) WithBuildah(value *BuildahTaskApplyConfiguration) *TaskApplyConfiguration {
	b.Buildah = value
	return b
}

// WithKaniko sets the Kaniko field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Kaniko field is set to the value of the last call.
func (b *TaskApplyConfiguration) WithKaniko(value *KanikoTaskApplyConfiguration) *TaskApplyConfiguration {
	b.Kaniko = value
	return b
}

// WithSpectrum sets the Spectrum field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Spectrum field is set to the value of the last call.
func (b *TaskApplyConfiguration) WithSpectrum(value *SpectrumTaskApplyConfiguration) *TaskApplyConfiguration {
	b.Spectrum = value
	return b
}

// WithS2i sets the S2i field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the S2i field is set to the value of the last call.
func (b *TaskApplyConfiguration) WithS2i(value *S2iTaskApplyConfiguration) *TaskApplyConfiguration {
	b.S2i = value
	return b
}

// WithJib sets the Jib field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Jib field is set to the value of the last call.
func (b *TaskApplyConfiguration) WithJib(value *JibTaskApplyConfiguration) *TaskApplyConfiguration {
	b.Jib = value
	return b
}
