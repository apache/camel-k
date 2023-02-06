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

package languages

import (
	"testing"

	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestRunSimpleXmlExamples(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-examples-xml"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		t.Run("run xml", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "files/xml.xml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "xml"), TestTimeoutLong).Should(Equal(v1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "xml", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "xml"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("init run xml", func(t *testing.T) {
			RunInitGeneratedExample(t, operatorID, ns, camelv1.LanguageXML)
		})
	})
}
