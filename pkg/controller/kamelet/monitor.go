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

// NewMonitorAction returns an action that monitors the kamelet after it's fully initialized.
func NewMonitorAction() Action {
	return &monitorAction{}
}

type monitorAction struct {
	baseAction
}

func (action *monitorAction) Name() string {
	return "monitor"
}

func (action *monitorAction) CanHandle(kamelet *v1alpha1.Kamelet) bool {
	return kamelet.Status.Phase == v1alpha1.KameletPhaseReady || kamelet.Status.Phase == v1alpha1.KameletPhaseError
}

func (action *monitorAction) Handle(ctx context.Context, kamelet *v1alpha1.Kamelet) (*v1alpha1.Kamelet, error) {
	return kameletutils.Initialize(kamelet)
}
