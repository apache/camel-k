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

	"github.com/apache/camel-k/pkg/util/test"
	"github.com/spf13/cobra"

	"github.com/apache/camel-k/pkg/apis/camel/v1"

	"github.com/stretchr/testify/assert"
)

//nolint:deadcode,unused
func addTestInstallCmd(options RootCmdOptions, rootCmd *cobra.Command) *installCmdOptions {
	//add a testing version of install Command
	installCmd, installOptions := newCmdInstall(&options)
	installCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	installCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(installCmd)
	return installOptions
}

//TODO: add a proper command flags test, take inspiration by run_test.go

func TestDecodeMavenSettings(t *testing.T) {
	var err error
	var val v1.ValueSource

	//
	// ConfigMap
	//
	val, err = decodeMavenSettings("configmap:maven-settings/s.xml")
	assert.Nil(t, err)
	assert.Nil(t, val.SecretKeyRef)
	assert.Equal(t, "maven-settings", val.ConfigMapKeyRef.Name)
	assert.Equal(t, "s.xml", val.ConfigMapKeyRef.Key)

	val, err = decodeMavenSettings("configmap:maven-settings")
	assert.Nil(t, err)
	assert.Nil(t, val.SecretKeyRef)
	assert.Equal(t, "maven-settings", val.ConfigMapKeyRef.Name)
	assert.Empty(t, val.ConfigMapKeyRef.Key)

	//
	// Secrets
	//
	val, err = decodeMavenSettings("secret:maven-settings-secret/s.xml")
	assert.Nil(t, err)
	assert.Nil(t, val.ConfigMapKeyRef)
	assert.Equal(t, "maven-settings-secret", val.SecretKeyRef.Name)
	assert.Equal(t, "s.xml", val.SecretKeyRef.Key)

	val, err = decodeMavenSettings("secret:maven-settings-secret")
	assert.Nil(t, err)
	assert.Nil(t, val.ConfigMapKeyRef)
	assert.Equal(t, "maven-settings-secret", val.SecretKeyRef.Name)
	assert.Empty(t, val.SecretKeyRef.Key)

	//
	// Errors
	//
	_, err = decodeMavenSettings("something:maven-settings-secret/s.xml")
	assert.NotNil(t, err)
	_, err = decodeMavenSettings("secret")
	assert.NotNil(t, err)
}
