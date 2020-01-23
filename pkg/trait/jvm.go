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
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/scylladb/go-set/strset"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/defaults"
)

const (
	defaultMainClass = "org.apache.camel.k.main.Application"
)

// The JVM trait is used to configure the JVM that runs the integration.
//
// +camel-k:trait=jvm
type jvmTrait struct {
	BaseTrait `property:",squash"`
}

func newJvmTrait() *jvmTrait {
	return &jvmTrait{
		BaseTrait: newBaseTrait("jvm"),
	}
}

func (t *jvmTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	return e.InPhase(v1.IntegrationKitPhaseReady, v1.IntegrationPhaseDeploying) ||
		e.InPhase(v1.IntegrationKitPhaseReady, v1.IntegrationPhaseRunning), nil
}

func (t *jvmTrait) Apply(e *Environment) error {
	kit := e.IntegrationKit

	if kit == nil && e.Integration.Status.Kit != "" {
		name := e.Integration.Status.Kit
		k := v1.NewIntegrationKit(e.Integration.Namespace, name)
		key := k8sclient.ObjectKey{
			Namespace: e.Integration.Namespace,
			Name:      name,
		}

		if err := t.client.Get(t.ctx, key, &k); err != nil {
			return errors.Wrapf(err, "unable to find integration kit %s, %s", name, err)
		}

		kit = &k
	}

	if kit == nil {
		return fmt.Errorf("unable to find integration kit %s", e.Integration.Status.Kit)
	}

	classpath := strset.New()

	classpath.Add("/etc/camel/resources")
	classpath.Add("./resources")

	quarkus := e.Catalog.GetTrait("quarkus").(*quarkusTrait)
	if !quarkus.isEnabled() {
		for _, artifact := range kit.Status.Artifacts {
			classpath.Add(artifact.Target)
		}
	}

	if kit.Labels["camel.apache.org/kit.type"] == v1.IntegrationKitTypeExternal {
		// In case of an external created kit, we do not have any information about
		// the classpath so we assume the all jars in /deployments/dependencies/ have
		// to be taken into account
		classpath.Add("/deployments/dependencies/*")
	}

	container := e.getIntegrationContainer()
	if container != nil {
		// Add mounted resources to the class path
		for _, m := range container.VolumeMounts {
			classpath.Add(m.MountPath)
		}
		items := classpath.List()
		// Keep class path sorted so that it's consistent over reconciliation cycles
		sort.Strings(items)

		container.Args = append(container.Args, "-cp", strings.Join(items, ":"))

		// Add main Class or JAR
		quarkus := e.Catalog.GetTrait("quarkus").(*quarkusTrait)
		if quarkus.isEnabled() {
			container.Args = append(container.Args, "-jar", "camel-k-integration-"+defaults.Version+"-runner.jar")
		} else {
			container.Args = append(container.Args, defaultMainClass)
		}

		// Set the container working directory
		container.WorkingDir = "/deployments"
	}

	return nil
}

// IsPlatformTrait overrides base class method
func (t *jvmTrait) IsPlatformTrait() bool {
	return true
}
