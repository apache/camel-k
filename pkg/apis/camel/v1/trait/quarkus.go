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
// NOTE: Compiling to a native executable, i.e. when using `package-type=native`, is only supported
// for kamelets, as well as YAML and XML integrations.
// It also requires at least 4GiB of memory, so the Pod running the native build, that is either
// the operator Pod, or the build Pod (depending on the build strategy configured for the platform),
// must have enough memory available.
//
// +camel-k:trait=quarkus.
type QuarkusTrait struct {
	Trait `property:",squash" json:",inline"`
	// The Quarkus package types, either `fast-jar` or `native` (default `fast-jar`).
	// In case both `fast-jar` and `native` are specified, two `IntegrationKit` resources are created,
	// with the `native` kit having precedence over the `fast-jar` one once ready.
	// The order influences the resolution of the current kit for the integration.
	// The kit corresponding to the first package type will be assigned to the
	// integration in case no existing kit that matches the integration exists.
	PackageTypes []QuarkusPackageType `property:"package-type" json:"packageTypes,omitempty"`
}

type QuarkusPackageType string

const (
	FastJarPackageType QuarkusPackageType = "fast-jar"
	NativePackageType  QuarkusPackageType = "native"
)
