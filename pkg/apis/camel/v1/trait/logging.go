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

// The Logging trait is used to configure Integration runtime logging options (such as color and format).
// The logging backend is provided by Quarkus, whose configuration is documented at https://quarkus.io/guides/logging.
//
// +camel-k:trait=logging.
type LoggingTrait struct {
	Trait `json:",inline" property:",squash"`

	// Colorize the log output
	Color *bool `json:"color,omitempty" property:"color"`
	// Logs message format
	Format string `json:"format,omitempty" property:"format"`
	// Adjust the logging level (defaults to `INFO`)
	// +kubebuilder:validation:Enum=FATAL;WARN;INFO;DEBUG;TRACE
	Level string `json:"level,omitempty" property:"level"`
	// Output the logs in JSON
	JSON *bool `json:"json,omitempty" property:"json"`
	// Enable "pretty printing" of the JSON logs
	JSONPrettyPrint *bool `json:"jsonPrettyPrint,omitempty" property:"json-pretty-print"`
}
