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

	"github.com/apache/camel-k/v2/pkg/util"
)

func init() {
	// Apply env vars for the test operator image to args if present
	imageName := os.Getenv("CAMEL_K_TEST_IMAGE_NAME")
	imageVersion := os.Getenv("CAMEL_K_TEST_IMAGE_VERSION")
	if imageName != "" || imageVersion != "" {
		if imageName == "" {
			imageName = TestImageName
		}
		if imageVersion == "" {
			imageVersion = TestImageVersion
		}
		KamelHooks = append(KamelHooks, func(args []string) []string {
			if len(args) > 0 && args[0] == "install" {
				// Prefer explicit args from test over env
				if !util.StringSliceExists(args, "--operator-image") && !util.StringContainsPrefix(args, "--operator-image=") {
					args = append(args, fmt.Sprintf("--operator-image=%s:%s", imageName, imageVersion))
				}
			}
			return args
		})
	}

}
