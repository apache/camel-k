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
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/cmd"
)

func TestKamelCLIConfig(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-cli-config"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		t.Run("check default namespace", func(t *testing.T) {
			_, err := os.Stat(cmd.DefaultConfigLocation)
			assert.True(t, os.IsNotExist(err), "No file at "+cmd.DefaultConfigLocation+" was expected")
			t.Cleanup(func() { os.Remove(cmd.DefaultConfigLocation) })
			Expect(Kamel("config", "--default-namespace", ns).Execute()).To(Succeed())
			_, err = os.Stat(cmd.DefaultConfigLocation)
			assert.Nil(t, err, "A file at "+cmd.DefaultConfigLocation+" was expected")
			Expect(Kamel("run", "--operator-id", operatorID, "files/yaml.yaml").Execute()).To(Succeed())

			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "yaml", v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "yaml"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			// first line of the integration logs
			logs := strings.Split(IntegrationLogs(ns, "yaml")(), "\n")[0]
			podName := IntegrationPod(ns, "yaml")().Name

			logsCLI := GetOutputStringAsync(Kamel("log", "yaml"))
			Eventually(logsCLI).Should(ContainSubstring("Monitoring pod " + podName))
			Eventually(logsCLI).Should(ContainSubstring(logs))

			// Clean up
			Expect(Kamel("delete", "--all").Execute()).To(Succeed())
		})
	})
}
