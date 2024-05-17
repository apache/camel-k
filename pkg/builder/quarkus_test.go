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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apache/camel-k/v2/pkg/util/boolean"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateQuarkusProjectCommon(t *testing.T) {
	p := generateQuarkusProjectCommon("1.2.3", "4.5.6")
	assert.Equal(t, "org.apache.camel.k.integration", p.GroupID)
	assert.Equal(t, "camel-k-integration", p.ArtifactID)
	assert.Equal(t, defaults.Version, p.Version)
	assert.Equal(t, "fast-jar", p.Properties["quarkus.package.type"])
	assert.Equal(t, "org.apache.camel.k", p.DependencyManagement.Dependencies[0].GroupID)
	assert.Equal(t, "camel-k-runtime-bom", p.DependencyManagement.Dependencies[0].ArtifactID)
	assert.Equal(t, "1.2.3", p.DependencyManagement.Dependencies[0].Version)
	assert.Equal(t, "pom", p.DependencyManagement.Dependencies[0].Type)
	assert.Equal(t, "import", p.DependencyManagement.Dependencies[0].Scope)
	assert.Equal(t, "io.quarkus", p.Build.Plugins[0].GroupID)
	assert.Equal(t, "quarkus-maven-plugin", p.Build.Plugins[0].ArtifactID)
	assert.Equal(t, "4.5.6", p.Build.Plugins[0].Version)
}

func TestLoadCamelQuarkusCatalogMissing(t *testing.T) {
	c, err := test.NewFakeClient()
	require.NoError(t, err)
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
	require.Error(t, err)
	assert.Equal(t, "unable to find catalog matching version requirement: runtime=1.2.3, provider=Quarkus", err.Error())
}

func TestLoadCamelQuarkusCatalogOk(t *testing.T) {
	runtimeCatalog := v1.RuntimeSpec{
		Version:      "1.2.3",
		Provider:     "Quarkus",
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
	require.NoError(t, err)
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
	require.NoError(t, err)
	assert.Equal(t, runtimeCatalog, builderContext.Catalog.Runtime)
}

func TestGenerateQuarkusProjectWithBuildTimeProperties(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-test-camel-k-quarkus-with-props")
	require.NoError(t, err)
	defaultCatalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	mavenProps := v1.Properties{}
	mavenProps.Add("quarkus.camel.hello", "world")
	mavenProps.Add("quarkus.camel.\"shouldnt\"", "fail")
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
	if strings.Contains(defaults.DefaultRuntimeVersion, "SNAPSHOT") {
		builderContext.Build.Maven.Repositories = []v1.Repository{
			{
				ID:   "APACHE-SNAPSHOT",
				Name: "Apache Snapshot",
				URL:  "https://repository.apache.org/content/repositories/snapshots-group",
				Snapshots: v1.RepositoryPolicy{
					Enabled:        true,
					UpdatePolicy:   "always",
					ChecksumPolicy: "ignore",
				},
				Releases: v1.RepositoryPolicy{
					Enabled: false,
				},
			},
		}
	}

	err = generateQuarkusProject(&builderContext)
	require.NoError(t, err)
	// use local Maven executable in tests
	t.Setenv("MAVEN_WRAPPER", boolean.FalseString)
	_, ok := os.LookupEnv("MAVEN_CMD")
	if !ok {
		t.Setenv("MAVEN_CMD", "mvn")
	}
	err = buildQuarkusRunner(&builderContext)
	require.NoError(t, err)
	appProps, err := os.ReadFile(filepath.Join(tmpDir, "maven", "src", "main", "resources", "application.properties"))
	require.NoError(t, err)
	assert.Contains(t, string(appProps), "quarkus.camel.hello=world\n")
	assert.Contains(t, string(appProps), "quarkus.camel.\"shouldnt\"=fail\n")
	assert.Contains(t, string(appProps), "my-build-time-var=my-build-time-val\n")
	assert.Contains(t, string(appProps), "my-build-time\var2=my-build-time-val2\n")
	// At this stage a maven project should have been executed. Verify the package was created.
	_, err = os.Stat(filepath.Join(tmpDir, "maven", "target", "camel-k-integration-"+defaults.Version+".jar"))
	require.NoError(t, err)
}

func TestGenerateQuarkusProjectWithNativeSources(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-test-camel-k-quarkus-native")
	require.NoError(t, err)
	defaultCatalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	builderContext := builderContext{
		C:         context.TODO(),
		Path:      tmpDir,
		Namespace: "test",
		Build: v1.BuilderTask{
			Runtime: defaultCatalog.Runtime,
			Maven: v1.MavenBuildSpec{
				MavenSpec: v1.MavenSpec{},
			},
			Sources: []v1.SourceSpec{v1.NewSourceSpec("Test.java", "bogus, irrelevant for test", v1.LanguageJavaSource)},
		},
	}
	if strings.Contains(defaults.DefaultRuntimeVersion, "SNAPSHOT") {
		builderContext.Build.Maven.Repositories = []v1.Repository{
			{
				ID:   "APACHE-SNAPSHOT",
				Name: "Apache Snapshot",
				URL:  "https://repository.apache.org/content/repositories/snapshots-group",
				Snapshots: v1.RepositoryPolicy{
					Enabled:        true,
					UpdatePolicy:   "always",
					ChecksumPolicy: "ignore",
				},
				Releases: v1.RepositoryPolicy{
					Enabled: false,
				},
			},
		}
	}

	err = prepareProjectWithSources(&builderContext)
	require.NoError(t, err)
	err = generateQuarkusProject(&builderContext)
	require.NoError(t, err)
	// use local Maven executable in tests
	t.Setenv("MAVEN_WRAPPER", boolean.FalseString)
	_, ok := os.LookupEnv("MAVEN_CMD")
	if !ok {
		t.Setenv("MAVEN_CMD", "mvn")
	}
	err = buildQuarkusRunner(&builderContext)
	require.NoError(t, err)
	appProps, err := os.ReadFile(filepath.Join(tmpDir, "maven", "src", "main", "resources", "application.properties"))
	require.NoError(t, err)
	assert.Contains(t, string(appProps), "quarkus.camel.routes-discovery.enabled=false\n")
	assert.Contains(t, string(appProps), "camel.main.routes-include-pattern = classpath:routes/Test.java\n")
	materializedRoute, err := os.ReadFile(filepath.Join(tmpDir, "maven", "src", "main", "resources", "routes", "Test.java"))
	require.NoError(t, err)
	assert.Contains(t, string(materializedRoute), "bogus, irrelevant for test")
	// At this stage a maven project should have been executed. Verify the package was created.
	_, err = os.Stat(filepath.Join(tmpDir, "maven", "target", "camel-k-integration-"+defaults.Version+".jar"))
	require.NoError(t, err)
}

func TestBuildQuarkusRunner(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-test-camel-k-quarkus")
	require.NoError(t, err)
	defaultCatalog, err := camel.DefaultCatalog()
	require.NoError(t, err)
	c, err := test.NewFakeClient(&v1.CamelCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "camel-catalog-" + defaults.DefaultRuntimeVersion,
		},
		Spec: v1.CamelCatalogSpec{
			Runtime: defaultCatalog.Runtime,
		},
	})
	require.NoError(t, err)
	mavenProps := v1.Properties{}
	mavenProps.Add("camel.hello", "world")
	builderContext := builderContext{
		Client:    c,
		Catalog:   defaultCatalog,
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
			Dependencies: []string{"mvn:org.apache.camel.k:camel-k-runtime"},
		},
	}
	if strings.Contains(defaults.DefaultRuntimeVersion, "SNAPSHOT") {
		builderContext.Build.Maven.Repositories = []v1.Repository{
			{
				ID:   "APACHE-SNAPSHOT",
				Name: "Apache Snapshot",
				URL:  "https://repository.apache.org/content/repositories/snapshots-group",
				Snapshots: v1.RepositoryPolicy{
					Enabled:        true,
					UpdatePolicy:   "always",
					ChecksumPolicy: "ignore",
				},
				Releases: v1.RepositoryPolicy{
					Enabled: false,
				},
			},
		}
	}
	err = generateQuarkusProject(&builderContext)
	require.NoError(t, err)
	err = injectDependencies(&builderContext)
	require.NoError(t, err)
	err = sanitizeDependencies(&builderContext)
	require.NoError(t, err)
	// use local Maven executable in tests
	t.Setenv("MAVEN_WRAPPER", boolean.FalseString)
	_, ok := os.LookupEnv("MAVEN_CMD")
	if !ok {
		t.Setenv("MAVEN_CMD", "mvn")
	}
	err = buildQuarkusRunner(&builderContext)
	require.NoError(t, err)
	// Verify default application properties
	appProps, err := os.ReadFile(filepath.Join(tmpDir, "maven", "src", "main", "resources", "application.properties"))
	require.NoError(t, err)
	assert.Contains(t, string(appProps), "camel.hello=world\n")
	assert.Contains(t, string(appProps), "quarkus.banner.enabled=false\n")
	assert.Contains(t, string(appProps), "quarkus.camel.service.discovery.include-patterns=META-INF/services/org/apache/camel/datatype/converter/*,META-INF/services/org/apache/camel/datatype/transformer/*,META-INF/services/org/apache/camel/transformer/*\n")
	assert.Contains(t, string(appProps), "quarkus.class-loading.parent-first-artifacts=org.graalvm.regex:regex\n")
	// At this stage a maven project should have been executed. Verify the package was created.
	_, err = os.Stat(filepath.Join(tmpDir, "maven", "target", "camel-k-integration-"+defaults.Version+".jar"))
	require.NoError(t, err)

	// We use this same unit test to verify dependencies generated
	// (and spare some build time to avoid running another maven process)
	err = computeQuarkusDependencies(&builderContext)
	require.NoError(t, err)
	assert.NotEmpty(t, builderContext.Artifacts)
	camelRuntimeDepFound := false
	expectedArtifact := fmt.Sprintf("org.apache.camel.k.camel-k-runtime-%s.jar", defaults.DefaultRuntimeVersion)
	for _, artifact := range builderContext.Artifacts {
		if artifact.ID == expectedArtifact {
			camelRuntimeDepFound = true
			break
		}
	}
	assert.True(t, camelRuntimeDepFound, "Did not find expected artifact: %s", expectedArtifact)
}
