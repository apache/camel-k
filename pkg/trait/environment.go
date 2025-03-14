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
	"fmt"
	"os"

	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/envvar"
	"github.com/apache/camel-k/v2/pkg/util/property"
)

type environmentTrait struct {
	BasePlatformTrait
	traitv1.EnvironmentTrait `property:",squash"`
}

const (
	environmentTraitID    = "environment"
	environmentTraitOrder = 800

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
		BasePlatformTrait: NewBasePlatformTrait(environmentTraitID, environmentTraitOrder),
	}
}

func (t *environmentTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil {
		return false, nil, nil
	}

	return e.IntegrationInRunningPhases(), nil, nil
}

func (t *environmentTrait) Apply(e *Environment) error {
	envvar.SetVal(&e.EnvVars, envVarCamelKVersion, e.Integration.Status.Version)
	envvar.SetVal(&e.EnvVars, envVarOperatorID, defaults.OperatorID())
	if e.Integration != nil {
		envvar.SetVal(&e.EnvVars, envVarCamelKIntegration, e.Integration.Name)
	}
	envvar.SetVal(&e.EnvVars, envVarCamelKRuntimeVersion, e.Integration.Status.RuntimeVersion)
	envvar.SetVal(&e.EnvVars, envVarMountPathConfigMaps, camel.ConfigConfigmapsMountPath)
	envvar.SetVal(&e.EnvVars, envVarMountPathSecrets, camel.ConfigSecretsMountPath)
	if e.CamelCatalog.GetRuntimeProvider() == v1.RuntimeProviderPlainQuarkus {
		envvar.SetVal(&e.EnvVars, "QUARKUS_CONFIG_LOCATIONS",
			fmt.Sprintf("%s/application.properties,%s/user.properties", camel.BasePath, camel.ConfDPath))
	}

	if ptr.Deref(t.ContainerMeta, true) {
		envvar.SetValFrom(&e.EnvVars, envVarNamespace, "metadata.namespace")
		envvar.SetValFrom(&e.EnvVars, envVarPodName, "metadata.name")
	}

	if ptr.Deref(t.HTTPProxy, true) {
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
			confs := v1.PlainConfigSecretRegexp.FindAllStringSubmatch(v, -1)
			if len(confs) > 0 {
				var res, err = v1.DecodeValueSource(v, "")
				if err != nil {
					return err
				}
				envvar.SetValFromValueSource(&e.EnvVars, k, res)
			} else {
				envvar.SetVal(&e.EnvVars, k, v)
			}
		}
	}

	return nil
}
