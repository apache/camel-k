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

package integration

import (
	"context"

	"github.com/pkg/errors"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

// NewMonitorAction creates a new monitoring action for an integration
func NewMonitorAction() Action {
	return &monitorAction{}
}

type monitorAction struct {
	baseAction
}

func (action *monitorAction) Name() string {
	return "monitor"
}

func (action *monitorAction) CanHandle(integration *v1.Integration) bool {
	return integration.Status.Phase == v1.IntegrationPhaseRunning
}

func (action *monitorAction) Handle(ctx context.Context, integration *v1.Integration) (*v1.Integration, error) {
	hash, err := digest.ComputeForIntegration(integration)
	if err != nil {
		return nil, err
	}

	if hash != integration.Status.Digest {
		action.L.Info("Integration needs a rebuild")

		integration.Initialize()
		integration.Status.Digest = hash

		return integration, nil
	}

	kit, err := kubernetes.GetIntegrationKit(ctx, action.client, integration.Status.IntegrationKit.Name, integration.Status.IntegrationKit.Namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to find integration kit %s/%s, %s", integration.Status.IntegrationKit.Namespace, integration.Status.IntegrationKit.Name, err)
	}

	// Run traits that are enabled for the running phase
	_, err = trait.Apply(ctx, action.client, integration, kit)
	if err != nil {
		return nil, err
	}

	// Enforce the scale sub-resource label selector.
	// It is used by the HPA that queries the scale sub-resource endpoint,
	// to list the pods owned by the integration.
	integration.Status.Selector = v1.IntegrationLabel + "=" + integration.Name

	// Check replicas
	replicaSets := &appsv1.ReplicaSetList{}
	err = action.client.List(ctx, replicaSets,
		k8sclient.InNamespace(integration.Namespace),
		k8sclient.MatchingLabels{
			v1.IntegrationLabel: integration.Name,
		})
	if err != nil {
		return nil, err
	}

	// And update the scale status accordingly
	if len(replicaSets.Items) > 0 {
		replicaSet := findLatestReplicaSet(replicaSets)
		replicas := replicaSet.Status.Replicas
		if integration.Status.Replicas == nil || replicas != *integration.Status.Replicas {
			integration.Status.Replicas = &replicas
		}
	}

	// Mirror ready condition from the owned resource (e.g.ReplicaSet, Deployment, CronJob, ...)
	// into the owning integration
	previous := integration.Status.GetCondition(v1.IntegrationConditionReady)
	kubernetes.MirrorReadyCondition(ctx, action.client, integration)

	if next := integration.Status.GetCondition(v1.IntegrationConditionReady); (previous == nil || previous.FirstTruthyTime == nil || previous.FirstTruthyTime.IsZero()) &&
		next != nil && next.Status == corev1.ConditionTrue && !(next.FirstTruthyTime == nil || next.FirstTruthyTime.IsZero()) {
		// Observe the time to first readiness metric
		duration := next.FirstTruthyTime.Time.Sub(integration.Status.InitializationTimestamp.Time)
		action.L.Infof("First readiness after %s", duration)
		timeToFirstReadiness.Observe(duration.Seconds())
	}

	// the integration pod may be in running phase, but the corresponding container running the integration code
	// may be in error state, in this case we should check the deployment status and set the integration status accordingly.
	if kubernetes.IsConditionTrue(integration, v1.IntegrationConditionDeploymentAvailable) {
		deployment, err := kubernetes.GetDeployment(ctx, action.client, integration.Name, integration.Namespace)
		if err != nil {
			return nil, err
		}

		deployUnavailable := false
		progressingFailing := false
		for _, c := range deployment.Status.Conditions {
			// first, check if the container status is not available
			if c.Type == appsv1.DeploymentAvailable {
				deployUnavailable = c.Status == corev1.ConditionFalse
			}
			// second, check when it is progressing and reason is the replicas are available but the number of replicas are zero
			// in this case, the container integration is failing
			if c.Type == appsv1.DeploymentProgressing {
				progressingFailing = c.Status == corev1.ConditionTrue && c.Reason == "NewReplicaSetAvailable" && deployment.Status.AvailableReplicas < 1
			}
		}
		if deployUnavailable && progressingFailing {
			notAvailableCondition := v1.IntegrationCondition{
				Type:    v1.IntegrationConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  v1.IntegrationConditionErrorReason,
				Message: "The corresponding pod(s) may be in error state, look at the pod status or log for errors",
			}
			integration.Status.SetConditions(notAvailableCondition)
			integration.Status.Phase = v1.IntegrationPhaseError
			return integration, nil
		}
	}

	return integration, nil
}

func findLatestReplicaSet(list *appsv1.ReplicaSetList) *appsv1.ReplicaSet {
	latest := list.Items[0]
	for i, rs := range list.Items[1:] {
		if latest.CreationTimestamp.Before(&rs.CreationTimestamp) {
			latest = list.Items[i+1]
		}
	}
	return &latest
}
