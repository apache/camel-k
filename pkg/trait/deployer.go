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

// The deployer trait can be used to explicitly select the kind of high level resource that
// will deploy the integration.
//
// +camel-k:trait=deployer
type deployerTrait struct {
	BaseTrait `property:",squash"`
	// Allows to explicitly select the desired deployment kind between `deployment` or `knative-service` when creating the resources for running the integration.
	Kind string `property:"kind"`
}

func newDeployerTrait() *deployerTrait {
	return &deployerTrait{
		BaseTrait: newBaseTrait("deployer"),
	}
}

func (t *deployerTrait) Configure(e *Environment) (bool, error) {
	return true, nil
}

func (t *deployerTrait) Apply(e *Environment) error {
	return nil
}

// IsPlatformTrait overrides base class method
func (t *deployerTrait) IsPlatformTrait() bool {
	return true
}
