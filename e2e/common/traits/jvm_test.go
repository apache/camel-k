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

package traits

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestJVMTrait(t *testing.T) {
	RegisterTestingT(t)

	// Store a configmap holding a jar
	var cmData = make(map[string][]byte)
	// We calculate the expected content
	source, err := os.ReadFile("./files/jvm/sample-1.0.jar")
	assert.Nil(t, err)
	cmData["sample-1.0.jar"] = source
	err = CreateBinaryConfigmap(ns, "my-deps", cmData)
	assert.Nil(t, err)

	t.Run("JVM trait classpath", func(t *testing.T) {
		Expect(KamelRunWithID(operatorID, ns, "./files/jvm/Classpath.java",
			"--resource", "configmap:my-deps",
			"-t", "jvm.classpath=/etc/camel/resources/my-deps/sample-1.0.jar",
		).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "classpath"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "classpath", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "classpath"), TestTimeoutShort).Should(ContainSubstring("Hello World!"))
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
