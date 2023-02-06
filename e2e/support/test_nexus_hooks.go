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
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	nexusNamespace   = "nexus"
	nexusService     = "nexus"
	nexusMavenMirror = "http://nexus.nexus/repository/maven-public/@id=nexus@mirrorOf=central"
)

func init() {
	// Nexus repository mirror is disabled by default for E2E testing
	nexus := os.Getenv("TEST_ENABLE_NEXUS")
	if nexus == "true" {
		svcChecked := false
		svcExists := true
		KamelHooks = append(KamelHooks, func(args []string) []string {
			if len(args) > 0 && args[0] == "install" {
				// Enable mirror only if nexus service exists
				if !svcChecked {
					svc := corev1.Service{}
					key := ctrl.ObjectKey{
						Namespace: nexusNamespace,
						Name:      nexusService,
					}

					if err := TestClient().Get(TestContext, key, &svc); err != nil {
						svcExists = false
					}
					svcChecked = true
				}
				if svcExists {
					args = append(args, fmt.Sprintf("--maven-repository=%s", nexusMavenMirror))
				}
			}
			return args
		})
	}
}
