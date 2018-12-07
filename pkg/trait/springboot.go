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
	"sort"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/builder/springboot"
	"github.com/apache/camel-k/pkg/util"
)

type springBootTrait struct {
	BaseTrait `property:",squash"`
}

func newSpringBootTrait() *springBootTrait {
	return &springBootTrait{
		BaseTrait: BaseTrait{
			id: ID("springboot"),
		},
	}
}

func (t *springBootTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled == nil || !*t.Enabled {
		return false, nil
	}

	if e.IntegrationContextInPhase(v1alpha1.IntegrationContextPhaseBuilding) {
		return true, nil
	}
	if e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying) {
		return true, nil
	}
	if e.IntegrationInPhase("") {
		return true, nil
	}

	return false, nil
}

func (t *springBootTrait) Apply(e *Environment) error {

	//
	// Integration
	//

	if e.Integration != nil && e.Integration.Status.Phase == "" {
		util.StringSliceUniqueAdd(&e.Integration.Spec.Dependencies, "runtime:spring-boot")

		// sort the dependencies to get always the same list if they don't change
		sort.Strings(e.Integration.Spec.Dependencies)
	}

	if e.Integration != nil && e.Integration.Status.Phase == v1alpha1.IntegrationPhaseDeploying {
		// Override env vars
		e.EnvVars["JAVA_MAIN_CLASS"] = "org.springframework.boot.loader.PropertiesLauncher"
		e.EnvVars["LOADER_PATH"] = "/deployments/dependencies/"

		if e.Integration.Spec.Context != "" {
			name := e.Integration.Spec.Context
			ctx := v1alpha1.NewIntegrationContext(e.Integration.Namespace, name)

			if err := sdk.Get(&ctx); err != nil {
				return errors.Wrapf(err, "unable to find integration context %s, %s", ctx.Name, err)
			}

			deps := make([]string, 0, len(ctx.Status.Artifacts))
			for _, artifact := range ctx.Status.Artifacts {
				if strings.HasPrefix(artifact.ID, "org.apache.camel.k:camel-k-runtime-spring-boot:") {
					// do not include runner jar
					continue
				}

				deps = append(deps, artifact.Target)
			}

			e.EnvVars["LOADER_HOME"] = "/deployments"
			e.EnvVars["LOADER_PATH"] = strings.Join(deps, ",")
		}
	}

	//
	// Integration Context
	//

	if e.Context != nil && e.Context.Status.Phase == v1alpha1.IntegrationContextPhaseBuilding {
		// add custom initialization logic
		e.Steps = append(e.Steps, builder.NewStep("initialize/spring-boot", builder.IntiPhase, springboot.Initialize))
		e.Steps = append(e.Steps, builder.NewStep("build/compute-boot-dependencies", builder.ProjectBuildPhase+1, springboot.ComputeDependencies))

		// replace project generator
		for i := 0; i < len(e.Steps); i++ {
			if e.Steps[i].Phase() == builder.ProjectGenerationPhase {
				e.Steps[i] = builder.NewStep("generate/spring-boot", builder.ProjectGenerationPhase, springboot.GenerateProject)
			}
		}
	}

	return nil
}
