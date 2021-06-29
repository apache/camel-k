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

package common

import (
	"context"
	"testing"

	. "github.com/apache/camel-k/e2e/support"
	"github.com/apache/camel-k/pkg/cmd"
	. "github.com/onsi/gomega"
)

func TestDuplicateParameters(t *testing.T) {
	RegisterTestingT(t)

	ctx, cancel := context.WithCancel(TestContext)
	defer cancel()

	// run kamel to output the traits/configuration structure in json format to check the processed values
	// the tracing.enabled is false inside JavaDuplicateParams.java, so we can check the output of this trait as true.
	cmdParams := []string{"kamel", "run", "files/JavaDuplicateParams.java", "-o", "json", "-t", "tracing.enabled=true", "--trait", "pull-secret.enabled=true", "--property", "prop1=true", "-p", "prop2=true"}
	comm, _, _ := cmd.NewKamelWithModelineCommand(ctx, cmdParams)

	// the command is executed inside GetOutputString function
	commOutput := GetOutputString(comm)

	outParams := `"traits":{"affinity":{"configuration":{"enabled":true}},"pull-secret":{"configuration":{"enabled":true}},"tracing":{"configuration":{"enabled":true}}},"configuration":[{"type":"property","value":"prop1 = true"},{"type":"property","value":"prop2 = true"},{"type":"property","value":"foo = bar"}]}`
	Expect(commOutput).To(ContainSubstring(outParams))
}
