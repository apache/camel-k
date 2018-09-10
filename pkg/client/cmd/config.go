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
	"os/user"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/k8sclient"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

func initKubeClient(cmd *cobra.Command) error {
	kubeconfig := cmd.Flag("config").Value.String()
	if kubeconfig == "" {
		usr, err := user.Current()
		if err != nil {
			return err
		}

		kubeconfig = filepath.Join(usr.HomeDir, ".kube", "config")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return err
	}

	k8sclient.CustomConfig = config
	return nil
}
