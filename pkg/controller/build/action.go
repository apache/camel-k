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

package build

import (
	"context"

	"k8s.io/client-go/tools/record"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/event"
	"github.com/apache/camel-k/pkg/util/log"
)

// Action --
type Action interface {
	client.Injectable
	log.Injectable
	event.Injectable

	// Name returns user friendly name for the action
	Name() string

	// CanHandle returns true if the action can handle the build
	CanHandle(build *v1.Build) bool

	// Handle executes the handling function
	Handle(ctx context.Context, build *v1.Build) (*v1.Build, error)
}

type baseAction struct {
	client   client.Client
	L        log.Logger
	recorder record.EventRecorder
}

func (action *baseAction) InjectClient(client client.Client) {
	action.client = client
}

func (action *baseAction) InjectLogger(log log.Logger) {
	action.L = log
}

func (action *baseAction) InjectRecorder(recorder record.EventRecorder) {
	action.recorder = recorder
}
