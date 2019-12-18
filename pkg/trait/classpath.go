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
	"github.com/apache/camel-k/pkg/util/envvar"
)

// The Classpath trait is used internally to configure the classpath of the final integration.
//
// +camel-k:trait=classpath
type classpathTrait struct {
	BaseTrait `property:",squash"`
}

func newClasspathTrait() *classpathTrait {
	return &classpathTrait{
		BaseTrait: newBaseTrait("classpath"),
	}
}

func (t *classpathTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	return e.InPhase(v1.IntegrationKitPhaseReady, v1.IntegrationPhaseDeploying) ||
		e.InPhase(v1.IntegrationKitPhaseReady, v1.IntegrationPhaseRunning), nil
}

func (t *classpathTrait) Apply(e *Environment) error {
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

	if e.Classpath == nil {
		e.Classpath = strset.New()
	}

	e.Classpath.Add("/etc/camel/resources")
	e.Classpath.Add("./resources")

	quarkus := e.Catalog.GetTrait("quarkus").(*quarkusTrait)
	if quarkus.isEnabled() {
		quarkus.addClasspath(e)
	} else {
		for _, artifact := range kit.Status.Artifacts {
			e.Classpath.Add(artifact.Target)
		}
	}

	if kit.Labels["camel.apache.org/kit.type"] == v1.IntegrationKitTypeExternal {
		//
		// In case of an external created kit, we do not have any information about
		// the classpath so we assume the all jars in /deployments/dependencies/ have
		// to be taken into account
		//
		e.Classpath.Add("/deployments/dependencies/*")
	}

	containerName := defaultContainerName
	dt := e.Catalog.GetTrait(containerTraitID)
	if dt != nil {
		containerName = dt.(*containerTrait).Name
	}

	container := e.Resources.GetContainerByName(containerName)
	if container != nil {
		for _, m := range container.VolumeMounts {
			e.Classpath.Add(m.MountPath)
		}

		items := e.Classpath.List()

		// keep classpath sorted
		sort.Strings(items)

		envvar.SetVal(&container.Env, "JAVA_CLASSPATH", strings.Join(items, ":"))
	}

	return nil
}

// IsPlatformTrait overrides base class method
func (t *classpathTrait) IsPlatformTrait() bool {
	return true
}
