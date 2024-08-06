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

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	lock sync.Mutex
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
	InstallOperatorWithConf(t, ctx, g, ns, "", false, nil)
}

// InstallOperatorWithConf is in charge to install a namespaced operator with additional configurations.
func InstallOperatorWithConf(t *testing.T, ctx context.Context, g *WithT, ns, operatorID string, global bool, envs map[string]string) {
	lock.Lock()
	defer lock.Unlock()
	KAMEL_INSTALL_REGISTRY := os.Getenv("KAMEL_INSTALL_REGISTRY")
	args := []string{fmt.Sprintf("NAMESPACE=%s", ns)}
	if KAMEL_INSTALL_REGISTRY != "" {
		args = append(args, fmt.Sprintf("REGISTRY=%s", KAMEL_INSTALL_REGISTRY))
	}
	if operatorID != "" {
		fmt.Printf("Setting operator ID property as %s\n", operatorID)
		args = append(args, fmt.Sprintf("OPERATOR_ID=%s", operatorID))
	}
	if envs != nil {
		envArgs := make([]string, len(envs))
		for k, v := range envs {
			envArgs = append(envArgs, fmt.Sprintf("%s=%s", k, v))
		}
		if len(envArgs) > 0 {
			joinedArgs := strings.Join(envArgs, " ")
			fmt.Printf("Setting operator env vars as %s\n", joinedArgs)
			args = append(args, fmt.Sprintf("ENV=%s", joinedArgs))
		}
	}
	makeRule := "install-k8s-ns"
	if global {
		fmt.Printf("Preparing for global installation")
		makeRule = "install-k8s-global"
	}
	// TODO make this a func input variable instead
	os.Setenv("CAMEL_K_TEST_MAKE_DIR", "../../")
	ExpectExecSucceed(t, g,
		Make(t, makeRule, args...),
	)
	// Let's make sure the operator has been deployed
	g.Eventually(OperatorPod(t, ctx, ns)).ShouldNot(BeNil())
}

// UninstallOperator will delete operator resources from namespace (keeps CRDs).
func UninstallOperator(t *testing.T, ctx context.Context, g *WithT, ns, makedir string) {
	lock.Lock()
	defer lock.Unlock()
	args := []string{fmt.Sprintf("NAMESPACE=%s", ns)}

	os.Setenv("CAMEL_K_TEST_MAKE_DIR", makedir)
	ExpectExecSucceed(t, g,
		Make(t, "uninstall", args...),
	)
}

// UninstallCRDs will delete Camel K installed CRDs.
func UninstallCRDs(t *testing.T, ctx context.Context, g *WithT, makedir string) {
	lock.Lock()
	defer lock.Unlock()

	os.Setenv("CAMEL_K_TEST_MAKE_DIR", makedir)
	ExpectExecSucceed(t, g,
		Make(t, "uninstall-crds"),
	)
}

func ExpectExecSucceed(t *testing.T, g *WithT, command *exec.Cmd) {
	ExpectExecSucceedWithTimeout(t, g, command, "")
}

func ExpectExecSucceedWithTimeout(t *testing.T, g *WithT, command *exec.Cmd, timeout string) {
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
	if timeout != "" {
		session.Wait(timeout)
	} else {
		session.Wait()
	}

	g.Eventually(session).Should(gexec.Exit(0))
	require.NoError(t, err)
	assert.NotContains(t, strings.ToUpper(cmdErr.String()), "ERROR")
}
