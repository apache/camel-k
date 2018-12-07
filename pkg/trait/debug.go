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
)

type debugTrait struct {
	BaseTrait `property:",squash"`
}

func newDebugTrait() *debugTrait {
	return &debugTrait{
		BaseTrait: BaseTrait{
			id: ID("debug"),
		},
	}
}

func (t *debugTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && *t.Enabled {
		return e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying), nil
	}

	return false, nil
}

func (t *debugTrait) Apply(e *Environment) error {
	// this is all that's needed as long as the base image is `fabric8/s2i-java` look into builder/builder.go
	e.EnvVars["JAVA_DEBUG"] = True

	return nil
}
