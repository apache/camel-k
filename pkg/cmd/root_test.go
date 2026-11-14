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
	"context"
	"os"
	"testing"

	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func kamelTestPostAddCommandInit(t *testing.T, rootCmd *cobra.Command, options *RootCmdOptions) {
	t.Helper()

	err := kamelPostAddCommandInit(rootCmd, options.Flags)
	require.NoError(t, err)
}

func kamelTestPreAddCommandInitWithClient(client client.Client) (*RootCmdOptions, *cobra.Command) {
	options := RootCmdOptions{
		Context: context.Background(),
		Flags:   viper.New(),
		_client: client,
	}
	rootCmd := kamelPreAddCommandInit(&options)
	rootCmd.Run = EmptyRun
	return &options, rootCmd
}

func kamelTestPreAddCommandInit() (*RootCmdOptions, *cobra.Command) {
	fakeClient, _ := internal.NewFakeClient()
	return kamelTestPreAddCommandInitWithClient(fakeClient)
}

// We must ALWAYS clean the environment variables and viper library properties to avoid mess up with the rest of the tests.
func teardown(t *testing.T, v *viper.Viper) {
	t.Helper()
	if err := os.Setenv("KAMEL_RUN_ENVS", ""); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}
