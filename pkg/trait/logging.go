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

	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/envvar"
)

const (
	envVarQuarkusConsoleColor     = "QUARKUS_CONSOLE_COLOR"
	envVarQuarkusLogLevel         = "QUARKUS_LOG_LEVEL"
	envVarQuarkusLogConsoleFormat = "QUARKUS_LOG_CONSOLE_FORMAT"
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
			Level: defaultLogLevel,
		},
	}
}

func (l loggingTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil {
		return false, nil, nil
	}

	if !pointer.BoolDeref(l.Enabled, true) {
		return false, NewIntegrationConditionUserDisabled("Logging"), nil
	}

	return e.IntegrationInRunningPhases(), nil, nil
}

func (l loggingTrait) Apply(e *Environment) error {
	if e.CamelCatalog.Runtime.Capabilities["logging"].RuntimeProperties != nil {
		l.setCatalogConfiguration(e)
	} else {
		l.setEnvConfiguration(e)
	}

	return nil
}

// Deprecated: to be removed in future release in favor of func setCatalogConfiguration().
func (l loggingTrait) setEnvConfiguration(e *Environment) {
	envvar.SetVal(&e.EnvVars, envVarQuarkusLogLevel, l.Level)

	if l.Format != "" {
		envvar.SetVal(&e.EnvVars, envVarQuarkusLogConsoleFormat, l.Format)
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
}

func (l loggingTrait) setCatalogConfiguration(e *Environment) {
	if e.ApplicationProperties == nil {
		e.ApplicationProperties = make(map[string]string)
	}
	e.ApplicationProperties["camel.k.logging.level"] = l.Level
	if l.Format != "" {
		e.ApplicationProperties["camel.k.logging.format"] = l.Format
	}
	if pointer.BoolDeref(l.JSON, false) {
		e.ApplicationProperties["camel.k.logging.json"] = True
		if pointer.BoolDeref(l.JSONPrettyPrint, false) {
			e.ApplicationProperties["camel.k.logging.jsonPrettyPrint"] = True
		}
	} else {
		// If the trait is false OR unset, we default to false.
		e.ApplicationProperties["camel.k.logging.json"] = False
		if pointer.BoolDeref(l.Color, true) {
			e.ApplicationProperties["camel.k.logging.color"] = True
		}
	}

	for _, cp := range e.CamelCatalog.Runtime.Capabilities["logging"].RuntimeProperties {
		if CapabilityPropertyKey(cp.Value, e.ApplicationProperties) != "" {
			e.ApplicationProperties[CapabilityPropertyKey(cp.Key, e.ApplicationProperties)] = cp.Value
		}
	}
}
