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

package integrationplatform

import (
	"context"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

// Action --
type Action interface {
	inject.Client

	// a user friendly name for the action
	Name() string

	// returns true if the action can handle the integration context
	CanHandle(platform *v1alpha1.IntegrationPlatform) bool

	// executes the handling function
	Handle(ctx context.Context, platform *v1alpha1.IntegrationPlatform) error
}

type baseAction struct {
	client client.Client
}

func (action *baseAction) InjectClient(client client.Client) error {
	action.client = client
	return nil
}
