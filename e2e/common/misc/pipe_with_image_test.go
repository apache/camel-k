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

package misc

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
)

func TestPipeWithImage(t *testing.T) {
	RegisterTestingT(t)

	bindingID := "with-image-binding"

	t.Run("run with initial image", func(t *testing.T) {
		expectedImage := "docker.io/jmalloc/echo-server:0.3.2"

		Expect(KamelBindWithID(operatorID, ns,
			"my-own-timer-source",
			"my-own-log-sink",
			"--annotation", "trait.camel.apache.org/container.image="+expectedImage,
			"--annotation", "trait.camel.apache.org/jvm.enabled=false",
			"--annotation", "trait.camel.apache.org/kamelets.enabled=false",
			"--annotation", "trait.camel.apache.org/dependencies.enabled=false",
			"--annotation", "test=1",
			"--name", bindingID,
		).Execute()).To(Succeed())

		Eventually(IntegrationGeneration(ns, bindingID)).
			Should(gstruct.PointTo(BeNumerically("==", 1)))
		Eventually(Integration(ns, bindingID)).Should(WithTransform(Annotations, And(
			HaveKeyWithValue("test", "1"),
			HaveKeyWithValue("trait.camel.apache.org/container.image", expectedImage),
		)))
		Eventually(IntegrationStatusImage(ns, bindingID)).
			Should(Equal(expectedImage))
		Eventually(IntegrationPodPhase(ns, bindingID), TestTimeoutLong).
			Should(Equal(corev1.PodRunning))
		Eventually(IntegrationPodImage(ns, bindingID)).
			Should(Equal(expectedImage))
	})

	t.Run("run with new image", func(t *testing.T) {
		expectedImage := "docker.io/jmalloc/echo-server:0.3.3"

		Expect(KamelBindWithID(operatorID, ns,
			"my-own-timer-source",
			"my-own-log-sink",
			"--annotation", "trait.camel.apache.org/container.image="+expectedImage,
			"--annotation", "trait.camel.apache.org/jvm.enabled=false",
			"--annotation", "trait.camel.apache.org/kamelets.enabled=false",
			"--annotation", "trait.camel.apache.org/dependencies.enabled=false",
			"--annotation", "test=2",
			"--name", bindingID,
		).Execute()).To(Succeed())
		Eventually(IntegrationGeneration(ns, bindingID)).
			Should(gstruct.PointTo(BeNumerically("==", 1)))
		Eventually(Integration(ns, bindingID)).Should(WithTransform(Annotations, And(
			HaveKeyWithValue("test", "2"),
			HaveKeyWithValue("trait.camel.apache.org/container.image", expectedImage),
		)))
		Eventually(IntegrationStatusImage(ns, bindingID)).
			Should(Equal(expectedImage))
		Eventually(IntegrationPodPhase(ns, bindingID), TestTimeoutLong).
			Should(Equal(corev1.PodRunning))
		Eventually(IntegrationPodImage(ns, bindingID)).
			Should(Equal(expectedImage))
	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}
