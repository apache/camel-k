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
	"os"

	"k8s.io/utils/pointer"

	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/envvar"
	"github.com/apache/camel-k/pkg/util/property"
)

type environmentTrait struct {
	BaseTrait
	traitv1.EnvironmentTrait `property:",squash"`
}

const (
	envVarNamespace            = "NAMESPACE"
	envVarPodName              = "POD_NAME"
	envVarOperatorID           = "CAMEL_K_OPERATOR_ID"
	envVarCamelKVersion        = "CAMEL_K_VERSION"
	envVarCamelKIntegration    = "CAMEL_K_INTEGRATION"
	envVarCamelKRuntimeVersion = "CAMEL_K_RUNTIME_VERSION"
	envVarMountPathConfigMaps  = "CAMEL_K_MOUNT_PATH_CONFIGMAPS"

	// Disabling gosec linter as it may triggers:
	//
	//   pkg/trait/environment.go:41: G101: Potential hardcoded credentials (gosec)
	//	   envVarMountPathSecrets     = "CAMEL_K_MOUNT_PATH_SECRETS"
	//
	// #nosec G101
	envVarMountPathSecrets = "CAMEL_K_MOUNT_PATH_SECRETS"
)

func newEnvironmentTrait() Trait {
	return &environmentTrait{
		BaseTrait: NewBaseTrait("environment", 800),
		EnvironmentTrait: traitv1.EnvironmentTrait{
			ContainerMeta: pointer.Bool(true),
		},
	}
}

func (t *environmentTrait) Configure(e *Environment) (bool, error) {
	if e.Integration == nil || !pointer.BoolDeref(t.Enabled, true) {
		return false, nil
	}

	return e.IntegrationInRunningPhases(), nil
}

func (t *environmentTrait) Apply(e *Environment) error {
	envvar.SetVal(&e.EnvVars, envVarCamelKVersion, defaults.Version)
	envvar.SetVal(&e.EnvVars, envVarOperatorID, defaults.OperatorID())
	if e.Integration != nil {
		envvar.SetVal(&e.EnvVars, envVarCamelKIntegration, e.Integration.Name)
	}
	envvar.SetVal(&e.EnvVars, envVarCamelKRuntimeVersion, e.RuntimeVersion)
	envvar.SetVal(&e.EnvVars, envVarMountPathConfigMaps, camel.ConfigConfigmapsMountPath)
	envvar.SetVal(&e.EnvVars, envVarMountPathSecrets, camel.ConfigSecretsMountPath)

	if pointer.BoolDeref(t.ContainerMeta, true) {
		envvar.SetValFrom(&e.EnvVars, envVarNamespace, "metadata.namespace")
		envvar.SetValFrom(&e.EnvVars, envVarPodName, "metadata.name")
	}

	if pointer.BoolDeref(t.HTTPProxy, true) {
		if HTTPProxy, ok := os.LookupEnv("HTTP_PROXY"); ok {
			envvar.SetVal(&e.EnvVars, "HTTP_PROXY", HTTPProxy)
		}
		if HTTPSProxy, ok := os.LookupEnv("HTTPS_PROXY"); ok {
			envvar.SetVal(&e.EnvVars, "HTTPS_PROXY", HTTPSProxy)
		}
		if noProxy, ok := os.LookupEnv("NO_PROXY"); ok {
			envvar.SetVal(&e.EnvVars, "NO_PROXY", noProxy)
		}
	}

	if t.Vars != nil {
		for _, env := range t.Vars {
			k, v := property.SplitPropertyFileEntry(env)
			envvar.SetVal(&e.EnvVars, k, v)
		}
	}

	return nil
}

// IsPlatformTrait overrides base class method.
func (t *environmentTrait) IsPlatformTrait() bool {
	return true
}
