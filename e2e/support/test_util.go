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

package support

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"github.com/stretchr/testify/assert"
)

func init() {

}

func EqualP(expected interface{}) types.GomegaMatcher {
	return PointTo(Equal(expected))
}

func MatchFieldsP(options Options, fields Fields) types.GomegaMatcher {
	return PointTo(MatchFields(options, fields))
}

func GetEnvOrDefault(key string, deflt string) string {
	env, exists := os.LookupEnv(key)
	if exists {
		return env
	} else {
		return deflt
	}
}

func ExpectExecSucceed(t *testing.T, command *exec.Cmd) {
	t.Helper()

	var cmdOut strings.Builder
	var cmdErr strings.Builder

	defer func() {
		if t.Failed() {
			t.Logf("Output from exec command:\n%s\n", cmdOut.String())
			t.Logf("Error from exec command:\n%s\n", cmdErr.String())
		}
	}()

	session, err := gexec.Start(command, &cmdOut, &cmdErr)
	session.Wait()
	Eventually(session).Should(gexec.Exit(0))
	assert.NoError(t, err)
	assert.NotContains(t, strings.ToUpper(cmdErr.String()), "ERROR")
}

//
// Expect a command error with an exit code of 1
//
func ExpectExecError(t *testing.T, command *exec.Cmd) {
	t.Helper()

	var cmdOut strings.Builder
	var cmdErr strings.Builder

	defer func() {
		if t.Failed() {
			t.Logf("Output from exec command:\n%s\n", cmdOut.String())
			t.Logf("Error from exec command:\n%s\n", cmdErr.String())
		}
	}()

	session, err := gexec.Start(command, &cmdOut, &cmdErr)
	session.Wait()
	Eventually(session).ShouldNot(gexec.Exit(0))
	assert.NoError(t, err)
	assert.Contains(t, strings.ToUpper(cmdErr.String()), "ERROR")
}

// Clean up the cluster ready for the next set of tests
func Cleanup() {
	// Remove the locally installed operator
	UninstallAll()

	// Ensure the CRDs & ClusterRoles are reinstalled if not already
	Kamel("install", "--olm=false", "--cluster-setup").Execute()
}

// Removes all items
func UninstallAll() {
	Kamel("uninstall", "--olm=false", "--all").Execute()
}
