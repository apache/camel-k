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

package e2e

import (
	"testing"
	"time"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	. "github.com/onsi/gomega"
)

func TestKitMainFullBuild(t *testing.T) {
	doNamedKitFullBuild(t, "main")
}

func TestKitGroovyFullBuild(t *testing.T) {
	doNamedKitFullBuild(t, "groovy")
}

func TestKitKotlinFullBuild(t *testing.T) {
	doNamedKitFullBuild(t, "kotlin")
}

func TestKitJSFullBuild(t *testing.T) {
	doNamedKitFullBuild(t, "js")
}

func TestKitXMLFullBuild(t *testing.T) {
	doNamedKitFullBuild(t, "xml")
}

func TestKitJavaFullBuild(t *testing.T) {
	doNamedKitFullBuild(t, "java")
}

func TestKitYAMLFullBuild(t *testing.T) {
	doNamedKitFullBuild(t, "yaml")
}

func TestKitKnativeFullBuild(t *testing.T) {
	doNamedKitFullBuild(t, "knative")
}

func doNamedKitFullBuild(t *testing.T, name string) {
	withNewTestNamespace(t, func(ns string) {
		Expect(kamel("install", "-n", ns, "--kit", name).Execute()).Should(BeNil())
		Eventually(build(ns, name)).ShouldNot(BeNil())
		Eventually(func() v1alpha1.BuildPhase {
			return build(ns, name)().Status.Phase
		}, 5*time.Minute).Should(Equal(v1alpha1.BuildPhaseSucceeded))
	})
}
