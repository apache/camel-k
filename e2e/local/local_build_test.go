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
	"fmt"
	"io"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	. "github.com/apache/camel-k/e2e/support"
	testutil "github.com/apache/camel-k/e2e/support/util"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
)

// Camel version used to validate the test results
var camelVersion = getCamelVersion()

func getCamelVersion() string {
	catalog, err := camel.DefaultCatalog()
	if err != nil {
		panic(err)
	}
	version := catalog.GetCamelVersion()
	if version == "" {
		panic("unable to resolve the Camel version from catalog")
	}
	return version
}

func TestLocalBuild(t *testing.T) {
	RegisterTestingT(t)

	ctx, cancel := context.WithCancel(TestContext)
	defer cancel()
	piper, pipew := io.Pipe()
	defer pipew.Close()
	defer piper.Close()

	file := testutil.MakeTempCopy(t, "files/yaml.yaml")
	image := "test/test-" + strings.ToLower(util.RandomString(10))

	kamelBuild := KamelWithContext(ctx, "local", "build", file, "--image", image)
	kamelBuild.SetOut(pipew)
	kamelBuild.SetErr(pipew)

	msgTagged := "Successfully tagged"
	logScanner := testutil.NewLogScanner(ctx, piper, msgTagged, image)

	go func() {
		err := kamelBuild.Execute()
		assert.NoError(t, err)
		cancel()
	}()

	Eventually(logScanner.IsFound(msgTagged), TestTimeoutMedium).Should(BeTrue())
	Eventually(logScanner.IsFound(image), TestTimeoutMedium).Should(BeTrue())
	Eventually(DockerImages, TestTimeoutShort).Should(ContainSubstring(image))
}

func TestLocalBuildWithTrait(t *testing.T) {
	RegisterTestingT(t)

	ctx, cancel := context.WithCancel(TestContext)
	defer cancel()
	piper, pipew := io.Pipe()
	defer pipew.Close()
	defer piper.Close()

	file := testutil.MakeTempCopy(t, "files/trait.groovy")
	image := "test/test-" + strings.ToLower(util.RandomString(10))

	kamelBuild := KamelWithContext(ctx, "local", "build", file, "--image", image)
	kamelBuild.SetOut(pipew)
	kamelBuild.SetErr(pipew)

	msgWarning := "Warning: traits are specified but don't take effect for local run: [jolokia.enabled=true]"
	msgTagged := "Successfully tagged"
	logScanner := testutil.NewLogScanner(ctx, piper, msgWarning, msgTagged, image)

	go func() {
		err := kamelBuild.Execute()
		assert.NoError(t, err)
		cancel()
	}()

	Eventually(logScanner.IsFound(msgWarning), TestTimeoutMedium).Should(BeTrue())
	Eventually(logScanner.IsFound(msgTagged), TestTimeoutMedium).Should(BeTrue())
	Eventually(logScanner.IsFound(image), TestTimeoutMedium).Should(BeTrue())
	Eventually(DockerImages, TestTimeoutMedium).Should(ContainSubstring(image))
}

func TestLocalBuildIntegrationDirectory(t *testing.T) {
	RegisterTestingT(t)

	ctx, cancel := context.WithCancel(TestContext)
	defer cancel()

	file := testutil.MakeTempCopy(t, "files/yaml.yaml")
	dir := testutil.MakeTempDir(t)

	kamelBuild := KamelWithContext(ctx, "local", "build", file, "--integration-directory", dir)

	go func() {
		err := kamelBuild.Execute()
		assert.NoError(t, err)
	}()

	Eventually(dir+"/dependencies", TestTimeoutShort).Should(BeADirectory())
	Eventually(dependency(dir, "org.apache.camel.camel-timer-%s.jar", camelVersion), TestTimeoutShort).Should(BeAnExistingFile())
	Eventually(dependency(dir, "org.apache.camel.camel-log-%s.jar", camelVersion), TestTimeoutShort).Should(BeAnExistingFile())
	Eventually(dir+"/properties", TestTimeoutShort).Should(BeADirectory())
	Eventually(dir+"/routes/yaml.yaml", TestTimeoutShort).Should(BeAnExistingFile())

	// Run the same command again on the existing directory
	err := kamelBuild.Execute()
	assert.NoError(t, err)
}

func TestLocalBuildIntegrationDirectoryWithSpaces(t *testing.T) {
	RegisterTestingT(t)

	ctx, cancel := context.WithCancel(TestContext)
	defer cancel()

	file := testutil.MakeTempCopy(t, "files/yaml.yaml")
	dir := testutil.MakeTempDir(t) + " - Camel rocks!"

	kamelBuild := KamelWithContext(ctx, "local", "build", file, "--integration-directory", dir)

	go func() {
		err := kamelBuild.Execute()
		assert.NoError(t, err)
		cancel()
	}()

	Eventually(dir+"/dependencies", TestTimeoutShort).Should(BeADirectory())
	Eventually(dependency(dir, "org.apache.camel.camel-timer-%s.jar", camelVersion), TestTimeoutShort).Should(BeAnExistingFile())
	Eventually(dependency(dir, "org.apache.camel.camel-log-%s.jar", camelVersion), TestTimeoutShort).Should(BeAnExistingFile())
	Eventually(dir+"/properties", TestTimeoutShort).Should(BeADirectory())
	Eventually(dir+"/routes/yaml.yaml", TestTimeoutShort).Should(BeAnExistingFile())
}

func TestLocalBuildIntegrationDirectoryWithMultiBytes(t *testing.T) {
	RegisterTestingT(t)

	ctx, cancel := context.WithCancel(TestContext)
	defer cancel()

	file := testutil.MakeTempCopy(t, "files/yaml.yaml")
	dir := testutil.MakeTempDir(t) + "_らくだ" // Camel

	kamelBuild := KamelWithContext(ctx, "local", "build", file, "--integration-directory", dir)

	go func() {
		err := kamelBuild.Execute()
		assert.NoError(t, err)
		cancel()
	}()

	Eventually(dir+"/dependencies", TestTimeoutShort).Should(BeADirectory())
	Eventually(dependency(dir, "org.apache.camel.camel-timer-%s.jar", camelVersion), TestTimeoutShort).Should(BeAnExistingFile())
	Eventually(dependency(dir, "org.apache.camel.camel-log-%s.jar", camelVersion), TestTimeoutShort).Should(BeAnExistingFile())
	Eventually(dir+"/properties", TestTimeoutShort).Should(BeADirectory())
	Eventually(dir+"/routes/yaml.yaml", TestTimeoutShort).Should(BeAnExistingFile())
}

func TestLocalBuildDependenciesOnly(t *testing.T) {
	RegisterTestingT(t)

	ctx, cancel := context.WithCancel(TestContext)
	defer cancel()

	file := testutil.MakeTempCopy(t, "files/yaml.yaml")
	dir := testutil.MakeTempDir(t)

	kamelBuild := KamelWithContext(ctx, "local", "build", file, "--integration-directory", dir, "--dependencies-only", "-d", "camel-amqp")

	go func() {
		err := kamelBuild.Execute()
		assert.NoError(t, err)
		cancel()
	}()

	Eventually(dir+"/dependencies", TestTimeoutShort).Should(BeADirectory())
	Eventually(dependency(dir, "org.apache.camel.camel-timer-%s.jar", camelVersion), TestTimeoutShort).Should(BeAnExistingFile())
	Eventually(dependency(dir, "org.apache.camel.camel-log-%s.jar", camelVersion), TestTimeoutShort).Should(BeAnExistingFile())
	Eventually(dependency(dir, "org.apache.camel.camel-amqp-%s.jar", camelVersion), TestTimeoutShort).Should(BeAnExistingFile())
	Expect(dir + "/properties").ShouldNot(BeADirectory())
	Expect(dir + "/routes/yaml.yaml").ShouldNot(BeAnExistingFile())
}

func TestLocalBuildModelineDependencies(t *testing.T) {
	RegisterTestingT(t)

	ctx, cancel := context.WithCancel(TestContext)
	defer cancel()

	file := testutil.MakeTempCopy(t, "files/dependency.groovy")
	dir := testutil.MakeTempDir(t)

	kamelBuild := KamelWithContext(ctx, "local", "build", file, "--integration-directory", dir, "-d", "camel-amqp")

	go func() {
		err := kamelBuild.Execute()
		assert.NoError(t, err)
	}()

	Eventually(dir+"/dependencies", TestTimeoutShort).Should(BeADirectory())
	Eventually(dependency(dir, "org.apache.camel.camel-timer-%s.jar", camelVersion), TestTimeoutShort).Should(BeAnExistingFile())
	Eventually(dependency(dir, "org.apache.camel.camel-log-%s.jar", camelVersion), TestTimeoutShort).Should(BeAnExistingFile())
	Eventually(dependency(dir, "org.apache.camel.camel-amqp-%s.jar", camelVersion), TestTimeoutShort).Should(BeAnExistingFile())
	// camel dependency
	Eventually(dependency(dir, "org.apache.camel.camel-twitter-%s.jar", camelVersion), TestTimeoutMedium).Should(BeAnExistingFile())
	// mvn dependency
	Eventually(dependency(dir, "com.google.guava.guava-31.1-jre.jar"), TestTimeoutMedium).Should(BeAnExistingFile())
	// github dependency
	Eventually(dependency(dir, "com.github.squakez.samplejp-v1.0.jar"), TestTimeoutMedium).Should(BeAnExistingFile())
	Eventually(dir+"/routes/dependency.groovy", TestTimeoutShort).Should(BeAnExistingFile())
}

func dependency(dir string, jar string, params ...interface{}) string {
	params = append([]interface{}{dir}, params...)
	return fmt.Sprintf("%s/dependencies/"+jar, params...)
}
