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
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestStructuredLogs(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		name := RandomizedSuffixName("java")
		g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name).Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

		pod := OperatorPodGlobal(t, ctx)()
		g.Expect(pod).NotTo(BeNil())

		// pod.Namespace could be different from ns if using global operator
		fmt.Printf("Fetching logs for operator pod %s in namespace %s", pod.Name, pod.Namespace)
		logOptions := &corev1.PodLogOptions{
			Container: "camel-k-operator",
		}
		logs, err := StructuredLogs(t, ctx, pod.Namespace, pod.Name, logOptions, false)
		g.Expect(err).To(BeNil())
		g.Expect(logs).NotTo(BeEmpty())
	})
}
