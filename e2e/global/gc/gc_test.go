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

package registry

import (
	"context"
	"io"
	"testing"

	testutil "github.com/apache/camel-k/e2e/support/util"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	. "github.com/apache/camel-k/e2e/support"
)

func TestGCIntegrationKits(t *testing.T) {
	// First Test is where we don't have delete rights to the Image Repository
	// Simple GC use case that only deletes IntegrationKits that aren't referenced by any Integration
	WithNewTestNamespace(t, func(ns string) {
		t.Run("Garbage Collect IntegrationKits not used", func(t *testing.T) {
			// Create a dummy integration kit
			name := "foobar"
			Expect(KamelWithContext(TestContext, "kit", "create", "foobar", "-n", ns).Execute()).To(Succeed())
			Eventually(Kit(ns, name)().Name).Should(Equal(name))

			// check that gc deletes it
			ctx, cancel := context.WithCancel(TestContext)
			defer cancel()
			piper, pipew := io.Pipe()
			defer pipew.Close()
			defer piper.Close()

			kamelBuild := KamelWithContext(TestContext, "gc", "-y", "-n", ns)
			kamelBuild.SetOut(pipew)
			kamelBuild.SetErr(pipew)

			logScanner := testutil.NewStrictLogScanner(ctx, piper, true, "The following Integration Kits will be deleted:", "foobar  in namespace: "+ns)

			go func() {
				err := kamelBuild.Execute()
				assert.NoError(t, err)
				logScanner.Done()
				cancel()
			}()

			Eventually(logScanner.ExactMatch(), TestTimeoutShort).Should(BeTrue())
			Eventually(Kits(ns)).Should(BeEmpty())
		})

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
