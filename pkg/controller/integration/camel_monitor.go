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

	monitorv1alpha1 "github.com/camel-tooling/camel-monitor-operator/pkg/apis/camel/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func addCamelMonitorCondition(ctx context.Context, c ctrl.Reader, integration *v1.Integration) error {
	monitor := &monitorv1alpha1.CamelMonitor{}
	if err := c.Get(ctx, ctrl.ObjectKeyFromObject(integration), monitor); err != nil {
		if k8serrors.IsNotFound(err) {
			integration.Status.RemoveCondition(v1.IntegrationConditionCamelMonitorAvailable)

			return nil
		}

		return err
	}
	integration.Status.SetCondition(v1.IntegrationConditionCamelMonitorAvailable, corev1.ConditionTrue,
		"CamelMonitorAvailable", "A CamelMonitor resource is available")

	return nil
}
