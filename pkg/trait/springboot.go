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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/builder/springboot"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/envvar"
)

type springBootTrait struct {
	BaseTrait `property:",squash"`
}

func newSpringBootTrait() *springBootTrait {
	return &springBootTrait{
		BaseTrait: newBaseTrait("springboot"),
	}
}

func (t *springBootTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled == nil || !*t.Enabled {
		return false, nil
	}

	if e.IntegrationContextInPhase(v1alpha1.IntegrationContextPhaseBuildSubmitted) {
		return true, nil
	}
	if e.InPhase(v1alpha1.IntegrationContextPhaseReady, v1alpha1.IntegrationPhaseDeploying) {
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

	if e.IntegrationInPhase("") {
		util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "runtime:spring-boot")

		// sort the dependencies to get always the same list if they don't change
		sort.Strings(e.Integration.Status.Dependencies)
	}

	if e.InPhase(v1alpha1.IntegrationContextPhaseReady, v1alpha1.IntegrationPhaseDeploying) {
		// Remove classpath
		envvar.Remove(&e.EnvVars, "JAVA_CLASSPATH")

		// Override env vars
		envvar.SetVal(&e.EnvVars, "JAVA_MAIN_CLASS", "org.springframework.boot.loader.PropertiesLauncher")

		deps := make([]string, 0, 2+len(e.IntegrationContext.Status.Artifacts))
		deps = append(deps, "/etc/camel/resources")
		deps = append(deps, "./resources")

		for _, artifact := range e.IntegrationContext.Status.Artifacts {
			if strings.HasPrefix(artifact.ID, "org.apache.camel.k:camel-k-runtime-spring-boot:") {
				// do not include runner jar
				continue
			}
			if strings.HasPrefix(artifact.ID, "org.apache.logging.log4j:") {
				// do not include logging, deps are embedded in runner jar
				continue
			}

			deps = append(deps, artifact.Target)
		}

		if e.IntegrationContext.Labels["camel.apache.org/context.type"] == v1alpha1.IntegrationContextTypeExternal {
			//
			// In case of an external created context. we do not have any information about
			// the classpath so we assume the all jars in /deployments/dependencies/ have
			// to be taken into account
			//
			deps = append(deps, "/deployments/dependencies/")
		}

		envvar.SetVal(&e.EnvVars, "LOADER_HOME", "/deployments")
		envvar.SetVal(&e.EnvVars, "LOADER_PATH", strings.Join(deps, ","))
	}

	//
	// Integration IntegrationContext
	//

	if e.IntegrationContextInPhase(v1alpha1.IntegrationContextPhaseBuildSubmitted) {
		// add custom initialization logic
		e.Steps = append(e.Steps, builder.NewStep("initialize/spring-boot", builder.InitPhase, springboot.Initialize))
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
