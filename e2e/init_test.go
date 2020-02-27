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
	"fmt"
	"path"
	"testing"

	"github.com/apache/camel-k/e2e/util"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

func TestRunInitGeneratedExamples(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {
		Expect(kamel("install", "-n", ns).Execute()).Should(BeNil())

		for _, lang := range camelv1.Languages {
			t.Run("init run "+string(lang), func(t *testing.T) {
				RegisterTestingT(t)
				dir := util.MakeTempDir(t)
				itName := fmt.Sprintf("init%s", string(lang))          // e.g. initjava
				fileName := fmt.Sprintf("%s.%s", itName, string(lang)) // e.g. initjava.java
				file := path.Join(dir, fileName)
				Expect(kamel("init", file).Execute()).Should(BeNil())
				Expect(kamel("run", "-n", ns, file).Execute()).Should(BeNil())
				Eventually(integrationPodPhase(ns, itName), testTimeoutMedium).Should(Equal(v1.PodRunning))
				Eventually(integrationLogs(ns, itName), testTimeoutShort).Should(ContainSubstring(languageInitExpectedString(lang)))
				Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
			})
		}
	})
}

func languageInitExpectedString(lang camelv1.Language) string {
	langDesc := string(lang)
	if lang == camelv1.LanguageKotlin {
		langDesc = "kotlin"
	}
	return fmt.Sprintf(" Hello Camel K from %s", langDesc)
}
