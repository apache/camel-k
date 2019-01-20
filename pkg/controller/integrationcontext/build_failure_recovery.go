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

package integrationcontext

import (
	"context"
	"time"

	"github.com/jpillora/backoff"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/sirupsen/logrus"
)

// NewErrorRecoveryAction creates a new error recovering handling action for the context
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

func (action *errorRecoveryAction) CanHandle(ictx *v1alpha1.IntegrationContext) bool {
	return ictx.Status.Phase == v1alpha1.IntegrationContextPhaseBuildFailureRecovery
}

func (action *errorRecoveryAction) Handle(ctx context.Context, ictx *v1alpha1.IntegrationContext) error {
	// The integration platform needs to be initialized before starting handle
	// context in status error
	if _, err := platform.GetCurrentPlatform(ctx, action.client, ictx.Namespace); err != nil {
		logrus.Info("Waiting for a integration platform to be initialized")
		return nil
	}

	if ictx.Status.Failure.Recovery.Attempt > ictx.Status.Failure.Recovery.AttemptMax {
		target := ictx.DeepCopy()
		target.Status.Phase = v1alpha1.IntegrationContextPhaseError

		logrus.Infof("Max recovery attempt reached for context %s, transition to phase %s", ictx.Name, string(target.Status.Phase))

		return action.client.Update(ctx, target)
	}

	if ictx.Status.Failure != nil {
		lastAttempt := ictx.Status.Failure.Recovery.AttemptTime
		if lastAttempt.IsZero() {
			lastAttempt = ictx.Status.Failure.Time
		}

		elapsed := time.Since(lastAttempt).Seconds()
		elapsedMin := action.backOff.ForAttempt(float64(ictx.Status.Failure.Recovery.Attempt)).Seconds()

		if elapsed < elapsedMin {
			return nil
		}

		target := ictx.DeepCopy()
		target.Status = v1alpha1.IntegrationContextStatus{}
		target.Status.Phase = ""
		target.Status.Failure = ictx.Status.Failure
		target.Status.Failure.Recovery.Attempt = ictx.Status.Failure.Recovery.Attempt + 1
		target.Status.Failure.Recovery.AttemptTime = time.Now()

		logrus.Infof("Recovery attempt for context %s (%d/%d)",
			ictx.Name,
			target.Status.Failure.Recovery.Attempt,
			target.Status.Failure.Recovery.AttemptMax,
		)

		return action.client.Update(ctx, target)
	}

	return nil
}
