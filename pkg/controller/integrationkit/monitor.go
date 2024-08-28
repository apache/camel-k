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

package integrationkit

import (
	"context"
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/digest"
	corev1 "k8s.io/api/core/v1"
)

// NewMonitorAction creates a new monitoring handling action for the kit.
func NewMonitorAction() Action {
	return &monitorAction{}
}

type monitorAction struct {
	baseAction
}

func (action *monitorAction) Name() string {
	return "monitor"
}

func (action *monitorAction) CanHandle(kit *v1.IntegrationKit) bool {
	return kit.Status.Phase == v1.IntegrationKitPhaseReady
}

func (action *monitorAction) Handle(ctx context.Context, kit *v1.IntegrationKit) (*v1.IntegrationKit, error) {
	if kit.IsExternal() || kit.IsSynthetic() {
		// do nothing, it's not a managed kit
		// if it's a syntetic Kit add a condition to warn this is a
		// deprecated feature which may be removed soon.
		if kit.IsSynthetic() {
			kit.Status.SetCondition(
				v1.IntegrationKitConditionType("SyntheticKitDeprecated"),
				corev1.ConditionTrue,
				"DeprecationNotice",
				"Synthetic IntegrationKit feature is deprecated and will be removed soon.",
			)
			action.L.Infof("WARN: Synthetic IntegrationKit feature is deprecated and will be removed soon.")
		}
		return kit, nil
	}
	hash, err := digest.ComputeForIntegrationKit(kit)
	if err != nil {
		return nil, err
	}
	if hash != kit.Status.Digest {
		action.L.Info("IntegrationKit needs a rebuild")

		kit.Status.Digest = hash
		kit.Status.Phase = v1.IntegrationKitPhaseInitialization

		return kit, nil
	}
	if kit.Spec.Image != "" && kit.Spec.Image != kit.Status.Image {
		kit.Status.Phase = v1.IntegrationKitPhaseInitialization

		return kit, nil
	}

	action.checkTraitAnnotationsDeprecatedNotice(kit)

	return nil, nil
}

// Deprecated: to be removed in future versions, when we won't support any longer trait annotations into IntegrationKits.
func (action *monitorAction) checkTraitAnnotationsDeprecatedNotice(integrationKit *v1.IntegrationKit) {
	if integrationKit.Annotations != nil {
		for k := range integrationKit.Annotations {
			if strings.HasPrefix(k, v1.TraitAnnotationPrefix) {
				integrationKit.Status.SetCondition(
					v1.IntegrationKitConditionType("AnnotationTraitsDeprecated"),
					corev1.ConditionTrue,
					"DeprecationNotice",
					"Annotation traits configuration is deprecated and will be removed soon. Use .spec.traits configuration instead.",
				)

				action.L.Infof(
					"WARN: annotation traits configuration is deprecated and will be removed soon. Use .spec.traits configuration for %s integration kit instead.",
					integrationKit.Name,
				)
				return
			}
		}
	}
}
