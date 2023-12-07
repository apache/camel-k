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
	"testing"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestDependencyTrait(t *testing.T) {
	RegisterTestingT(t)

	t.Run("Discover dependencies", func(t *testing.T) {
		Expect(KamelRunWithID(operatorID, ns, "files/RouteDeps.java").Execute()).To(Succeed())
		// time.Sleep(1 * time.Minute)

		deps := []string{"camel:aws2-s3", "camel:caffeine", "camel:dropbox", "camel:jacksonxml", "camel:kafka", "camel:kamelet", "camel:mongodb", "camel:telegram", "camel:zipfile"}
		Eventually(IntegrationPhase(ns, "route-deps"), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseBuildingKit))
		Eventually(IntegrationStatusDependencies(ns, "route-deps"), TestTimeoutShort).Should(ContainElements(deps))
	})
	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}
