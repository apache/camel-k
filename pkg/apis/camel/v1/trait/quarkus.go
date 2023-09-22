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
// Compiling to a native executable, i.e. when using `package-type=native`, requires at least
// 4GiB of memory, so the Pod running the native build, must have enough memory available.
//
// +camel-k:trait=quarkus.
type QuarkusTrait struct {
	Trait `property:",squash" json:",inline"`
	// The Quarkus package types, `fast-jar`, `native-sources` or `native` (default `fast-jar`). `native` is deprecated.
	// In case both `fast-jar` and `native` or `native-sources` are specified, two `IntegrationKit` resources are created,
	// with the native kit having precedence over the `fast-jar` one once ready.
	// The order influences the resolution of the current kit for the integration.
	// The kit corresponding to the first package type will be assigned to the
	// integration in case no existing kit that matches the integration exists.
	PackageTypes []QuarkusPackageType `property:"package-type" json:"packageTypes,omitempty"`
}

// QuarkusPackageType is the type of Quarkus build packaging.
// +kubebuilder:validation:Enum=fast-jar;native-sources;native
type QuarkusPackageType string

const (
	// FastJarPackageType represents "fast jar" Quarkus packaging.
	FastJarPackageType QuarkusPackageType = "fast-jar"
	// NativePackageType represents "native" Quarkus packaging.
	// Deprecated: use native-sources instead.
	NativePackageType QuarkusPackageType = "native"
	// NativeSourcesPackageType represents "native-sources" Quarkus packaging.
	NativeSourcesPackageType QuarkusPackageType = "native-sources"
)
