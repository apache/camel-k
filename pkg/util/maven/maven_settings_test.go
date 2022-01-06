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
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
)

const expectedSettings = `<?xml version="1.0" encoding="UTF-8"?>
<settings xmlns="http://maven.apache.org/SETTINGS/1.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" ` +
	`xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.0.0 https://maven.apache.org/xsd/settings-1.0.0.xsd">
  <localRepository>/tmp/artifacts/m2</localRepository>
  <profiles>
    <profile>
      <id>my-profile</id>
      <activation>
        <activeByDefault>true</activeByDefault>
      </activation>
      <repositories>
        <repository>
          <id>central</id>
          <url>https://repo.maven.apache.org/maven2</url>
          <snapshots>
            <enabled>false</enabled>
            <checksumPolicy>warn</checksumPolicy>
          </snapshots>
          <releases>
            <enabled>true</enabled>
            <updatePolicy>never</updatePolicy>
            <checksumPolicy>fail</checksumPolicy>
          </releases>
        </repository>
      </repositories>
      <pluginRepositories></pluginRepositories>
    </profile>
  </profiles>
  <proxies></proxies>
  <mirrors></mirrors>
</settings>`

const expectedDefaultSettings = `<?xml version="1.0" encoding="UTF-8"?>
<settings xmlns="http://maven.apache.org/SETTINGS/1.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" ` +
	`xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.0.0 https://maven.apache.org/xsd/settings-1.0.0.xsd">
  <localRepository></localRepository>
  <profiles>
    <profile>
      <id>maven-settings</id>
      <activation>
        <activeByDefault>true</activeByDefault>
      </activation>
      <repositories>
        <repository>
          <id>central</id>
          <url>https://repo.maven.apache.org/maven2</url>
          <snapshots>
            <enabled>false</enabled>
            <checksumPolicy>fail</checksumPolicy>
          </snapshots>
          <releases>
            <enabled>true</enabled>
            <checksumPolicy>fail</checksumPolicy>
          </releases>
        </repository>
      </repositories>
      <pluginRepositories>
        <pluginRepository>
          <id>central</id>
          <url>https://repo.maven.apache.org/maven2</url>
          <snapshots>
            <enabled>false</enabled>
            <checksumPolicy>fail</checksumPolicy>
          </snapshots>
          <releases>
            <enabled>true</enabled>
            <checksumPolicy>fail</checksumPolicy>
          </releases>
        </pluginRepository>
      </pluginRepositories>
    </profile>
  </profiles>
  <proxies></proxies>
  <mirrors></mirrors>
</settings>`

const expectedDefaultSettingsWithExtraRepo = `<?xml version="1.0" encoding="UTF-8"?>
<settings xmlns="http://maven.apache.org/SETTINGS/1.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" ` +
	`xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.0.0 https://maven.apache.org/xsd/settings-1.0.0.xsd">
  <localRepository></localRepository>
  <profiles>
    <profile>
      <id>maven-settings</id>
      <activation>
        <activeByDefault>true</activeByDefault>
      </activation>
      <repositories>
        <repository>
          <id>central</id>
          <url>https://repo1.maven.org/maven2</url>
          <snapshots>
            <enabled>false</enabled>
            <checksumPolicy>fail</checksumPolicy>
          </snapshots>
          <releases>
            <enabled>true</enabled>
            <checksumPolicy>fail</checksumPolicy>
          </releases>
        </repository>
        <repository>
          <id>foo</id>
          <url>https://foo.bar.org/repo</url>
          <snapshots>
            <enabled>false</enabled>
            <checksumPolicy>fail</checksumPolicy>
          </snapshots>
          <releases>
            <enabled>true</enabled>
            <checksumPolicy>fail</checksumPolicy>
          </releases>
        </repository>
      </repositories>
      <pluginRepositories>
        <pluginRepository>
          <id>central</id>
          <url>https://repo1.maven.org/maven2</url>
          <snapshots>
            <enabled>false</enabled>
            <checksumPolicy>fail</checksumPolicy>
          </snapshots>
          <releases>
            <enabled>true</enabled>
            <checksumPolicy>fail</checksumPolicy>
          </releases>
        </pluginRepository>
        <pluginRepository>
          <id>foo</id>
          <url>https://foo.bar.org/repo</url>
          <snapshots>
            <enabled>false</enabled>
            <checksumPolicy>fail</checksumPolicy>
          </snapshots>
          <releases>
            <enabled>true</enabled>
            <checksumPolicy>fail</checksumPolicy>
          </releases>
        </pluginRepository>
      </pluginRepositories>
    </profile>
  </profiles>
  <proxies></proxies>
  <mirrors>
    <mirror>
      <id>foo</id>
      <url>https://foo.bar.org/repo</url>
      <mirrorOf>*</mirrorOf>
    </mirror>
  </mirrors>
</settings>`

func TestSettingsGeneration(t *testing.T) {
	settings, err := NewSettings()
	assert.Nil(t, err)

	settings.LocalRepository = "/tmp/artifacts/m2"
	settings.Profiles = []Profile{
		{
			ID: "my-profile",
			Activation: Activation{
				ActiveByDefault: true,
			},
			Repositories: []v1.Repository{
				{
					ID:  "central",
					URL: "https://repo.maven.apache.org/maven2",
					Snapshots: v1.RepositoryPolicy{
						Enabled:        false,
						ChecksumPolicy: "warn",
					},
					Releases: v1.RepositoryPolicy{
						Enabled:        true,
						UpdatePolicy:   "never",
						ChecksumPolicy: "fail",
					},
				},
			},
		},
	}

	content, err := util.EncodeXML(settings)

	assert.Nil(t, err)
	assert.NotNil(t, settings)

	assert.Equal(t, expectedSettings, string(content))
}

func TestDefaultSettingsGeneration(t *testing.T) {
	settings := NewDefaultSettings([]v1.Repository{}, []Mirror{})

	content, err := util.EncodeXML(settings)

	assert.Nil(t, err)
	assert.NotNil(t, settings)

	assert.Equal(t, expectedDefaultSettings, string(content))
}

func TestDefaultSettingsGenerationWithAdditionalRepo(t *testing.T) {
	repositories := []v1.Repository{
		NewRepository("https://repo1.maven.org/maven2@id=central"),
		NewRepository("https://foo.bar.org/repo@id=foo"),
	}
	mirrors := []Mirror{
		NewMirror("https://foo.bar.org/repo@id=foo@mirrorOf=*"),
	}
	settings := NewDefaultSettings(repositories, mirrors)

	content, err := util.EncodeXML(settings)

	assert.Nil(t, err)
	assert.NotNil(t, settings)

	assert.Equal(t, expectedDefaultSettingsWithExtraRepo, string(content))
}

func TestCreateSettingsConfigMap(t *testing.T) {
	settings := NewDefaultSettings([]v1.Repository{}, []Mirror{})

	configMap, err := SettingsConfigMap("foo", "bar", settings)
	assert.Nil(t, err)
	assert.NotNil(t, configMap)

	content, err := util.EncodeXML(settings)

	assert.Nil(t, err)
	assert.NotNil(t, settings)

	assert.Equal(t, string(content), configMap.Data["settings.xml"])
}
