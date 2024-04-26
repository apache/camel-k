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

package cmd

import (
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/test"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
)

const cmdDebug = "debug"

// nolint: unparam
func initializeDebugCmdOptions(t *testing.T, initObjs ...runtime.Object) (*cobra.Command, *debugCmdOptions) {
	t.Helper()
	fakeClient, err := test.NewFakeClient(initObjs...)
	require.NoError(t, err)
	options, rootCmd := kamelTestPreAddCommandInitWithClient(fakeClient)
	options.Namespace = "default"
	debugCmdOptions := addTestDebugCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd, options)

	return rootCmd, debugCmdOptions
}

func addTestDebugCmd(options RootCmdOptions, rootCmd *cobra.Command) *debugCmdOptions {
	debugCmd, debugOptions := newCmdDebug(&options)
	debugCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(debugCmd)
	return debugOptions
}

func TestToggle(t *testing.T) {
	defaultIntegration, defaultKit := nominalDebugIntegration("my-it-test")

	_, debugCmdOptions := initializeDebugCmdOptions(t, &defaultIntegration, &defaultKit)
	// toggle on
	it := debugCmdOptions.toggle(&defaultIntegration, true)
	assert.Equal(t, pointer.Bool(true), it.Spec.Traits.JVM.Debug)
	// toggle off
	it = debugCmdOptions.toggle(&defaultIntegration, false)
	assert.Nil(t, it.Spec.Traits.JVM.Debug)
}

func nominalDebugIntegration(name string) (v1.Integration, v1.IntegrationKit) {
	it := v1.NewIntegration("default", name)
	it.Status.Phase = v1.IntegrationPhaseRunning
	it.Status.Image = "my-special-image"
	ik := v1.NewIntegrationKit("default", name+"-kit")
	it.Status.IntegrationKit = &corev1.ObjectReference{
		Namespace: ik.Namespace,
		Name:      ik.Name,
		Kind:      ik.Kind,
	}
	return it, *ik
}
