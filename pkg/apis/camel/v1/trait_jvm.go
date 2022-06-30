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

package v1

// The JVM trait is used to configure the JVM that runs the integration.
//
// +camel-k:trait=jvm.
type JVMTrait struct {
	Trait `property:",squash" json:",inline"`
	// Activates remote debugging, so that a debugger can be attached to the JVM, e.g., using port-forwarding
	Debug *bool `property:"debug" json:"debug,omitempty"`
	// Suspends the target JVM immediately before the main class is loaded
	DebugSuspend *bool `property:"debug-suspend" json:"debugSuspend,omitempty"`
	// Prints the command used the start the JVM in the container logs (default `true`)
	PrintCommand *bool `property:"print-command" json:"printCommand,omitempty"`
	// Transport address at which to listen for the newly launched JVM (default `*:5005`)
	DebugAddress string `property:"debug-address" json:"debugAddress,omitempty"`
	// A list of JVM options
	Options []string `property:"options" json:"options,omitempty"`
	// Additional JVM classpath (use `Linux` classpath separator)
	Classpath string `property:"classpath" json:"classpath,omitempty"`
}
