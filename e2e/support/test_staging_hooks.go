// +build integration

// To enable compilation of this file in Goland, go to "File -> Settings -> Go -> Build Tags & Vendoring -> Build Tags -> Custom tags" and add "integration"

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
	"fmt"
	"os"
)

func init() {
	// Let's use the STAGING_RUNTIME_REPO if available
	runtimeRepo := os.Getenv("STAGING_RUNTIME_REPO")
	if runtimeRepo != "" {
		KamelHooks = append(KamelHooks, func(cmd []string) []string {
			if len(cmd) > 0 && cmd[0] == "install" {
				cmd = append(cmd, fmt.Sprintf("--maven-repository=%s", runtimeRepo))
			}
			return cmd
		})
	}

	// this hook can be also used to test a released version of the operator, e.g. the staging version during a voting period
	// Uncomment the following lines and change references to enable the hook

	//TestImageName = "docker.io/camelk/camel-k"
	//TestImageVersion = "1.0.0-M2"

	//KamelHooks = append(KamelHooks, func(cmd []string) []string {
	//	if len(cmd) > 0 && cmd[0] == "install" {
	//		cmd = append(cmd, "--operator-image=docker.io/camelk/camel-k:1.0.0-M2")
	//		cmd = append(cmd, "--maven-repository=https://repository.apache.org/content/repositories/orgapachecamel-1156")
	//	}
	//	return cmd
	//})

}
