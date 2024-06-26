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
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/apache/camel-k/v2/pkg/util/log"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	lock   sync.Mutex
	lockID sync.Mutex
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

// InstallOperator is in charge to install a namespaced operator. The func must be
// executed in a critical section as there may be concurrent access to it.
func InstallOperator(t *testing.T, ctx context.Context, g *WithT, ns string) {
	lock.Lock()
	defer lock.Unlock()
	KAMEL_INSTALL_REGISTRY := os.Getenv("KAMEL_INSTALL_REGISTRY")
	os.Setenv("CAMEL_K_TEST_MAKE_DIR", "../../")
	fmt.Printf("Installing namespaced operator in namespace %s with registry %s\n", ns, KAMEL_INSTALL_REGISTRY)
	ExpectExecSucceed(t, g,
		Make(t,
			fmt.Sprintf("NAMESPACE=%s", ns),
			fmt.Sprintf("REGISTRY=%s", KAMEL_INSTALL_REGISTRY),
			"install-k8s-ns"),
	)
	// Let's make sure the operator has been deployed
	g.Eventually(OperatorPod(t, ctx, ns)).ShouldNot(BeNil())
}

// InstallOperatorWitID is in charge to install a namespaced operator with a given operator ID name.
func InstallOperatorWithID(t *testing.T, ctx context.Context, g *WithT, ns, operatorID string) {
	lockID.Lock()
	defer lockID.Unlock()
	os.Setenv("CAMEL_K_TEST_MAKE_DIR", "../../")
	fmt.Printf("Setting operator ID property as %s\n", operatorID)
	ExpectExecSucceed(t, g,
		Make(t,
			fmt.Sprintf("OPERATOR_ID=%s", operatorID),
			"set-install-k8s-ns-operator-id"),
	)

	InstallOperator(t, ctx, g, ns)
}

func ExpectExecSucceed(t *testing.T, g *WithT, command *exec.Cmd) {
	t.Helper()

	var cmdOut strings.Builder
	var cmdErr strings.Builder

	defer func() {
		if t.Failed() {
			t.Logf("Output from exec command:\n%s\n", cmdOut.String())
			t.Logf("Error from exec command:\n%s\n", cmdErr.String())
		}
	}()

	RegisterTestingT(t)
	session, err := gexec.Start(command, &cmdOut, &cmdErr)
	session.Wait()
	g.Eventually(session).Should(gexec.Exit(0))
	require.NoError(t, err)
	assert.NotContains(t, strings.ToUpper(cmdErr.String()), "ERROR")
}

// Cleanup Clean up the cluster ready for the next set of tests
func Cleanup(t *testing.T, ctx context.Context) {
	// Remove the locally installed operator
	if err := UninstallAll(t, ctx); err != nil {
		log.Error(err, "Failed to uninstall Camel K")
	}

	// Ensure the CRDs & ClusterRoles are reinstalled if not already
	if err := Kamel(t, ctx, "install", "--olm=false", "--cluster-setup").Execute(); err != nil {
		log.Error(err, "Failed to perform Camel K cluster setup")
	}
}

// UninstallAll Removes all items
func UninstallAll(t *testing.T, ctx context.Context) error {
	return Kamel(t, ctx, "uninstall", "--olm=false", "--all").Execute()
}

// UninstallFromNamespace Removes operator from given namespace
func UninstallFromNamespace(t *testing.T, ctx context.Context, ns string) error {
	return Kamel(t, ctx, "uninstall", "--olm=false", "-n", ns).Execute()
}
