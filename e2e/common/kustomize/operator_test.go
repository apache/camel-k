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

package kustomize

import (
	"fmt"
	"testing"

	. "github.com/apache/camel-k/e2e/support"
	. "github.com/onsi/gomega"
)

func TestBasicOperator(t *testing.T) {
	defer Uninstall()

	WithNewTestNamespace(t, func(ns string) {
		ExecMake(t, Make("setup-cluster", fmt.Sprintf("NAMESPACE=%s", ns)))
		ExecMake(t, Make("setup", fmt.Sprintf("NAMESPACE=%s", ns)))
		ExecMake(t, Make("operator", fmt.Sprintf("NAMESPACE=%s", ns)))

		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
	})
}

func TestAlternativeImageOperator(t *testing.T) {
	defer Uninstall()

	WithNewTestNamespace(t, func(ns string) {

		ExecMake(t, Make("setup-cluster", fmt.Sprintf("NAMESPACE=%s", ns)))
		ExecMake(t, Make("setup", fmt.Sprintf("NAMESPACE=%s", ns)))

		newImage := "quay.io/kameltest/kamel-operator"
		newTag := "1.1.1"
		ExecMake(t, Make("operator", fmt.Sprintf("CUSTOM_IMAGE=%s", newImage), fmt.Sprintf("CUSTOM_VERSION=%s", newTag), fmt.Sprintf("NAMESPACE=%s", ns)))

		Eventually(OperatorImage(ns)).Should(Equal(fmt.Sprintf("%s:%s", newImage, newTag)))
	})
}

func TestGlobalOperator(t *testing.T) {
	defer Uninstall()

	WithNewTestNamespace(t, func(ns string) {
		ExecMake(t, Make("setup-cluster", fmt.Sprintf("NAMESPACE=%s", ns)))
		ExecMake(t, Make("setup", fmt.Sprintf("NAMESPACE=%s", ns), "GLOBAL=true"))

		ExecMake(t, Make("operator", fmt.Sprintf("NAMESPACE=%s", ns), "GLOBAL=true"))

		podFunc := OperatorPod(ns)
		Eventually(podFunc).Should(Not(BeNil()))
		pod := podFunc()

		containers := pod.Spec.Containers
		Expect(containers).NotTo(BeEmpty())

		envvars := containers[0].Env
		Expect(envvars).NotTo(BeEmpty())

		found := false
		for _, v := range envvars {
			if v.Name == "WATCH_NAMESPACE" {
				Expect(v.Value).To(Equal("\"\""))
				found = true
				break
			}
		}
		Expect(found).To(BeTrue())
	})
}
