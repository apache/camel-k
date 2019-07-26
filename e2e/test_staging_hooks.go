// +build integration knative

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

package e2e

func init() {
	// this hook can be used to test a released version of the operator, e.g. the staging version during a voting period
	// Uncomment the following lines and change references to enable the hook
	kamelHooks = append(kamelHooks, func(cmd []string) []string {
		//if len(cmd) > 0 && cmd[0] == "install" {
		//	cmd = append(cmd, "--operator-image=docker.io/camelk/camel-k:1.0.0-M1")
		//	cmd = append(cmd, "--maven-repository=https://repository.apache.org/content/repositories/orgapachecamel-1145")
		//}
		return cmd
	})

}
