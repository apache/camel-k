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
	"strings"

	"github.com/pkg/errors"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/envvar"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type classpathTrait struct {
	BaseTrait `property:",squash"`
}

func newClasspathTrait() *classpathTrait {
	return &classpathTrait{
		BaseTrait: BaseTrait{
			id: ID("classpath"),
		},
	}
}

func (t *classpathTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}
	if e.InPhase(v1alpha1.IntegrationContextPhaseReady, v1alpha1.IntegrationPhaseDeploying) {
		return true, nil
	}

	return false, nil
}

func (t *classpathTrait) Apply(e *Environment) error {
	ctx := e.Context

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

	deps := make([]string, 0, 2+len(ctx.Status.Artifacts))
	deps = append(deps, "/etc/camel/resources")
	deps = append(deps, "./resources")

	for _, artifact := range ctx.Status.Artifacts {
		deps = append(deps, artifact.Target)
	}

	envvar.SetVal(&e.EnvVars, "JAVA_CLASSPATH", strings.Join(deps, ":"))

	return nil
}
