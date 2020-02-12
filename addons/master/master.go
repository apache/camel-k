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

package master

import (
	"github.com/apache/camel-k/pkg/trait"
)

// The Master trait allows to configure the integration to automatically leverage Kubernetes resources for doing
// leader election and starting *master* routes only on certain instances.
//
// It's activated automatically when using the master endpoint in a route, e.g. `from("master:telegram:bots")...`.
//
// +camel-k:trait=master
type masterTrait struct {
	trait.BaseTrait `property:",squash"`
}

func NewMasterTrait() trait.Trait {
	return &masterTrait{
		BaseTrait: trait.NewBaseTrait("master", 2500),
	}
}

func (t *masterTrait) Configure(e *trait.Environment) (bool, error) {
	return false, nil
}

func (t *masterTrait) Apply(e *trait.Environment) error {

	return nil
}
