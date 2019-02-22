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
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/envvar"
)

type environmentTrait struct {
	BaseTrait     `property:",squash"`
	ContainerMeta bool `property:"container-meta"`
}

const (
	envVarNamespace            = "NAMESPACE"
	envVarPodName              = "POD_NAME"
	envVarCamelKVersion        = "CAMEL_K_VERSION"
	envVarCamelKRuntimeVersion = "CAMEL_K_RUNTIME_VERSION"
	envVarCamelVersion         = "CAMEL_VERSION"
)

func newEnvironmentTrait() *environmentTrait {
	return &environmentTrait{
		BaseTrait:     newBaseTrait("environment"),
		ContainerMeta: true,
	}
}

func (t *environmentTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled == nil || *t.Enabled {
		return e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying), nil
	}

	return false, nil
}

func (t *environmentTrait) Apply(e *Environment) error {
	envvar.SetVal(&e.EnvVars, envVarCamelKVersion, defaults.Version)
	envvar.SetVal(&e.EnvVars, envVarCamelKRuntimeVersion, e.RuntimeVersion)
	envvar.SetVal(&e.EnvVars, envVarCamelVersion, e.CamelCatalog.Version)

	if t.ContainerMeta {
		envvar.SetValFrom(&e.EnvVars, envVarNamespace, "metadata.namespace")
		envvar.SetValFrom(&e.EnvVars, envVarPodName, "metadata.name")
	}

	return nil
}
