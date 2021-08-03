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
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/envvar"
)

const (
	envVarQuarkusLogLevel                  = "QUARKUS_LOG_LEVEL"
	envVarQuarkusLogConsoleColor           = "QUARKUS_LOG_CONSOLE_COLOR"
	envVarQuarkusLogConsoleFormat          = "QUARKUS_LOG_CONSOLE_FORMAT"
	envVarQuarkusLogConsoleJson            = "QUARKUS_LOG_CONSOLE_JSON"
	envVarQuarkusLogConsoleJsonPrettyPrint = "QUARKUS_LOG_CONSOLE_JSON_PRETTY_PRINT"
	depQuarkusLoggingJson                  = "mvn:io.quarkus:quarkus-logging-json"
	defaultLogLevel                        = "INFO"
)

// The Logging trait is used to configure Integration runtime logging options (such as color and format).
// The logging backend is provided by Quarkus, whose configuration is documented at https://quarkus.io/guides/logging.
//
// +camel-k:trait=logging
type loggingTrait struct {
	BaseTrait `property:",squash"`
	// Colorize the log output
	Color *bool `property:"color" json:"color,omitempty"`
	// Logs message format
	Format string `property:"format" json:"format,omitempty"`
	// Adjust the logging level (defaults to INFO)
	Level string `property:"level" json:"level,omitempty"`
	// Output the logs in JSON
	Json *bool `property:"json" json:"json,omitempty"`
	// Enable "pretty printing" of the JSON logs
	JsonPrettyPrint *bool `property:"json-pretty-print" json:"jsonPrettyPrint,omitempty"`
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

	return environment.IntegrationInPhase(v1.IntegrationPhaseInitialization, v1.IntegrationPhaseDeploying,
		v1.IntegrationPhaseRunning), nil
}

func (l loggingTrait) Apply(environment *Environment) error {
	if environment.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		if IsTrue(l.Json) {
			if environment.Integration.Status.Dependencies == nil {
				environment.Integration.Status.Dependencies = make([]string, 0)
			}
			util.StringSliceUniqueAdd(&environment.Integration.Status.Dependencies, depQuarkusLoggingJson)
		}

		return nil
	}

	envvar.SetVal(&environment.EnvVars, envVarQuarkusLogLevel, l.Level)

	if l.Format != "" {
		envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleFormat, l.Format)
	}

	if IsTrue(l.Json) {
		envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleJson, True)
		if IsTrue(l.JsonPrettyPrint) {
			envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleJsonPrettyPrint, True)
		}
	} else {
		envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleJson, False)

		if IsNilOrTrue(l.Color) {
			envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleColor, True)
		}
	}

	return nil
}
