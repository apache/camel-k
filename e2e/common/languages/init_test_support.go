// +build integration

// To enable compilation of this file in Goland, go to "File -> Settings -> Go -> Build Tags & Vendoring -> Build Tags -> Custom tags" and add "integration"

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
	"fmt"
	"path"
	"testing"

	. "github.com/apache/camel-k/e2e/support"
	"github.com/apache/camel-k/e2e/support/util"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

func RunInitGeneratedExample(lang camelv1.Language, ns string, t *testing.T) {
	RegisterTestingT(t)
	dir := util.MakeTempDir(t)
	itName := fmt.Sprintf("init%s", string(lang))          // e.g. initjava
	fileName := fmt.Sprintf("%s.%s", itName, string(lang)) // e.g. initjava.java
	file := path.Join(dir, fileName)
	Expect(Kamel("init", file).Execute()).Should(BeNil())
	Expect(Kamel("run", "-n", ns, file).Execute()).Should(BeNil())
	Eventually(IntegrationPodPhase(ns, itName), TestTimeoutMedium).Should(Equal(v1.PodRunning))
	Eventually(IntegrationLogs(ns, itName), TestTimeoutShort).Should(ContainSubstring(languageInitExpectedString(lang)))
	Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
}

func languageInitExpectedString(lang camelv1.Language) string {
	langDesc := string(lang)
	if lang == camelv1.LanguageKotlin {
		langDesc = "kotlin"
	}
	return fmt.Sprintf(" Hello Camel K from %s", langDesc)
}
