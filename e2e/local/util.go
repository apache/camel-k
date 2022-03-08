//go:build integration
// +build integration

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

package local

import (
	"os/exec"
	"strings"

	. "github.com/apache/camel-k/e2e/support"
)

func Docker(args ...string) string {
	cmd := exec.CommandContext(TestContext, "docker", args...)
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	return string(out)
}

func DockerImages() string {
	return Docker("images")
}

func StopDockerContainers() {
	output := Docker("container", "list", "--quiet")
	containers := strings.Fields(output)
	if len(containers) > 0 {
		args := append([]string{"stop"}, containers...)
		Docker(args...)
	}
}
