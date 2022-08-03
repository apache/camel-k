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
	"github.com/onsi/gomega/gstruct"
	"testing"

	. "github.com/apache/camel-k/e2e/support"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

func TestBindingWithImage(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-binding-image"
		bindingID := "with-image-binding"

		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		from := corev1.ObjectReference{
			Kind:       "Kamelet",
			Name:       "my-own-timer-source",
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		}

		to := corev1.ObjectReference{
			Kind:       "Kamelet",
			Name:       "my-own-log-sink",
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		}

		emptyMap := map[string]string{}

		annotations1 := map[string]string{
			"trait.camel.apache.org/container.image":      "docker.io/jmalloc/echo-server:0.3.2",
			"trait.camel.apache.org/jvm.enabled":          "false",
			"trait.camel.apache.org/kamelets.enabled":     "false",
			"trait.camel.apache.org/dependencies.enabled": "false",
			"test": "1",
		}
		annotations2 := map[string]string{
			"trait.camel.apache.org/container.image":      "docker.io/jmalloc/echo-server:0.3.3",
			"trait.camel.apache.org/jvm.enabled":          "false",
			"trait.camel.apache.org/kamelets.enabled":     "false",
			"trait.camel.apache.org/dependencies.enabled": "false",
			"test": "2",
		}

		t.Run("run with initial image", func(t *testing.T) {
			expectedImage := annotations1["trait.camel.apache.org/container.image"]

			RegisterTestingT(t)

			Expect(BindKameletTo(ns, bindingID, annotations1, from, to, emptyMap, emptyMap)()).
				To(Succeed())
			Eventually(IntegrationGeneration(ns, bindingID)).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			Eventually(Integration(ns, bindingID)).Should(WithTransform(Annotations, And(
				HaveKeyWithValue("test", "1"))),
				HaveKeyWithValue("trait.camel.apache.org/container.image", expectedImage))
			Eventually(IntegrationStatusImage(ns, bindingID)).
				Should(Equal(expectedImage))
			Eventually(IntegrationPodPhase(ns, bindingID), TestTimeoutLong).
				Should(Equal(corev1.PodRunning))
			Eventually(IntegrationPodImage(ns, bindingID)).
				Should(Equal(expectedImage))
		})

		t.Run("run with new image", func(t *testing.T) {
			expectedImage := annotations2["trait.camel.apache.org/container.image"]

			RegisterTestingT(t)

			Expect(BindKameletTo(ns, bindingID, annotations2, from, to, emptyMap, emptyMap)()).
				To(Succeed())
			Eventually(IntegrationGeneration(ns, bindingID)).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			Eventually(Integration(ns, bindingID)).Should(WithTransform(Annotations, And(
				HaveKeyWithValue("test", "2"))),
				HaveKeyWithValue("trait.camel.apache.org/container.image", expectedImage))
			Eventually(IntegrationStatusImage(ns, bindingID)).
				Should(Equal(expectedImage))
			Eventually(IntegrationPodPhase(ns, bindingID), TestTimeoutLong).
				Should(Equal(corev1.PodRunning))
			Eventually(IntegrationPodImage(ns, bindingID)).
				Should(Equal(expectedImage))
		})

		// Cleanup
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
	})
}
