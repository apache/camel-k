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

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/e2e/support"
)

func TestKamelCLIHelp(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		t.Run("default help message", func(t *testing.T) {
			helpMsg := GetOutputString(Kamel("help"))
			Expect(helpMsg).To(ContainSubstring("Apache Camel K is a lightweight integration platform, born on Kubernetes"))
			Expect(helpMsg).To(ContainSubstring("Usage:"))
			Expect(helpMsg).To(ContainSubstring("Available Commands:"))
			Expect(helpMsg).To(ContainSubstring("Flags:"))
		})

		t.Run("'get' command help (short flag)", func(t *testing.T) {
			helpMsg := GetOutputString(Kamel("get", "-h"))
			Expect(helpMsg).To(ContainSubstring("Get the status of integrations deployed on Kubernetes"))
			Expect(helpMsg).To(ContainSubstring("Usage:"))
			Expect(helpMsg).To(ContainSubstring("Flags:"))
		})

		t.Run("'bind' command help (long flag)", func(t *testing.T) {
			helpMsg := GetOutputString(Kamel("bind", "--help"))
			Expect(helpMsg).To(ContainSubstring("Bind Kubernetes resources, such as Kamelets, in an integration flow."))
			Expect(helpMsg).To(ContainSubstring("kamel bind [source] [sink] ... [flags]"))
			Expect(helpMsg).To(ContainSubstring("Global Flags:"))
		})
	})
}
