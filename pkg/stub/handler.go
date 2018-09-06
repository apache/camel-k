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

package stub

import (
	"context"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/apache/camel-k/pkg/stub/action"
	"github.com/sirupsen/logrus"
)

func NewHandler(ctx context.Context, namespace string) sdk.Handler {
	return &Handler{
		actionPool: []action.Action{
			action.NewInitializeAction(),
			action.NewBuildAction(ctx, namespace),
			action.NewDeployAction(),
			action.NewMonitorAction(),
		},
	}
}

type Handler struct {
	actionPool	[]action.Action
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.Integration:
		for _, a := range h.actionPool {
			if a.CanHandle(o) {
				logrus.Info("Invoking action ", a.Name(), " on integration ", o.Name)
				if err := a.Handle(o); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
