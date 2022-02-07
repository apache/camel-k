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

package kamelet

import (
	"context"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	kameletutils "github.com/apache/camel-k/pkg/kamelet"
)

// NewInitializeAction returns a action that initializes the kamelet configuration when not provided by the user.
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
	baseAction
}

func (action *initializeAction) Name() string {
	return "initialize"
}

func (action *initializeAction) CanHandle(kamelet *v1alpha1.Kamelet) bool {
	return kamelet.Status.Phase == v1alpha1.KameletPhaseNone || kamelet.Status.Phase == v1alpha1.KameletPhaseError
}

func (action *initializeAction) Handle(ctx context.Context, kamelet *v1alpha1.Kamelet) (*v1alpha1.Kamelet, error) {
	return kameletutils.Initialize(kamelet)
}
