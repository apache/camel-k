//go:build integration
// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "integration"

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

package common

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestOwnerTrait(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("selected labels and annotations are propagated to the Deployment", func(t *testing.T) {
			name := RandomizedSuffixName("owner-selected")
			selectedLabel := "owner-test-selected-label"
			unselectedLabel := "owner-test-unselected-label"
			selectedAnnotation := "kubectl.kubernetes.io/owner-test-selected"
			unselectedAnnotation := "kubectl.kubernetes.io/owner-test-unselected"
			defaultContainerAnnotation := "kubectl.kubernetes.io/default-container"

			g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java",
				"--name", name,
				"--label", selectedLabel+"=selected",
				"--label", unselectedLabel+"=unselected",
				"--annotation", selectedAnnotation+"=selected",
				"--annotation", unselectedAnnotation+"=unselected",
				"--annotation", defaultContainerAnnotation+"=must-not-overwrite",
				"-t", "owner.target-labels="+selectedLabel,
				"-t", "owner.target-annotations="+selectedAnnotation+","+defaultContainerAnnotation,
			).Execute()).To(Succeed())

			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).
				Should(Equal(corev1.ConditionTrue))

			integration := Integration(t, ctx, ns, name)()
			g.Expect(integration).NotTo(BeNil())

			deployment := Deployment(t, ctx, ns, name)()
			g.Expect(deployment).NotTo(BeNil())
			g.Expect(deployment.Labels).To(HaveKeyWithValue(selectedLabel, "selected"))
			g.Expect(deployment.Labels).NotTo(HaveKey(unselectedLabel))
			g.Expect(deployment.Spec.Template.Labels).To(HaveKeyWithValue(selectedLabel, "selected"))
			g.Expect(deployment.Spec.Template.Labels).NotTo(HaveKey(unselectedLabel))

			g.Expect(deployment.Annotations).To(HaveKeyWithValue(selectedAnnotation, "selected"))
			g.Expect(deployment.Annotations).NotTo(HaveKey(unselectedAnnotation))
			g.Expect(deployment.Spec.Template.Annotations).To(HaveKeyWithValue(selectedAnnotation, "selected"))
			g.Expect(deployment.Spec.Template.Annotations).NotTo(HaveKey(unselectedAnnotation))

			// The Deployment trait sets this annotation before the Owner trait runs.
			// Owner propagation must not overwrite metadata already present on a resource.
			g.Expect(deployment.Annotations).To(HaveKey(defaultContainerAnnotation))
			g.Expect(deployment.Annotations[defaultContainerAnnotation]).NotTo(Equal("must-not-overwrite"))
			g.Expect(deployment.Spec.Template.Annotations[defaultContainerAnnotation]).
				NotTo(Equal("must-not-overwrite"))

			g.Expect(deployment.OwnerReferences).To(HaveLen(1))
			ownerReference := deployment.OwnerReferences[0]
			g.Expect(ownerReference.APIVersion).To(Equal(integration.APIVersion))
			g.Expect(ownerReference.Kind).To(Equal(integration.Kind))
			g.Expect(ownerReference.Name).To(Equal(integration.Name))
			g.Expect(ownerReference.UID).To(Equal(integration.UID))
			g.Expect(ownerReference.Controller).To(PointTo(BeTrue()))
			g.Expect(ownerReference.BlockOwnerDeletion).To(PointTo(BeTrue()))
		})

		t.Run("wildcard propagates all labels and annotations", func(t *testing.T) {
			name := RandomizedSuffixName("owner-wildcard")
			labelOne := "owner-test-wildcard-one"
			labelTwo := "owner-test-wildcard-two"
			annotationOne := "kubectl.kubernetes.io/owner-test-wildcard-one"
			annotationTwo := "kubectl.kubernetes.io/owner-test-wildcard-two"

			g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java",
				"--name", name,
				"--label", labelOne+"=one",
				"--label", labelTwo+"=two",
				"--annotation", annotationOne+"=one",
				"--annotation", annotationTwo+"=two",
				"-t", "owner.target-labels=*",
				"-t", "owner.target-annotations=*",
			).Execute()).To(Succeed())

			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(Deployment(t, ctx, ns, name), TestTimeoutMedium).ShouldNot(BeNil())

			deployment := Deployment(t, ctx, ns, name)()
			g.Expect(deployment.Labels).To(HaveKeyWithValue(labelOne, "one"))
			g.Expect(deployment.Labels).To(HaveKeyWithValue(labelTwo, "two"))
			g.Expect(deployment.Spec.Template.Labels).To(HaveKeyWithValue(labelOne, "one"))
			g.Expect(deployment.Spec.Template.Labels).To(HaveKeyWithValue(labelTwo, "two"))
			g.Expect(deployment.Annotations).To(HaveKeyWithValue(annotationOne, "one"))
			g.Expect(deployment.Annotations).To(HaveKeyWithValue(annotationTwo, "two"))
			g.Expect(deployment.Spec.Template.Annotations).To(HaveKeyWithValue(annotationOne, "one"))
			g.Expect(deployment.Spec.Template.Annotations).To(HaveKeyWithValue(annotationTwo, "two"))
			g.Expect(deployment.OwnerReferences).To(HaveLen(1))
		})

		t.Run("disabled owner trait does not propagate metadata or set owner references", func(t *testing.T) {
			name := RandomizedSuffixName("owner-disabled")
			label := "owner-test-disabled-label"
			annotation := "kubectl.kubernetes.io/owner-test-disabled"

			g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java",
				"--name", name,
				"--label", label+"=disabled",
				"--annotation", annotation+"=disabled",
				"-t", "owner.enabled=false",
				"-t", "owner.target-labels=*",
				"-t", "owner.target-annotations=*",
			).Execute()).To(Succeed())

			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(Deployment(t, ctx, ns, name), TestTimeoutMedium).ShouldNot(BeNil())

			deployment := Deployment(t, ctx, ns, name)()
			g.Expect(deployment.OwnerReferences).To(BeEmpty())
			g.Expect(deployment.Labels).NotTo(HaveKey(label))
			g.Expect(deployment.Spec.Template.Labels).NotTo(HaveKey(label))
			g.Expect(deployment.Annotations).NotTo(HaveKey(annotation))
			g.Expect(deployment.Spec.Template.Annotations).NotTo(HaveKey(annotation))
		})
	})
}
