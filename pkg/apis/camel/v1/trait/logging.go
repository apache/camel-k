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
	Trait `property:",squash" json:",inline"`
	// Colorize the log output
	Color *bool `property:"color" json:"color,omitempty"`
	// Logs message format
	Format string `property:"format" json:"format,omitempty"`
	// Adjust the logging level (defaults to INFO)
	Level string `property:"level" json:"level,omitempty"`
	// Output the logs in JSON
	JSON *bool `property:"json" json:"json,omitempty"`
	// Enable "pretty printing" of the JSON logs
	JSONPrettyPrint *bool `property:"json-pretty-print" json:"jsonPrettyPrint,omitempty"`
}
