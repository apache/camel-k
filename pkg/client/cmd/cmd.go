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
	"github.com/spf13/cobra"
	"github.com/apache/camel-k/pkg/client/cmd/version"
	"github.com/apache/camel-k/pkg/client/cmd/run"
)

func NewKamelCommand() (*cobra.Command, error) {

	var rootCmd = cobra.Command{
		Use: "kamel",
		Short: "Kamel is a awesome client tool for running Apache Camel integrations natively on Kubernetes",
		Long: "Apache Camel K (a.k.a. Kamel) is a lightweight integration framework\nbuilt from Apache Camel that runs natively on Kubernetes and is\nspecifically designed for serverless and microservice architectures.",

	}

	var kubeconfig string
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "config", "", "Path to the config file to use for CLI requests")

	// Initialize the Kubernetes client to allow using the operator-sdk
	err := initKubeClient(&rootCmd)
	if err != nil {
		return nil, err
	}

	rootCmd.AddCommand(version.NewCmdVersion(), run.NewCmdRun())
	return &rootCmd, nil
}