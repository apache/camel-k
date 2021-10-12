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

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pkg/errors"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func newInitializePodAction(reader ctrl.Reader) Action {
	return &initializePodAction{
		reader: reader,
	}
}

type initializePodAction struct {
	baseAction
	reader ctrl.Reader
}

// Name returns a common name of the action
func (action *initializePodAction) Name() string {
	return "initialize-pod"
}

// CanHandle tells whether this action can handle the build
func (action *initializePodAction) CanHandle(build *v1.Build) bool {
	return build.Status.Phase == "" || build.Status.Phase == v1.BuildPhaseInitialization
}

// Handle handles the builds
func (action *initializePodAction) Handle(ctx context.Context, build *v1.Build) (*v1.Build, error) {
	if err := deleteBuilderPod(ctx, action.client, build); err != nil {
		return nil, errors.Wrap(err, "cannot delete build pod")
	}

	pod, err := getBuilderPod(ctx, action.reader, build)
	if err != nil || pod != nil {
		// We return and wait for the pod to be deleted before de-queue the build pod.
		return nil, err
	}

	build.Status.Phase = v1.BuildPhaseScheduling

	return build, nil
}
