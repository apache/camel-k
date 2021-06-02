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
	"strconv"

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
	defaultLogFormat                       = ""
	defaultLogLevel                        = "INFO"
)

// This trait is used to control logging options (such as color and the format). The logging backend is provided by
// Quarkus and configuration details for things like the the log format can be found on https://quarkus.io/guides/logging
//
// +camel-k:trait=logging
type loggingTrait struct {
	BaseTrait `property:",squash"`
	// Colorize the log output
	Color *bool `property:"color" json:"color,omitempty"`
	// Log message format
	Format string `property:"format" json:"format,omitempty"`
	// Adjust the log level for the integrations (defaults to INFO)
	Level string `property:"level" json:"level,omitempty"`
	// Output the log in json format
	Json *bool `property:"json" json:"json,omitempty"`
	// Enable "pretty printing" of the json log
	JsonPrettyPrint *bool `property:"json-pretty-print" json:"jsonPrettyPrint,omitempty"`
}

func newLoggingTraitTrait() Trait {
	return &loggingTrait{
		BaseTrait:       NewBaseTrait("logging", 800),
		Color:           util.BoolP(true),
		Format:          defaultLogFormat,
		Level:           defaultLogLevel,
		Json:            util.BoolP(false),
		JsonPrettyPrint: util.BoolP(false),
	}
}

func (l loggingTrait) Configure(environment *Environment) (bool, error) {
	if l.Enabled != nil && !*l.Enabled {
		return false, nil
	}

	return environment.IntegrationInPhase(v1.IntegrationPhaseInitialization, v1.IntegrationPhaseDeploying,
		v1.IntegrationPhaseRunning), nil
}

func (l loggingTrait) Apply(environment *Environment) error {

	if environment.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		if *l.Json {
			if environment.Integration.Status.Dependencies == nil {
				environment.Integration.Status.Dependencies = make([]string, 0)
			}

			util.StringSliceUniqueAdd(&environment.Integration.Status.Dependencies, depQuarkusLoggingJson)
		}

		return nil
	}

	envvar.SetVal(&environment.EnvVars, envVarQuarkusLogLevel, l.Level)

	if l.Format != defaultLogFormat {
		envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleFormat, l.Format)
	}

	envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleJson, strconv.FormatBool(*l.Json))
	envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleJsonPrettyPrint, strconv.FormatBool(*l.JsonPrettyPrint))

	if !*l.Json {
		envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleColor, strconv.FormatBool(*l.Color))
	}

	return nil
}
