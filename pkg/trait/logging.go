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
	"strings"

	"k8s.io/utils/pointer"

	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/envvar"
)

const (
	envVarQuarkusConsoleColor     = "QUARKUS_CONSOLE_COLOR"
	envVarQuarkusLogLevel         = "QUARKUS_LOG_LEVEL"
	envVarQuarkusLogConsoleFormat = "QUARKUS_LOG_CONSOLE_FORMAT"
	envVarQuarkusLogCategory      = "QUARKUS_LOG_CATEGORY"
	// nolint: gosec // no sensitive credentials
	envVarQuarkusLogConsoleJSON            = "QUARKUS_LOG_CONSOLE_JSON"
	envVarQuarkusLogConsoleJSONPrettyPrint = "QUARKUS_LOG_CONSOLE_JSON_PRETTY_PRINT"
	defaultLogLevel                        = "INFO"
)

type loggingTrait struct {
	BaseTrait
	traitv1.LoggingTrait `property:",squash"`
}

func newLoggingTraitTrait() Trait {
	return &loggingTrait{
		BaseTrait: NewBaseTrait("logging", 800),
		LoggingTrait: traitv1.LoggingTrait{
			Level:    defaultLogLevel,
			Category: map[string]string{},
		},
	}
}

func (l loggingTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil {
		return false, nil, nil
	}

	if !pointer.BoolDeref(l.Enabled, true) {
		return false, NewIntegrationConditionUserDisabled(), nil
	}

	return e.IntegrationInRunningPhases(), nil, nil
}

func (l loggingTrait) Apply(e *Environment) error {
	envvar.SetVal(&e.EnvVars, envVarQuarkusLogLevel, l.Level)

	if l.Format != "" {
		envvar.SetVal(&e.EnvVars, envVarQuarkusLogConsoleFormat, l.Format)
	}

	if len(l.Category) > 0 {
		for k, v := range l.Category {
			envVarQuarkusPackage := strings.ReplaceAll(strings.ToUpper(k), ".", "_")
			envVarQuarkusLogCategoryPackageFormat := envVarQuarkusLogCategory + "_" + envVarQuarkusPackage + "_LEVEL"
			envVarQuarkusLogCatagoryValue := strings.ToUpper(v)
			envvar.SetVal(&e.EnvVars, envVarQuarkusLogCategoryPackageFormat, envVarQuarkusLogCatagoryValue)
		}
	}

	if pointer.BoolDeref(l.JSON, false) {
		envvar.SetVal(&e.EnvVars, envVarQuarkusLogConsoleJSON, True)
		if pointer.BoolDeref(l.JSONPrettyPrint, false) {
			envvar.SetVal(&e.EnvVars, envVarQuarkusLogConsoleJSONPrettyPrint, True)
		}
	} else {
		// If the trait is false OR unset, we default to false.
		envvar.SetVal(&e.EnvVars, envVarQuarkusLogConsoleJSON, False)

		if pointer.BoolDeref(l.Color, true) {
			envvar.SetVal(&e.EnvVars, envVarQuarkusConsoleColor, True)
		}
	}

	return nil
}
