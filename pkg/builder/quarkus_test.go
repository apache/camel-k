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

package builder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/test"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateQuarkusProjectCommon(t *testing.T) {
	p, err := generateQuarkusProjectCommon("io.quarkus.platform", "4.5.6")
	assert.Nil(t, err)
	assert.Equal(t, "org.apache.camel.k.integration", p.GroupID)
	assert.Equal(t, "camel-k-integration", p.ArtifactID)
	assert.Equal(t, defaults.Version, p.Version)
	assert.Equal(t, "fast-jar", p.Properties["quarkus.package.type"])
	assert.Equal(t, "io.quarkus.platform", p.Properties["quarkus.platform.group-id"])
	assert.Equal(t, "4.5.6", p.Properties["quarkus.platform.version"])
}

func TestLoadCamelQuarkusCatalogMissing(t *testing.T) {
	c, err := test.NewFakeClient()
	assert.Nil(t, err)
	builderContext := builderContext{
		Client:    c,
		C:         context.TODO(),
		Namespace: "test",
		Build: v1.BuilderTask{
			Runtime: v1.RuntimeSpec{
				Version:  "1.2.3",
				Provider: "Quarkus",
			},
		},
	}
	err = loadCamelQuarkusCatalog(&builderContext)
	assert.NotNil(t, err)
	assert.Equal(t, "unable to find catalog matching version requirement: runtime=1.2.3, provider=Quarkus", err.Error())
}

func TestLoadCamelQuarkusCatalogOk(t *testing.T) {
	runtimeCatalog := v1.RuntimeSpec{
		Version:  "1.2.3",
		Provider: "Quarkus",
		Metadata: map[string]string{
			"quarkus.group.id": "io.quarkus.platform",
		},
		Dependencies: make([]v1.MavenArtifact, 0),
	}
	c, err := test.NewFakeClient(&v1.CamelCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "my-fake-catalog",
		},
		Spec: v1.CamelCatalogSpec{
			Runtime: runtimeCatalog,
		},
	})
	assert.Nil(t, err)
	builderContext := builderContext{
		Client:    c,
		C:         context.TODO(),
		Namespace: "default",
		Build: v1.BuilderTask{
			Runtime: v1.RuntimeSpec{
				Version:  "1.2.3",
				Provider: "Quarkus",
			},
		},
	}
	err = loadCamelQuarkusCatalog(&builderContext)
	assert.Nil(t, err)
	assert.Equal(t, runtimeCatalog, builderContext.Catalog.Runtime)
}

func TestGenerateQuarkusProject(t *testing.T) {
	mavenProps := v1.Properties{}
	mavenProps.Add("quarkus.camel.hello", "world")
	builderContext := builderContext{
		C:         context.TODO(),
		Namespace: "test",
		Build: v1.BuilderTask{
			Runtime: v1.RuntimeSpec{
				Version:  "1.2.3",
				Provider: "Quarkus",
				Metadata: map[string]string{
					"quarkus.version":  "3.2.3",
					"quarkus.group.id": "org.acme.quarkus",
				},
			},
			Maven: v1.MavenBuildSpec{
				MavenSpec: v1.MavenSpec{
					Properties: mavenProps,
				},
			},
		},
	}
	err := generateQuarkusProject(&builderContext)
	assert.Nil(t, err)
	assert.Equal(t, "org.acme.quarkus", builderContext.Maven.Project.Properties["quarkus.platform.group-id"])
	assert.Equal(t, "3.2.3", builderContext.Maven.Project.Properties["quarkus.platform.version"])
	assert.Len(t, builderContext.Maven.Project.DependencyManagement.Dependencies, 2)
	assert.Len(t, builderContext.Maven.Project.Dependencies, 1)
	assert.Equal(t, "camel-quarkus-core", builderContext.Maven.Project.Dependencies[0].ArtifactID)
}

func TestBuildQuarkusRunner(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-test-camel-k-quarkus")
	assert.Nil(t, err)
	defaultCatalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	mavenProps := v1.Properties{}
	mavenProps.Add("camel.hello", "world")
	builderContext := builderContext{
		C:         context.TODO(),
		Path:      tmpDir,
		Namespace: "test",
		Build: v1.BuilderTask{
			Runtime: defaultCatalog.Runtime,
			Maven: v1.MavenBuildSpec{
				MavenSpec: v1.MavenSpec{
					Properties: mavenProps,
				},
			},
		},
	}

	err = generateQuarkusProject(&builderContext)
	assert.Nil(t, err)
	err = buildQuarkusRunner(&builderContext)
	assert.Nil(t, err)
	// Verify default application properties
	appProps, err := os.ReadFile(filepath.Join(tmpDir, "maven", "src", "main", "resources", "application.properties"))
	assert.Nil(t, err)
	assert.Contains(t, string(appProps), "camel.hello=world\n")
	assert.Contains(t, string(appProps), "quarkus.banner.enabled=false\n")
	assert.Contains(t, string(appProps), "quarkus.camel.service.discovery.include-patterns=META-INF/services/org/apache/camel/datatype/converter/*,META-INF/services/org/apache/camel/datatype/transformer/*,META-INF/services/org/apache/camel/transformer/*\n")
	assert.Contains(t, string(appProps), "quarkus.class-loading.parent-first-artifacts=org.graalvm.regex:regex\n")
	// At this stage a maven project should have been executed. Verify the package was created.
	_, err = os.Stat(filepath.Join(tmpDir, "maven", "target", "camel-k-integration-"+defaults.Version+".jar"))
	assert.Nil(t, err)

	// We use this same unit test to verify dependencies generated
	// (and spare some build time to avoid running another maven process)
	err = computeQuarkusDependencies(&builderContext)
	assert.Nil(t, err)
	assert.NotEmpty(t, builderContext.Artifacts)
	camelQuarkusCoreFound := false
	// TODO catalog has a bug, must solve within next releases
	// expectedArtifact := fmt.Sprintf("org.apache.camel.quarkus.camel-quarkus-core-%s.jar", defaultCatalog.Runtime.Metadata["camel-quarkus.version"])
	expectedArtifact := "org.apache.camel.quarkus.camel-quarkus-core-3.2.2.jar"
	for _, artifact := range builderContext.Artifacts {
		if artifact.ID == expectedArtifact {
			camelQuarkusCoreFound = true
			break
		}
	}
	assert.True(t, camelQuarkusCoreFound, "Did not find expected artifact: %s", expectedArtifact)
}

func TestGenerateQuarkusProjectWithBuildTimeProperties(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-test-camel-k-quarkus-with-props")
	assert.Nil(t, err)
	defaultCatalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	mavenProps := v1.Properties{}
	mavenProps.Add("quarkus.camel.hello", "world")
	mavenProps.Add("my-build-time-var", "my-build-time-val")
	mavenProps.Add("my-build-time\var2", "my-build-time-val2")
	builderContext := builderContext{
		C:         context.TODO(),
		Path:      tmpDir,
		Namespace: "test",
		Build: v1.BuilderTask{
			Runtime: defaultCatalog.Runtime,
			Maven: v1.MavenBuildSpec{
				MavenSpec: v1.MavenSpec{
					Properties: mavenProps,
				},
			},
		},
	}

	err = generateQuarkusProject(&builderContext)
	assert.Nil(t, err)
	err = buildQuarkusRunner(&builderContext)
	assert.Nil(t, err)
	appProps, err := os.ReadFile(filepath.Join(tmpDir, "maven", "src", "main", "resources", "application.properties"))
	assert.Nil(t, err)
	assert.Contains(t, string(appProps), "camel.hello=world\n")
	assert.Contains(t, string(appProps), "my-build-time-var=my-build-time-val\n")
	assert.Contains(t, string(appProps), "my-build-time\var2=my-build-time-val2\n")
	// At this stage a maven project should have been executed. Verify the package was created.
	_, err = os.Stat(filepath.Join(tmpDir, "maven", "target", "camel-k-integration-"+defaults.Version+".jar"))
	assert.Nil(t, err)
}
