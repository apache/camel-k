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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/platform"

	"github.com/jpillora/backoff"
)

// NewErrorRecoveryAction creates a new error recovering handling action for the build
func NewErrorRecoveryAction() Action {
	//TODO: externalize options
	return &errorRecoveryAction{
		backOff: backoff.Backoff{
			Min:    5 * time.Second,
			Max:    1 * time.Minute,
			Factor: 2,
			Jitter: false,
		},
	}
}

type errorRecoveryAction struct {
	baseAction
	backOff backoff.Backoff
}

func (action *errorRecoveryAction) Name() string {
	return "error-recovery"
}

func (action *errorRecoveryAction) CanHandle(build *v1alpha1.Build) bool {
	return build.Status.Phase == v1alpha1.BuildPhaseFailed
}

func (action *errorRecoveryAction) Handle(ctx context.Context, build *v1alpha1.Build) error {
	// The integration platform must be initialized before handling the error recovery
	if _, err := platform.GetCurrentPlatform(ctx, action.client, build.Namespace); err != nil {
		action.L.Info("Waiting for an integration platform to be initialized")
		return nil
	}

	if build.Status.Failure == nil {
		build.Status.Failure = &v1alpha1.Failure{
			Reason: build.Status.Error,
			Time:   metav1.Now(),
			Recovery: v1alpha1.FailureRecovery{
				Attempt:    0,
				AttemptMax: 5,
			},
		}
	}

	err := action.client.Status().Update(ctx, build)
	if err != nil {
		return err
	}

	target := build.DeepCopy()

	if build.Status.Failure.Recovery.Attempt >= build.Status.Failure.Recovery.AttemptMax {
		target.Status.Phase = v1alpha1.BuildPhaseError

		action.L.Info("Max recovery attempt reached, transition to error phase")

		return action.client.Status().Update(ctx, target)
	}

	lastAttempt := build.Status.Failure.Recovery.AttemptTime.Time
	if lastAttempt.IsZero() {
		lastAttempt = build.Status.Failure.Time.Time
	}

	elapsed := time.Since(lastAttempt).Seconds()
	elapsedMin := action.backOff.ForAttempt(float64(build.Status.Failure.Recovery.Attempt)).Seconds()

	if elapsed < elapsedMin {
		return nil
	}

	target.Status = v1alpha1.BuildStatus{}
	target.Status.Phase = ""
	target.Status.Failure = build.Status.Failure
	target.Status.Failure.Recovery.Attempt = build.Status.Failure.Recovery.Attempt + 1
	target.Status.Failure.Recovery.AttemptTime = metav1.Now()

	action.L.Infof("Recovery attempt (%d/%d)",
		target.Status.Failure.Recovery.Attempt,
		target.Status.Failure.Recovery.AttemptMax,
	)

	return action.client.Status().Update(ctx, target)
}
