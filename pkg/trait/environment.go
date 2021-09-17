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
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/envvar"
)

// The environment trait is used internally to inject standard environment variables in the integration container,
// such as `NAMESPACE`, `POD_NAME` and others.
//
// +camel-k:trait=environment
type environmentTrait struct {
	BaseTrait `property:",squash"`
	// Enables injection of `NAMESPACE` and `POD_NAME` environment variables (default `true`)
	ContainerMeta *bool `property:"container-meta" json:"containerMeta,omitempty"`
}

const (
	envVarNamespace            = "NAMESPACE"
	envVarPodName              = "POD_NAME"
	envVarCamelKVersion        = "CAMEL_K_VERSION"
	envVarCamelKIntegration    = "CAMEL_K_INTEGRATION"
	envVarCamelKRuntimeVersion = "CAMEL_K_RUNTIME_VERSION"
	envVarMountPathConfigMaps  = "CAMEL_K_MOUNT_PATH_CONFIGMAPS"

	// Disabling gosec linter as it may triggers:
	//
	//   pkg/trait/environment.go:41: G101: Potential hardcoded credentials (gosec)
	//	   envVarMountPathSecrets     = "CAMEL_K_MOUNT_PATH_SECRETS"
	//
	// nolint: gosec
	envVarMountPathSecrets = "CAMEL_K_MOUNT_PATH_SECRETS"
)

func newEnvironmentTrait() Trait {
	return &environmentTrait{
		BaseTrait:     NewBaseTrait("environment", 800),
		ContainerMeta: BoolP(true),
	}
}

func (t *environmentTrait) Configure(e *Environment) (bool, error) {
	if IsNilOrTrue(t.Enabled) {
		return e.IntegrationInRunningPhases(), nil
	}

	return false, nil
}

func (t *environmentTrait) Apply(e *Environment) error {
	envvar.SetVal(&e.EnvVars, envVarCamelKVersion, defaults.Version)
	if e.Integration != nil {
		envvar.SetVal(&e.EnvVars, envVarCamelKIntegration, e.Integration.Name)
	}
	envvar.SetVal(&e.EnvVars, envVarCamelKRuntimeVersion, e.RuntimeVersion)
	envvar.SetVal(&e.EnvVars, envVarMountPathConfigMaps, configConfigmapsMountPath)
	envvar.SetVal(&e.EnvVars, envVarMountPathSecrets, configSecretsMountPath)

	if IsNilOrTrue(t.ContainerMeta) {
		envvar.SetValFrom(&e.EnvVars, envVarNamespace, "metadata.namespace")
		envvar.SetValFrom(&e.EnvVars, envVarPodName, "metadata.name")
	}

	return nil
}

// IsPlatformTrait overrides base class method
func (t *environmentTrait) IsPlatformTrait() bool {
	return true
}
