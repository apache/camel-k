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

package trait

// The Quarkus trait configures the Quarkus runtime.
//
// It's enabled by default.
//
// NOTE: A native based compilation will be forced to use a `pod` build strategy.
// Compiling to a native executable, i.e. when using `build-mode=native`, requires at least
// 4GiB of memory, so the Pod running the native build, must have enough memory available.
//
// +camel-k:trait=quarkus.
type QuarkusTrait struct {
	Trait `property:",squash" json:",inline"`
	// The Quarkus package types, `fast-jar` or `native` (default `fast-jar`).
	// In case both `fast-jar` and `native` are specified, two `IntegrationKit` resources are created,
	// with the native kit having precedence over the `fast-jar` one once ready.
	// The order influences the resolution of the current kit for the integration.
	// The kit corresponding to the first package type will be assigned to the
	// integration in case no existing kit that matches the integration exists.
	// Deprecated: use `build-mode` instead.
	PackageTypes []QuarkusPackageType `property:"package-type" json:"packageTypes,omitempty"`
	// The Quarkus mode to run: either `jvm` or `native` (default `jvm`).
	// In case both `jvm` and `native` are specified, two `IntegrationKit` resources are created,
	// with the `native` kit having precedence over the `jvm` one once ready.
	Modes []QuarkusMode `property:"build-mode" json:"buildMode,omitempty"`
}

// QuarkusMode is the type of Quarkus build packaging.
// +kubebuilder:validation:Enum=jvm;native
type QuarkusMode string

const (
	// JvmQuarkusMode represents "JVM mode" Quarkus execution.
	JvmQuarkusMode QuarkusMode = "jvm"
	// NativeQuarkusMode represents "Native mode" Quarkus execution.
	NativeQuarkusMode QuarkusMode = "native"
)

// QuarkusPackageType is the type of Quarkus build packaging.
// Deprecated: use `QuarkusMode` instead.
// +kubebuilder:validation:Enum=fast-jar;native
type QuarkusPackageType string

const (
	// FastJarPackageType represents "fast jar" Quarkus packaging.
	FastJarPackageType QuarkusPackageType = "fast-jar"
	// NativePackageType represents "native" Quarkus packaging.
	NativePackageType QuarkusPackageType = "native"
)
