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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/envvar"

	"github.com/scylladb/go-set/strset"

	"github.com/pkg/errors"

	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

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

	return e.InPhase(v1alpha1.IntegrationContextPhaseReady, v1alpha1.IntegrationPhaseDeploying), nil
}

func (t *classpathTrait) Apply(e *Environment) error {
	ctx := e.IntegrationContext

	if ctx == nil && e.Integration.Status.Context != "" {
		name := e.Integration.Status.Context
		c := v1alpha1.NewIntegrationContext(e.Integration.Namespace, name)
		key := k8sclient.ObjectKey{
			Namespace: e.Integration.Namespace,
			Name:      name,
		}

		if err := t.client.Get(t.ctx, key, &c); err != nil {
			return errors.Wrapf(err, "unable to find integration context %s, %s", name, err)
		}

		ctx = &c
	}

	if ctx == nil {
		return fmt.Errorf("unable to find integration context %s", e.Integration.Status.Context)
	}

	if e.Classpath == nil {
		e.Classpath = strset.New()
	}

	e.Classpath.Add("/etc/camel/resources")
	e.Classpath.Add("./resources")

	for _, artifact := range ctx.Status.Artifacts {
		e.Classpath.Add(artifact.Target)
	}

	if e.IntegrationContext.Labels["camel.apache.org/context.type"] == v1alpha1.IntegrationContextTypeExternal {
		//
		// In case of an external created context. we do not have any information about
		// the classpath so we assume the all jars in /deployments/dependencies/ have
		// to be taken into account
		//
		e.Classpath.Add("/deployments/dependencies/*")
	}

	if e.Resources != nil {
		//
		// Add mounted volumes as resources
		//
		e.Resources.VisitDeployment(func(deployment *appsv1.Deployment) {
			for i := 0; i < len(deployment.Spec.Template.Spec.Containers); i++ {
				cp := e.Classpath.Copy()

				for _, m := range deployment.Spec.Template.Spec.Containers[i].VolumeMounts {
					cp.Add(m.MountPath)
				}

				t.setJavaClasspath(cp, &deployment.Spec.Template.Spec.Containers[i].Env)
			}
		})
		e.Resources.VisitKnativeService(func(service *serving.Service) {
			for _, m := range service.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.VolumeMounts {
				e.Classpath.Add(m.MountPath)
			}

			t.setJavaClasspath(e.Classpath, &service.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.Env)
		})
	}

	return nil
}

func (t *classpathTrait) setJavaClasspath(cp *strset.Set, env *[]corev1.EnvVar) {
	items := cp.List()

	// keep classpath sorted
	sort.Strings(items)

	envvar.SetVal(env, "JAVA_CLASSPATH", strings.Join(items, ":"))
}
