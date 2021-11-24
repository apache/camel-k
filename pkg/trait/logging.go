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

import (
	"github.com/apache/camel-k/pkg/util/envvar"
)

const (
	envVarQuarkusLogLevel                  = "QUARKUS_LOG_LEVEL"
	envVarQuarkusLogConsoleColor           = "QUARKUS_LOG_CONSOLE_COLOR"
	envVarQuarkusLogConsoleFormat          = "QUARKUS_LOG_CONSOLE_FORMAT"
	envVarQuarkusLogConsoleJSON            = "QUARKUS_LOG_CONSOLE_JSON"
	envVarQuarkusLogConsoleJSONPrettyPrint = "QUARKUS_LOG_CONSOLE_JSON_PRETTY_PRINT"
	defaultLogLevel                        = "INFO"
)

// The Logging trait is used to configure Integration runtime logging options (such as color and format).
// The logging backend is provided by Quarkus, whose configuration is documented at https://quarkus.io/guides/logging.
//
// +camel-k:trait=logging.
type loggingTrait struct {
	BaseTrait `property:",squash"`
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

func newLoggingTraitTrait() Trait {
	return &loggingTrait{
		BaseTrait: NewBaseTrait("logging", 800),
		Level:     defaultLogLevel,
	}
}

func (l loggingTrait) Configure(environment *Environment) (bool, error) {
	if IsFalse(l.Enabled) {
		return false, nil
	}

	return environment.IntegrationInRunningPhases(), nil
}

func (l loggingTrait) Apply(environment *Environment) error {
	envvar.SetVal(&environment.EnvVars, envVarQuarkusLogLevel, l.Level)

	if l.Format != "" {
		envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleFormat, l.Format)
	}

	if IsTrue(l.JSON) {
		envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleJSON, True)
		if IsTrue(l.JSONPrettyPrint) {
			envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleJSONPrettyPrint, True)
		}
	} else {
		// If the trait is false OR unset, we default to false.
		envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleJSON, False)

		if IsNilOrTrue(l.Color) {
			envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleColor, True)
		}
	}

	return nil
}
