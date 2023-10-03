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

package support

import (
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestCommonCamelKInstallStartup(t *testing.T) {
	RegisterTestingT(t)

	ns := NewTestNamespace(false)
	Expect(ns).ShouldNot(BeNil())
	// the namespace is dynamic if there is some collision
	// we store this value as it will be used for cleaning in the teardown process
	SaveCIProcessID(ns.GetName())
	// fail fast if something did not work writing the resource
	Expect(GetCIProcessID()).ShouldNot(Equal(""))

	Expect(KamelInstallWithIDAndKameletCatalog(ns.GetName(), ns.GetName()).Execute()).To(Succeed())
	Eventually(OperatorPod(ns.GetName())).ShouldNot(BeNil())
	Eventually(Platform(ns.GetName())).ShouldNot(BeNil())
	Eventually(PlatformConditionStatus(ns.GetName(), v1.IntegrationPlatformConditionTypeCreated), TestTimeoutShort).
		Should(Equal(corev1.ConditionTrue))
}
