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
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
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

type loggingTrait struct {
	BaseTrait
	v1.LoggingTrait `property:",squash"`
}

func newLoggingTraitTrait() Trait {
	return &loggingTrait{
		BaseTrait: NewBaseTrait("logging", 800),
		LoggingTrait: v1.LoggingTrait{
			Level: defaultLogLevel,
		},
	}
}

func (l loggingTrait) Configure(environment *Environment) (bool, error) {
	if !pointer.BoolDeref(l.Enabled, true) {
		return false, nil
	}

	return environment.IntegrationInRunningPhases(), nil
}

func (l loggingTrait) Apply(environment *Environment) error {
	envvar.SetVal(&environment.EnvVars, envVarQuarkusLogLevel, l.Level)

	if l.Format != "" {
		envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleFormat, l.Format)
	}

	if pointer.BoolDeref(l.JSON, false) {
		envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleJSON, True)
		if pointer.BoolDeref(l.JSONPrettyPrint, false) {
			envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleJSONPrettyPrint, True)
		}
	} else {
		// If the trait is false OR unset, we default to false.
		envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleJSON, False)

		if pointer.BoolDeref(l.Color, true) {
			envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleColor, True)
		}
	}

	return nil
}
