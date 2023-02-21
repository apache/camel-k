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
	"os"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestDefaultCamelKInstallStartup(t *testing.T) {
	RegisterTestingT(t)

	ns := "camel-k-test-integration"
	os.Setenv("CAMEL_K_TEST_NS", ns)
	Expect(NewTestNamespace(false)).ShouldNot(BeNil())
	Expect(KamelInstallWithID(ns, ns).Execute()).To(Succeed())
	Eventually(OperatorPod(ns)).ShouldNot(BeNil())
	Eventually(Platform(ns)).ShouldNot(BeNil())
	Eventually(PlatformConditionStatus(ns, v1.IntegrationPlatformConditionReady), TestTimeoutShort).
		Should(Equal(corev1.ConditionTrue))
}
