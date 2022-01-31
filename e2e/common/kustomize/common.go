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

package kustomize

import (
	"os/exec"
	"strings"
	"testing"

	. "github.com/apache/camel-k/e2e/support"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/stretchr/testify/assert"
)

const (
	// v1.Build,          v1.Integration
	// v1.IntegrationKit, v1.IntegrationPlatform
	// v1alpha1.Kamelet,  v1alpha1.KameletBinding
	ExpCrds = 6

	// camel-k-operator, 			 camel-k-operator-events,
	// camel-k-operator-knative, 	 camel-k-operator-leases,
	// camel-k-operator-podmonitors, camel-k-operator-strimzi,
	// camel-k-operator-keda
	ExpKubePromoteRoles = 7

	// camel-k-edit
	// camel-k-operator-custom-resource-definitions
	// camel-k-operator-bind-addressable-resolver
	ExpKubeClusterRoles = 3

	// camel-k-operator-openshift
	ExpOSPromoteRoles = 1

	// camel-k-operator-console-openshift
	ExpOSClusterRoles = 1
)

func ExecMake(t *testing.T, command *exec.Cmd) {
	var cmdOut strings.Builder
	var cmdErr strings.Builder

	defer func() {
		if t.Failed() {
			t.Logf("Output from make command:\n%s\n", cmdOut.String())
			t.Logf("Error from make command:\n%s\n", cmdErr.String())
		}
	}()

	session, err := gexec.Start(command, &cmdOut, &cmdErr)
	session.Wait()
	Eventually(session).Should(gexec.Exit(0))
	assert.Nil(t, err)
	assert.NotContains(t, cmdErr.String(), "Error")
	assert.NotContains(t, cmdErr.String(), "ERROR")
}

func Uninstall() {
	// Removes all items including CRDs and ClusterRoles
	Kamel("uninstall", "--olm=false", "--all").Execute()
}
