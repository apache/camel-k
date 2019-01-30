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

package maven

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
)

const expectedPom = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" ` +
	`xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
  <modelVersion>4.0.0</modelVersion>
  <groupId>org.apache.camel.k.integration</groupId>
  <artifactId>camel-k-integration</artifactId>
  <version>1.0.0</version>
  <dependencyManagement>
    <dependencies>
      <dependency>
        <groupId>org.apache.camel</groupId>
        <artifactId>camel-bom</artifactId>
        <version>2.22.1</version>
        <type>pom</type>
        <scope>import</scope>
      </dependency>
    </dependencies>
  </dependencyManagement>
  <dependencies>
    <dependency>
      <groupId>org.apache.camel.k</groupId>
      <artifactId>camel-k-runtime-jvm</artifactId>
      <version>1.0.0</version>
    </dependency>
  </dependencies>
  <repositories>
    <repository>
      <id>central</id>
      <url>https://repo.maven.apache.org/maven2</url>
      <snapshots>
        <enabled>false</enabled>
      </snapshots>
      <releases>
        <enabled>true</enabled>
        <updatePolicy>never</updatePolicy>
      </releases>
    </repository>
  </repositories>
  <pluginRepositories>
    <pluginRepository>
      <id>central</id>
      <url>https://repo.maven.apache.org/maven2</url>
      <snapshots>
        <enabled>false</enabled>
      </snapshots>
      <releases>
        <enabled>true</enabled>
        <updatePolicy>never</updatePolicy>
      </releases>
    </pluginRepository>
  </pluginRepositories>
</project>`

func TestPomGeneration(t *testing.T) {
	project := Project{
		XMLName:           xml.Name{Local: "project"},
		XMLNs:             "http://maven.apache.org/POM/4.0.0",
		XMLNsXsi:          "http://www.w3.org/2001/XMLSchema-instance",
		XsiSchemaLocation: "http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd",
		ModelVersion:      "4.0.0",
		GroupID:           "org.apache.camel.k.integration",
		ArtifactID:        "camel-k-integration",
		Version:           "1.0.0",
		DependencyManagement: DependencyManagement{
			Dependencies: Dependencies{
				Dependencies: []Dependency{
					{
						GroupID:    "org.apache.camel",
						ArtifactID: "camel-bom",
						Version:    "2.22.1",
						Type:       "pom",
						Scope:      "import",
					},
				},
			},
		},
		Dependencies: Dependencies{
			Dependencies: []Dependency{
				{
					GroupID:    "org.apache.camel.k",
					ArtifactID: "camel-k-runtime-jvm",
					Version:    "1.0.0",
				},
			},
		},
		Repositories: Repositories{
			Repositories: []Repository{
				{
					ID:  "central",
					URL: "https://repo.maven.apache.org/maven2",
					Snapshots: RepositoryPolicy{
						Enabled: false,
					},
					Releases: RepositoryPolicy{
						Enabled:      true,
						UpdatePolicy: "never",
					},
				},
			},
		},
		PluginRepositories: PluginRepositories{
			Repositories: []Repository{
				{
					ID:  "central",
					URL: "https://repo.maven.apache.org/maven2",
					Snapshots: RepositoryPolicy{
						Enabled: false,
					},
					Releases: RepositoryPolicy{
						Enabled:      true,
						UpdatePolicy: "never",
					},
				},
			},
		},
	}

	pom, err := GeneratePomContent(project)

	assert.Nil(t, err)
	assert.NotNil(t, pom)

	assert.Equal(t, expectedPom, pom)
}

func TestParseSimpleGAV(t *testing.T) {
	dep, err := ParseGAV("org.apache.camel:camel-core:2.21.1")

	assert.Nil(t, err)
	assert.Equal(t, dep.GroupID, "org.apache.camel")
	assert.Equal(t, dep.ArtifactID, "camel-core")
	assert.Equal(t, dep.Version, "2.21.1")
	assert.Equal(t, dep.Type, "jar")
	assert.Equal(t, dep.Classifier, "")
}

func TestParseGAVWithType(t *testing.T) {
	dep, err := ParseGAV("org.apache.camel:camel-core:war:2.21.1")

	assert.Nil(t, err)
	assert.Equal(t, dep.GroupID, "org.apache.camel")
	assert.Equal(t, dep.ArtifactID, "camel-core")
	assert.Equal(t, dep.Version, "2.21.1")
	assert.Equal(t, dep.Type, "war")
	assert.Equal(t, dep.Classifier, "")
}

func TestParseGAVWithClassifierAndType(t *testing.T) {
	dep, err := ParseGAV("org.apache.camel:camel-core:war:test:2.21.1")

	assert.Nil(t, err)
	assert.Equal(t, dep.GroupID, "org.apache.camel")
	assert.Equal(t, dep.ArtifactID, "camel-core")
	assert.Equal(t, dep.Version, "2.21.1")
	assert.Equal(t, dep.Type, "war")
	assert.Equal(t, dep.Classifier, "test")
}

func TestParseGAVMvnNoVersion(t *testing.T) {
	dep, err := ParseGAV("mvn:org.apache.camel/camel-core")

	assert.Nil(t, err)
	assert.Equal(t, dep.GroupID, "mvn")
	assert.Equal(t, dep.ArtifactID, "org.apache.camel/camel-core")
}

func TestParseGAVErrorNoColumn(t *testing.T) {
	dep, err := ParseGAV("org.apache.camel.k.camel-k-runtime-noop-0.2.1-SNAPSHOT.jar")

	assert.EqualError(t, err, "GAV must match <groupId>:<artifactId>[:<packagingType>[:<classifier>]]:(<version>|'?')")
	assert.Equal(t, Dependency{}, dep)
}

func TestNewRepository(t *testing.T) {
	r := NewRepository("http://nexus/public")
	assert.Equal(t, "", r.ID)
	assert.Equal(t, "http://nexus/public", r.URL)
	assert.True(t, r.Releases.Enabled)
	assert.False(t, r.Snapshots.Enabled)
}

func TestNewRepositoryWithSnapshots(t *testing.T) {
	r := NewRepository("http://nexus/public@snapshots")
	assert.Equal(t, "", r.ID)
	assert.Equal(t, "http://nexus/public", r.URL)
	assert.True(t, r.Releases.Enabled)
	assert.True(t, r.Snapshots.Enabled)
}

func TestNewRepositoryWithSnapshotsAndID(t *testing.T) {
	r := NewRepository("http://nexus/public@snapshots@id=test")
	assert.Equal(t, "test", r.ID)
	assert.Equal(t, "http://nexus/public", r.URL)
	assert.True(t, r.Releases.Enabled)
	assert.True(t, r.Snapshots.Enabled)
}

func TestNewRepositoryWithID(t *testing.T) {
	r := NewRepository("http://nexus/public@id=test")
	assert.Equal(t, "test", r.ID)
	assert.Equal(t, "http://nexus/public", r.URL)
	assert.True(t, r.Releases.Enabled)
	assert.False(t, r.Snapshots.Enabled)
}
