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
	"strconv"
)

const (
	envVarQuarkusLogConsoleColor = "QUARKUS_LOG_CONSOLE_COLOR"
)

// This trait is used to control logging options (such as color)
//
// +camel-k:trait=logging
type loggingTrait struct {
	BaseTrait `property:",squash"`
	// Colorize the log output
	Color *bool `property:"color" json:"color,omitempty"`
}

func newLoggingTraitTrait() Trait {
	return &loggingTrait{
		BaseTrait: NewBaseTrait("logging", 800),
		Color:     util.BoolP(true),
	}
}

func (l loggingTrait) Configure(environment *Environment) (bool, error) {
	if l.Enabled != nil && !*l.Enabled {
		return false, nil
	}

	return environment.IntegrationInPhase(v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning), nil
}

func (l loggingTrait) Apply(environment *Environment) error {
	envvar.SetVal(&environment.EnvVars, envVarQuarkusLogConsoleColor, strconv.FormatBool(*l.Color))

	return nil
}
