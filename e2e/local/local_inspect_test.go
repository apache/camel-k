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
	"context"
	"io"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	. "github.com/apache/camel-k/e2e/support"
	testutil "github.com/apache/camel-k/e2e/support/util"
)

func TestLocalInspect(t *testing.T) {
	RegisterTestingT(t)

	ctx, cancel := context.WithCancel(TestContext)
	defer cancel()
	piper, pipew := io.Pipe()
	defer pipew.Close()
	defer piper.Close()

	file := testutil.MakeTempCopy(t, "files/yaml.yaml")

	kamelInspect := KamelWithContext(ctx, "local", "inspect", file)
	kamelInspect.SetOut(pipew)
	kamelInspect.SetErr(pipew)

	logScanner := testutil.NewLogScanner(ctx, piper,
		"camel:log",
		"camel:timer",
		"mvn:org.apache.camel.quarkus:camel-quarkus-yaml-dsl",
	)

	go func() {
		err := kamelInspect.Execute()
		assert.NoError(t, err)
		cancel()
	}()

	Eventually(logScanner.IsFound("camel:log"), TestTimeoutShort).Should(BeTrue())
	Eventually(logScanner.IsFound("camel:timer"), TestTimeoutShort).Should(BeTrue())
	Eventually(logScanner.IsFound("mvn:org.apache.camel.quarkus:camel-quarkus-yaml-dsl"), TestTimeoutShort).
		Should(BeTrue())
}

func TestLocalInspectWithDependencies(t *testing.T) {
	RegisterTestingT(t)

	ctx, cancel := context.WithCancel(TestContext)
	defer cancel()
	piper, pipew := io.Pipe()
	defer pipew.Close()
	defer piper.Close()

	file := testutil.MakeTempCopy(t, "files/yaml.yaml")

	kamelInspect := KamelWithContext(ctx, "local", "inspect", file, "-d", "camel-amqp", "-d", "camel-xxx")
	kamelInspect.SetOut(pipew)
	kamelInspect.SetErr(pipew)

	logScanner := testutil.NewLogScanner(ctx, piper,
		"Warning: dependency camel:xxx not found in Camel catalog",
		"camel:amqp",
		"camel:log",
		"camel:timer",
	)

	go func() {
		err := kamelInspect.Execute()
		assert.NoError(t, err)
		cancel()
	}()

	Eventually(logScanner.IsFound("Warning: dependency camel:xxx not found in Camel catalog"), TestTimeoutShort).
		Should(BeTrue())
	Eventually(logScanner.IsFound("camel:amqp"), TestTimeoutShort).Should(BeTrue())
	Eventually(logScanner.IsFound("camel:log"), TestTimeoutShort).Should(BeTrue())
	Eventually(logScanner.IsFound("camel:timer"), TestTimeoutShort).Should(BeTrue())
}
