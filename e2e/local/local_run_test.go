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
	"strings"
	"sync"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	. "github.com/apache/camel-k/e2e/support"
	testutil "github.com/apache/camel-k/e2e/support/util"
	"github.com/apache/camel-k/pkg/util"
)

func TestLocalRun(t *testing.T) {
	RegisterTestingT(t)

	ctx, cancel := context.WithTimeout(TestContext, TestTimeoutMedium)
	defer cancel()
	piper, pipew := io.Pipe()
	defer pipew.Close()
	defer piper.Close()

	file := testutil.MakeTempCopy(t, "files/yaml.yaml")

	kamelRun := kamelWithContext(ctx, "local", "run", file)
	kamelRun.SetOut(pipew)
	kamelRun.SetErr(pipew)

	logScanner := testutil.NewLogScanner(ctx, piper, "Magicstring!")

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		_ = kamelRun.Execute()
		cancel()
	}()

	Eventually(logScanner.IsFound("Magicstring!"), TestTimeoutMedium).Should(BeTrue())
}

func TestLocalRunWithDependencies(t *testing.T) {
	RegisterTestingT(t)

	ctx, cancel := context.WithTimeout(TestContext, TestTimeoutMedium)
	defer cancel()
	piper, pipew := io.Pipe()
	defer pipew.Close()
	defer piper.Close()

	file := testutil.MakeTempCopy(t, "files/dependency.groovy")

	kamelRun := kamelWithContext(ctx, "local", "run", file, "-d", "camel-amqp")
	kamelRun.SetOut(pipew)
	kamelRun.SetErr(pipew)

	logScanner := testutil.NewLogScanner(ctx, piper, "Magicstring!")

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		_ = kamelRun.Execute()
		cancel()
	}()

	Eventually(logScanner.IsFound("Magicstring!"), TestTimeoutMedium).Should(BeTrue())
}

func TestLocalRunWithInvalidDependency(t *testing.T) {
	RegisterTestingT(t)

	ctx, cancel := context.WithTimeout(TestContext, TestTimeoutMedium)
	defer cancel()
	piper, pipew := io.Pipe()
	defer pipew.Close()
	defer piper.Close()

	file := testutil.MakeTempCopy(t, "files/yaml.yaml")

	kamelRun := KamelWithContext(ctx, "local", "run", file,
		"-d", "camel-xxx",
		"-d", "mvn:org.apache.camel:camel-http:3.18.0",
		"-d", "mvn:org.apache.camel.quarkus:camel-quarkus-netty:2.11.0")
	kamelRun.SetOut(pipew)
	kamelRun.SetErr(pipew)

	warn1 := "Warning: dependency camel:xxx not found in Camel catalog"
	warn2 := "Warning: do not use mvn:org.apache.camel:camel-http:3.18.0. Use camel:http instead"
	warn3 := "Warning: do not use mvn:org.apache.camel.quarkus:camel-quarkus-netty:2.11.0. Use camel:netty instead"
	logScanner := testutil.NewLogScanner(ctx, piper, warn1, warn2, warn3)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		err := kamelRun.Execute()
		assert.Error(t, err)
		cancel()
	}()

	Eventually(logScanner.IsFound(warn1), TestTimeoutShort).Should(BeTrue())
	Eventually(logScanner.IsFound(warn2), TestTimeoutShort).Should(BeTrue())
	Eventually(logScanner.IsFound(warn3), TestTimeoutShort).Should(BeTrue())
	wg.Wait()
}

func TestLocalRunContainerize(t *testing.T) {
	RegisterTestingT(t)

	ctx, cancel := context.WithTimeout(TestContext, TestTimeoutMedium)
	defer cancel()
	piper, pipew := io.Pipe()
	defer pipew.Close()
	defer piper.Close()

	file := testutil.MakeTempCopy(t, "files/yaml.yaml")
	image := "test/test-" + strings.ToLower(util.RandomString(10))

	kamelRun := kamelWithContext(ctx, "local", "run", file, "--image", image, "--containerize")
	kamelRun.SetOut(pipew)
	kamelRun.SetErr(pipew)

	logScanner := testutil.NewLogScanner(ctx, piper, "Magicstring!")

	var wg sync.WaitGroup
	wg.Add(1)

	defer StopDockerContainers()
	go func() {
		defer wg.Done()
		_ = kamelRun.Execute()
		cancel()
	}()

	Eventually(logScanner.IsFound("Magicstring!"), TestTimeoutMedium).Should(BeTrue())
	Eventually(DockerImages, TestTimeoutShort).Should(ContainSubstring(image))
}

func TestLocalRunIntegrationDirectory(t *testing.T) {
	RegisterTestingT(t)

	ctx1, cancel1 := context.WithCancel(TestContext)
	defer cancel1()

	file := testutil.MakeTempCopy(t, "files/yaml.yaml")
	dir := testutil.MakeTempDir(t)

	kamelBuild := kamelWithContext(ctx1, "local", "build", file, "--integration-directory", dir)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		err := kamelBuild.Execute()
		assert.NoError(t, err)
		cancel1()
	}()

	Eventually(dir+"/dependencies", TestTimeoutShort).Should(BeADirectory())
	Eventually(dir+"/properties", TestTimeoutShort).Should(BeADirectory())
	Eventually(dir+"/routes/yaml.yaml", TestTimeoutShort).Should(BeAnExistingFile())

	ctx2, cancel2 := context.WithCancel(TestContext)
	defer cancel2()
	piper, pipew := io.Pipe()
	defer pipew.Close()
	defer piper.Close()

	kamelRun := kamelWithContext(ctx2, "local", "run", "--integration-directory", dir)
	kamelRun.SetOut(pipew)
	kamelRun.SetErr(pipew)

	logScanner := testutil.NewLogScanner(ctx2, piper, "Magicstring!")

	var wg2 sync.WaitGroup
	wg2.Add(1)

	go func() {
		defer wg2.Done()

		_ = kamelRun.Execute()
		cancel2()
	}()

	Eventually(dir+"/../quarkus", TestTimeoutShort).Should(BeADirectory())
	Eventually(dir+"/../app", TestTimeoutShort).Should(BeADirectory())
	Eventually(dir+"/../lib", TestTimeoutShort).Should(BeADirectory())
	Eventually(logScanner.IsFound("Magicstring!"), TestTimeoutMedium).Should(BeTrue())
}
