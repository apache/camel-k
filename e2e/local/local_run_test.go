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
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	. "github.com/apache/camel-k/e2e/support"
	testutil "github.com/apache/camel-k/e2e/support/util"
	"github.com/apache/camel-k/pkg/util"
)

func TestLocalRun(t *testing.T) {
	RegisterTestingT(t)

	ctx, cancel := context.WithCancel(TestContext)
	defer cancel()
	piper, pipew := io.Pipe()
	defer pipew.Close()
	defer piper.Close()

	file := testutil.MakeTempCopy(t, "files/yaml.yaml")

	kamelRun := KamelWithContext(ctx, "local", "run", file)
	kamelRun.SetOut(pipew)

	logScanner := testutil.NewLogScanner(ctx, piper, "Magicstring!")

	go func() {
		err := kamelRun.Execute()
		assert.NoError(t, err)
		cancel()
	}()

	Eventually(logScanner.IsFound("Magicstring!"), TestTimeoutMedium).Should(BeTrue())
}

func TestLocalRunContainerize(t *testing.T) {
	RegisterTestingT(t)

	ctx, cancel := context.WithCancel(TestContext)
	defer cancel()
	piper, pipew := io.Pipe()
	defer pipew.Close()
	defer piper.Close()

	file := testutil.MakeTempCopy(t, "files/yaml.yaml")
	image := "test/test-" + strings.ToLower(util.RandomString(10))

	kamelRun := KamelWithContext(ctx, "local", "run", file, "--image", image, "--containerize")
	kamelRun.SetOut(pipew)

	logScanner := testutil.NewLogScanner(ctx, piper, "Magicstring!")

	defer StopDockerContainers()
	go func() {
		err := kamelRun.Execute()
		assert.NoError(t, err)
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

	kamelBuild := KamelWithContext(ctx1, "local", "build", file, "--integration-directory", dir)

	go func() {
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

	kamelRun := KamelWithContext(ctx2, "local", "run", "--integration-directory", dir)
	kamelRun.SetOut(pipew)

	logScanner := testutil.NewLogScanner(ctx2, piper, "Magicstring!")

	go func() {
		err := kamelRun.Execute()
		assert.NoError(t, err)
		cancel2()
	}()

	Eventually(dir+"/../quarkus", TestTimeoutShort).Should(BeADirectory())
	Eventually(dir+"/../app", TestTimeoutShort).Should(BeADirectory())
	Eventually(dir+"/../lib", TestTimeoutShort).Should(BeADirectory())
	Eventually(logScanner.IsFound("Magicstring!"), TestTimeoutMedium).Should(BeTrue())
}
