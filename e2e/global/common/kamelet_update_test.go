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
	"testing"
	"time"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/e2e/support"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

const customLabel = "custom-label"

func TestBundleKameletUpdate(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(createBundleKamelet(ns, "http-sink")()).To(Succeed()) // Going to be replaced
		Expect(createUserKamelet(ns, "user-sink")()).To(Succeed())   // Left intact by the operator

		Expect(KamelInstall(ns).Execute()).To(Succeed())

		Eventually(Kamelet("http-sink", ns)).
			Should(WithTransform(KameletLabels, HaveKeyWithValue(customLabel, "true")))
		Consistently(Kamelet("user-sink", ns), 5*time.Second, 1*time.Second).
			Should(WithTransform(KameletLabels, HaveKeyWithValue(customLabel, "true")))

		// Cleanup
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
	})
}

func createBundleKamelet(ns string, name string) func() error {
	flow := map[string]interface{}{
		"from": map[string]interface{}{
			"uri": "kamelet:source",
		},
	}

	labels := map[string]string{
		customLabel:                  "true",
		v1alpha1.KameletBundledLabel: "true",
	}
	return CreateKamelet(ns, name, flow, nil, labels)
}

func createUserKamelet(ns string, name string) func() error {
	flow := map[string]interface{}{
		"from": map[string]interface{}{
			"uri": "kamelet:source",
		},
	}

	labels := map[string]string{
		customLabel: "true",
	}
	return CreateKamelet(ns, name, flow, nil, labels)
}
