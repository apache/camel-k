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

	"github.com/golangplus/testing/assert"
	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/e2e/support"
	"github.com/apache/camel-k/e2e/support/util"
)

func TestLocalRun(t *testing.T) {
	RegisterTestingT(t)

	ctx, cancel := context.WithCancel(TestContext)
	defer cancel()
	piper, pipew := io.Pipe()
	defer pipew.Close()
	defer piper.Close()

	file := util.MakeTempCopy(t, "files/yaml.yaml")

	kamelRun := KamelWithContext(ctx, "local", "run", file)
	kamelRun.SetOut(pipew)

	logScanner := util.NewLogScanner(ctx, piper, "Magicstring!")

	go func() {
		err := kamelRun.Execute()
		assert.NoError(t, err)
		cancel()
	}()

	Eventually(logScanner.IsFound("Magicstring!"), TestTimeoutMedium).Should(BeTrue())
}

func TestLocalContainerRun(t *testing.T) {
	RegisterTestingT(t)

	ctx, cancel := context.WithCancel(TestContext)
	defer cancel()
	piper, pipew := io.Pipe()
	defer pipew.Close()
	defer piper.Close()

	file := util.MakeTempCopy(t, "files/yaml.yaml")

	kamelRun := KamelWithContext(ctx, "local", "run", file, "--image", "test/test", "--containerize")
	kamelRun.SetOut(pipew)

	logScanner := util.NewLogScanner(ctx, piper, "Magicstring!")

	go func() {
		err := kamelRun.Execute()
		assert.NoError(t, err)
		cancel()
	}()

	Eventually(logScanner.IsFound("Magicstring!"), TestTimeoutMedium).Should(BeTrue())
}
