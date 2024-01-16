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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	. "github.com/apache/camel-k/v2/e2e/support"
)

func TestKameletClasspathLoading(t *testing.T) {
	RegisterTestingT(t)

	// Store a configmap on the cluster
	var cmData = make(map[string]string)
	cmData["my-timer-source.kamelet.yaml"] = `
# ---------------------------------------------------------------------------
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# ---------------------------------------------------------------------------

apiVersion: camel.apache.org/v1
kind: Kamelet
metadata:
  name: my-timer-source
  annotations:
    camel.apache.org/kamelet.support.level: "Preview"
    camel.apache.org/catalog.version: "0.3.0"
    camel.apache.org/kamelet.icon: data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0idXRmLTgiPz4NCjwhLS0gU3ZnIFZlY3RvciBJY29ucyA6IGh0dHA6Ly93d3cub25saW5ld2ViZm9udHMuY29tL2ljb24gLS0+DQo8IURPQ1RZUEUgc3ZnIFBVQkxJQyAiLS8vVzNDLy9EVEQgU1ZHIDEuMS8vRU4iICJodHRwOi8vd3d3LnczLm9yZy9HcmFwaGljcy9TVkcvMS4xL0RURC9zdmcxMS5kdGQiPg0KPHN2ZyB2ZXJzaW9uPSIxLjEiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiIHg9IjBweCIgeT0iMHB4IiB2aWV3Qm94PSIwIDAgMTAwMCAxMDAwIiBlbmFibGUtYmFja2dyb3VuZD0ibmV3IDAgMCAxMDAwIDEwMDAiIHhtbDpzcGFjZT0icHJlc2VydmUiPg0KPG1ldGFkYXRhPiBTdmcgVmVjdG9yIEljb25zIDogaHR0cDovL3d3dy5vbmxpbmV3ZWJmb250cy5jb20vaWNvbiA8L21ldGFkYXRhPg0KPGc+PGcgdHJhbnNmb3JtPSJ0cmFuc2xhdGUoMC4wMDAwMDAsNTExLjAwMDAwMCkgc2NhbGUoMC4xMDAwMDAsLTAuMTAwMDAwKSI+PHBhdGggZD0iTTM4ODguMSw0Nzc0Ljl2LTIzNS4xaDQxMS40aDQxNC4zbC04LjgtMzI5LjFsLTguOC0zMzJsLTExNy41LTguOGMtMjI5LjItMTQuNy02MjAtOTkuOS05MjUuNi0xOTYuOUMyMjU3LjQsMzIyMC42LDExNjcuMiwyMDY1LjgsODAyLjksNjQ5LjZjLTUxMS4zLTE5ODYuMywzODQuOS00MDAyLDIyMDYuNy00OTY1LjhjMzAyLjYtMTYxLjYsNzU4LjEtMzIwLjIsMTE1NC44LTQwNS41YzQyNi4xLTkxLjEsMTI1MS43LTkxLjEsMTY4MC43LDBjMTc2OC45LDM4MiwzMDQ0LjEsMTY1Ny4yLDM0MjYuMSwzNDI2LjFjOTEuMSw0MjYuMSw5MS4xLDEyNTQuNiwwLDE2NzcuOGMtNDIwLjIsMTk0Mi4yLTE5MzYuNCwzMzAyLjYtMzg5MC4zLDM0OTYuNmwtMTk5LjgsMjAuNnYzMjAuM3YzMjAuM2g0MTEuNGg0MTEuNHYyMzUuMVY1MDEwSDQ5NDUuOUgzODg4LjFWNDc3NC45eiBNNTc1My45LDMzNDkuOWM3NzguNy0xNjEuNiwxNDE5LjItNTA4LjMsMTk4My40LTEwNzIuNWM1NjQuMi01NjEuMiw4ODcuNC0xMTU3LjcsMTA2MC43LTE5NDIuMmM5OS45LTQzNy44LDk5LjktMTE0MywzLTE1ODAuOEM4NTYzLTIzMDYuNCw3OTY2LjUtMzE1OC41LDcwNDMuOS0zNzUyYy0zMzUtMjE0LjUtNzg3LjUtMzk2LjctMTI0OC44LTQ5OS41Yy00MzcuOC05Ny0xMTQzLTk3LTE1ODAuOCwyLjljLTc4NC41LDE3My4zLTEzODEsNDk2LjYtMTk0Mi4yLDEwNjAuN2MtNTcwLDU2Ny4xLTkwNy45LDExOTguOC0xMDc4LjQsMTk5OC4xYy03My41LDM0Ni43LTczLjUsMTEyMi40LDAsMTQ2OS4yYzE3MC40LDc5OS4yLDUwOC4zLDE0MzEsMTA3OC40LDE5OThDMjg5NSwyOTAwLjMsMzYzOC40LDMyNzMuNSw0NDkzLjQsMzM5MUM0Nzc4LjQsMzQzMi4xLDU0NzQuOCwzNDA4LjYsNTc1My45LDMzNDkuOXoiLz48cGF0aCBkPSJNNDcxMC44LDEzNzUuM1YyMDUuOUw0NTUyLjIsNjcuOGMtMzE3LjMtMjc5LjEtMzQwLjgtNjc4LjctNTUuOC05OTMuMWMyODcuOS0zMjAuMyw2OTMuNC0zMTcuMywxMDEzLjcsNS45bDE3MC40LDE3MC40aDEwNDMuMWgxMDQzLjFWLTUxNFYtMjc5SDY3MjkuNUg1NjkyLjJsLTQ5LjksMTE0LjZjLTU4LjgsMTMyLjItMjUyLjcsMzE3LjMtMzc2LjEsMzYxLjRsLTg1LjIsMjkuNHYxMTU3Ljd2MTE1Ny43aC0yMzUuMWgtMjM1LjFWMTM3NS4zeiBNNTE2Ni4zLTI5My42YzE0Ni45LTE0NCw0NC4xLTM5Ni43LTE2MS42LTM5Ni43Yy01NS44LDAtMTE3LjUsMjYuNC0xNjEuNiw3My40Yy00Nyw0NC4xLTczLjUsMTA1LjgtNzMuNSwxNjEuNnMyNi40LDExNy41LDczLjUsMTYxLjZjNDQuMSw0NywxMDUuOCw3My41LDE2MS42LDczLjVDNTA2MC41LTIyMC4yLDUxMjIuMi0yNDYuNyw1MTY2LjMtMjkzLjZ6Ii8+PC9nPjwvZz4NCjwvc3ZnPg==
    camel.apache.org/provider: "Apache Software Foundation"
    camel.apache.org/kamelet.group: "Timer"
  labels:
    camel.apache.org/kamelet.type: source
    camel.apache.org/kamelet.verified: "true"
spec:
  definition:
    title: Timer Source
    description: Produces periodic events with a custom payload.
    required:
      - message
    type: object
    properties:
      period:
        title: Period
        description: The interval between two events in milliseconds
        type: integer
        default: 1000
      message:
        title: Message
        description: The message to generate
        type: string
        example: hello world
      contentType:
        title: Content Type
        description: The content type of the message being generated
        type: string
        default: text/plain
  dependencies:
    - "camel:core"
    - "camel:timer"
    - "camel:kamelet"
  template:
    from:
      uri: timer:tick
      parameters:
        period: "{{period}}"
      steps:
        - setBody:
            constant: "{{message}}"
        - setHeader:
            name: "Content-Type"
            constant: "{{contentType}}"
        - to: kamelet:sink
`
	CreatePlainTextConfigmap(ns, "my-kamelet-cm", cmData)

	// Basic
	t.Run("test basic case", func(t *testing.T) {
		Expect(KamelRunWithID(operatorID, ns, "files/TimerKameletIntegration.java", "-t", "kamelets.enabled=false",
			"--resource", "configmap:my-kamelet-cm@/kamelets",
			"-p camel.component.kamelet.location=file:/kamelets",
			"-d", "camel:yaml-dsl",
			// kamelet dependencies
			"-d", "camel:timer").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "timer-kamelet-integration"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, "timer-kamelet-integration")).Should(ContainSubstring("important message"))

		// check integration schema does not contains unwanted default trait value.
		Eventually(UnstructuredIntegration(ns, "timer-kamelet-integration")).ShouldNot(BeNil())
		unstructuredIntegration := UnstructuredIntegration(ns, "timer-kamelet-integration")()
		kameletsTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "kamelets")
		Expect(kameletsTrait).ToNot(BeNil())
		Expect(len(kameletsTrait)).To(Equal(1))
		Expect(kameletsTrait["enabled"]).To(Equal(false))
	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}
