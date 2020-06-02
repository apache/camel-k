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

	appsv1 "k8s.io/api/apps/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/defaults"
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

		integration.Status.Digest = hash
		integration.Status.Phase = v1.IntegrationPhaseInitialization
		if integration.Spec.Profile != v1.TraitProfile("") {
			integration.Status.Profile = integration.Spec.Profile
		}
		integration.Status.Version = defaults.Version

		return integration, nil
	}

	// Run traits that are enabled for the running phase
	_, err = trait.Apply(ctx, action.client, integration, nil)
	if err != nil {
		return nil, err
	}

	// Check replicas
	replicaSets := &appsv1.ReplicaSetList{}
	err = action.client.List(ctx, replicaSets,
		k8sclient.InNamespace(integration.Namespace),
		k8sclient.MatchingLabels{
			"camel.apache.org/integration": integration.Name,
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

	// Mirror ready condition from the sub resource (e.g.ReplicaSet, Deployment, CronJob, ...) to the integration
	kubernetes.MirrorReadyCondition(ctx, action.client, integration)

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
