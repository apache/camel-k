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

package cli

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestMain(m *testing.M) {
	justCompile := GetEnvOrDefault("CAMEL_K_E2E_JUST_COMPILE", "false")
	if justCompile == "true" {
		os.Exit(m.Run())
	}

	g := NewGomega(func(message string, callerSkip ...int) {
		fmt.Printf("Test setup failed! - %s\n", message)
	})

	var t *testing.T

	g.Expect(TestClient(t)).ShouldNot(BeNil())

	// Install global operator for tests in this package, all tests must use this operatorID
	g.Expect(NewNamedTestNamespace(t, operatorNS, false)).ShouldNot(BeNil())
	g.Expect(CopyCamelCatalog(t, operatorNS, operatorID)).To(Succeed())
	g.Expect(KamelInstallWithIDAndKameletCatalog(t, operatorID, operatorNS, "--global", "--force").Execute()).To(Succeed())
	g.Eventually(SelectedPlatformPhase(t, operatorNS, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

	exitCode := m.Run()

	g.Expect(UninstallFromNamespace(t, operatorNS))
	g.Expect(DeleteNamespace(t, operatorNS)).To(Succeed())

	os.Exit(exitCode)
}
