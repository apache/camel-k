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

package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func main() {

	cmd := exec.Command("mvn", "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = fmt.Sprintf("Error: %v\n", err)
		return
	}

	for _, ln := range strings.Split(string(output), "\n") {
		switch {
		case strings.Contains(ln, "Apache Maven"):
			fmt.Printf("%v\n", ln)
			versionRegex := regexp.MustCompile(`Apache Maven ([1-9]+)(\.([0-9]+)){2,3}`)
			matches := versionRegex.FindStringSubmatch(ln)
			if len(matches) < 2 {
				_ = fmt.Sprintf("Unable to determine Apache Maven version: %s\n", ln)
				return
			}
		case strings.Contains(ln, "Java version"):
			fmt.Printf("%v\n", ln)
			versionRegex := regexp.MustCompile(`version: ([1-9]+)(\.([0-9]+)){2,3}`)
			matches := versionRegex.FindStringSubmatch(ln)
			if len(matches) < 2 {
				_ = fmt.Sprintf("Unable to determine Java version: %s\n", ln)
				return
			}
			majorVersion, err := strconv.Atoi(matches[1])
			if err != nil {
				_ = fmt.Sprintf("Error parsing Java version: %s - %v\n", ln, err)
				return
			}
			if majorVersion < 17 {
				_ = fmt.Sprintf("JDK version is below 17: %s\n", ln)
				return
			}
		}
	}
}
