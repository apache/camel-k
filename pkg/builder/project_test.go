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
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/apache/camel-k/pkg/util/test"
)

const customSettings = `<?xml version="1.0" encoding="UTF-8"?>
<settings xmlns="http://maven.apache.org/SETTINGS/1.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" ` +
	`xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.0.0 https://maven.apache.org/xsd/settings-1.0.0.xsd">
	<localRepository/>
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
  <mirrors>
    <mirror>
      <id>foo</id>
      <url>https://foo.bar.org/repo</url>
      <mirrorOf>*</mirrorOf>
    </mirror>
  </mirrors>
</settings>`

const expectedCustomSettingsWithExtraServers = `<?xml version="1.0" encoding="UTF-8"?>
<settings xmlns="http://maven.apache.org/SETTINGS/1.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" ` +
	`xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.0.0 https://maven.apache.org/xsd/settings-1.0.0.xsd">
	<localRepository/>
  <servers>
    <server>
	  <id>image-repository</id>
	  <username>jpoth</username>
	  <password>changeit</password>
	  <configuration>
	    <allowInsecureRegistries>false</allowInsecureRegistries>
	  </configuration>
    </server>	 
  </servers>

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
  <mirrors>
    <mirror>
      <id>foo</id>
      <url>https://foo.bar.org/repo</url>
      <mirrorOf>*</mirrorOf>
    </mirror>
  </mirrors>
</settings>`

func TestMavenSettingsFromConfigMap(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	c, err := test.NewFakeClient(
		&corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "maven-settings",
			},
			Data: map[string]string{
				"settings.xml": "setting-data",
			},
		},
	)

	assert.Nil(t, err)

	ctx := builderContext{
		Catalog:   catalog,
		Client:    c,
		Namespace: "ns",
		Build: v1.BuilderTask{
			Runtime: catalog.Runtime,
			Maven: v1.MavenBuildSpec{
				MavenSpec: v1.MavenSpec{
					Settings: v1.ValueSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "maven-settings",
							},
							Key: "settings.xml",
						},
					},
				},
			},
		},
	}

	err = Project.GenerateProjectSettings.execute(&ctx)
	assert.Nil(t, err)

	assert.Equal(t, []byte("setting-data"), ctx.Maven.UserSettings)
}

func TestMavenSettingsWithSettingsSecurityFromConfigMap(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	c, err := test.NewFakeClient(
		&corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "maven-settings",
			},
			Data: map[string]string{
				"settings.xml": "setting-data",
			},
		},
		&corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "maven-settings-security",
			},
			Data: map[string]string{
				"settings-security.xml": "setting-security-data",
			},
		},
	)

	assert.Nil(t, err)

	ctx := builderContext{
		Catalog:   catalog,
		Client:    c,
		Namespace: "ns",
		Build: v1.BuilderTask{
			Runtime: catalog.Runtime,
			Maven: v1.MavenBuildSpec{
				MavenSpec: v1.MavenSpec{
					Settings: v1.ValueSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "maven-settings",
							},
							Key: "settings.xml",
						},
					},
					SettingsSecurity: v1.ValueSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "maven-settings-security",
							},
							Key: "settings-security.xml",
						},
					},
				},
			},
		},
	}

	err = Project.GenerateProjectSettings.execute(&ctx)
	assert.Nil(t, err)

	assert.Equal(t, []byte("setting-data"), ctx.Maven.UserSettings)
	assert.Equal(t, []byte("setting-security-data"), ctx.Maven.SettingsSecurity)
}

func TestMavenSettingsFromSecret(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	c, err := test.NewFakeClient(
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "maven-settings",
			},
			Data: map[string][]byte{
				"settings.xml": []byte("setting-data"),
			},
		},
	)

	assert.Nil(t, err)

	ctx := builderContext{
		Catalog:   catalog,
		Client:    c,
		Namespace: "ns",
		Build: v1.BuilderTask{
			Runtime: catalog.Runtime,
			Maven: v1.MavenBuildSpec{
				MavenSpec: v1.MavenSpec{
					Settings: v1.ValueSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "maven-settings",
							},
							Key: "settings.xml",
						},
					},
				},
			},
		},
	}

	err = Project.GenerateProjectSettings.execute(&ctx)
	assert.Nil(t, err)

	assert.Equal(t, []byte("setting-data"), ctx.Maven.UserSettings)
}

func TestMavenSettingsWithSettingsSecurityFromSecret(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	c, err := test.NewFakeClient(
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "maven-settings",
			},
			Data: map[string][]byte{
				"settings.xml": []byte("setting-data"),
			},
		},
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "maven-settings-security",
			},
			Data: map[string][]byte{
				"settings-security.xml": []byte("setting-security-data"),
			},
		},
	)

	assert.Nil(t, err)

	ctx := builderContext{
		Catalog:   catalog,
		Client:    c,
		Namespace: "ns",
		Build: v1.BuilderTask{
			Runtime: catalog.Runtime,
			Maven: v1.MavenBuildSpec{
				MavenSpec: v1.MavenSpec{
					Settings: v1.ValueSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "maven-settings",
							},
							Key: "settings.xml",
						},
					},
					SettingsSecurity: v1.ValueSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "maven-settings-security",
							},
							Key: "settings-security.xml",
						},
					},
				},
			},
		},
	}

	err = Project.GenerateProjectSettings.execute(&ctx)
	assert.Nil(t, err)

	assert.Equal(t, []byte("setting-data"), ctx.Maven.UserSettings)
	assert.Equal(t, []byte("setting-security-data"), ctx.Maven.SettingsSecurity)
}

func TestInjectEmptyServersIntoDefaultMavenSettings(t *testing.T) {
	settings, err := maven.NewSettings(maven.DefaultRepositories)
	assert.Nil(t, err)

	content, err := util.EncodeXML(settings)
	assert.Nil(t, err)

	contentStr := string(content)
	newSettings := injectServersIntoMavenSettings(contentStr, nil)

	assert.Equal(t, contentStr, newSettings)
}

func TestInjectServersIntoDefaultMavenSettings(t *testing.T) {
	settings, err := maven.NewSettings(maven.DefaultRepositories)
	assert.Nil(t, err)

	servers := []v1.Server{
		{
			ID:       "image-repository",
			Username: "jpoth",
			Password: "changeit",
			Configuration: map[string]string{
				"allowInsecureRegistries": "false",
			},
		},
	}

	content, err := util.EncodeXML(settings)
	assert.Nil(t, err)

	contentStr := string(content)
	newSettings := injectServersIntoMavenSettings(contentStr, servers)

	settings.Servers = servers
	expectedNewSettings, err := util.EncodeXML(settings)
	assert.Nil(t, err)

	expectedNewSettingsStr := string(expectedNewSettings)
	assert.Equal(t, expectedNewSettingsStr, newSettings)
}

func TestInjectServersIntoCustomMavenSettings(t *testing.T) {
	servers := []v1.Server{
		{
			ID:       "image-repository",
			Username: "jpoth",
			Password: "changeit",
			Configuration: map[string]string{
				"allowInsecureRegistries": "false",
			},
		},
	}

	newSettings := injectServersIntoMavenSettings(customSettings, servers)

	assert.Equal(t, removeWhitespaces(expectedCustomSettingsWithExtraServers), removeWhitespaces(newSettings))
}

func removeWhitespaces(s string) string {
	re := regexp.MustCompile(`\s`)
	return re.ReplaceAllString(s, "")
}
