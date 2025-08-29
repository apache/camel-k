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
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMavenContext(t *testing.T) {
	mvnSimpleCompile := getMavenContext([]string{"compile", "-s", "my-settings.xml"}, nil)
	mvnOptionsCompile := getMavenContext([]string{"compile", "-s", "my-settings.xml"}, []string{"-DmyProperty=hello"})
	mvnSimplePackage := getMavenContext([]string{"package", "-s", "my-settings.xml"}, nil)
	mvnOptionsPackage := getMavenContext([]string{"package", "-s", "my-settings.xml"}, []string{"-DmyProperty=hello"})

	assert.Equal(t, "compile\n-s\nmy-settings.xml\n", mvnSimpleCompile)
	assert.Equal(t, "compile\n-s\nmy-settings.xml\n-DmyProperty=hello\n", mvnOptionsCompile)
	assert.Equal(t, "-s\nmy-settings.xml\n", mvnSimplePackage)
	assert.Equal(t, "-s\nmy-settings.xml\n-DmyProperty=hello\n", mvnOptionsPackage)
}

func TestGenerateMavenContext(t *testing.T) {
	dir := t.TempDir()
	err := generateMavenContext(dir, []string{"hello"}, nil)
	require.NoError(t, err)
	f, err := os.Stat(path.Join(dir, ".mvn", "maven.config"))
	require.NoError(t, err)
	assert.Equal(t, "maven.config", f.Name())
}

var expectPom = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
  <modelVersion>4.0.0</modelVersion>
  <groupId>gid</groupId>
  <artifactId>aid</artifactId>
  <version>1.2.3</version>
  <dependencyManagement>
    <dependencies>
      <dependency>
        <groupId>mgid</groupId>
        <artifactId>maid</artifactId>
        <version>1.0.0</version>
      </dependency>
    </dependencies>
  </dependencyManagement>
  <dependencies>
    <dependency>
      <groupId>gid</groupId>
      <artifactId>aid</artifactId>
      <version>0.0.1</version>
    </dependency>
  </dependencies>
  <repositories></repositories>
  <pluginRepositories></pluginRepositories>
  <profiles></profiles>
</project>`

func TestGenerateMavenPom(t *testing.T) {
	dir := t.TempDir()
	context := NewContext(dir)
	project := NewProject()
	project.GroupID = "gid"
	project.ArtifactID = "aid"
	project.Version = "1.2.3"
	project.AddDependency(NewDependency("gid", "aid", "0.0.1"))
	project.DependencyManagement = &DependencyManagement{}
	project.DependencyManagement.Dependencies = []Dependency{
		NewDependency("mgid", "maid", "1.0.0"),
	}

	err := generateProjectPom(context, project)
	require.NoError(t, err)

	p := path.Join(dir, "pom.xml")
	f, err := os.Stat(p)
	require.NoError(t, err)
	assert.Equal(t, "pom.xml", f.Name())
	content, err := util.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, expectPom, string(content))
}

func TestDoSettings(t *testing.T) {
	dir := t.TempDir()
	// Required to simulate an existing maven local repo existing directory
	localRepo, err := os.MkdirTemp(dir, "repo")
	require.NoError(t, err)
	command := Command{
		context: NewContext(dir),
		project: NewProject(),
	}
	command.context.GlobalSettings = []byte("<globalSettings>here</globalSettings>")
	command.context.UserSettings = []byte("<userSettings>here</userSettings>")
	command.context.SettingsSecurity = []byte("<settingsSecurity>here</settingsSecurity>")
	command.context.AdditionalEntries = map[string]interface{}{
		"some-binary": []byte("some content converted to binary"),
	}
	command.context.LocalRepository = localRepo
	command.context.AdditionalArguments = []string{"package"}
	command.context.ExtraMavenOpts = []string{"extra"}

	err = command.DoSettings(context.TODO())
	require.NoError(t, err)

	p := path.Join(dir, "settings.xml")
	f, err := os.Stat(p)
	require.NoError(t, err)
	assert.Equal(t, "settings.xml", f.Name())
	content, err := util.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, "<globalSettings>here</globalSettings>", string(content))

	p = path.Join(dir, "user-settings.xml")
	f, err = os.Stat(p)
	require.NoError(t, err)
	assert.Equal(t, "user-settings.xml", f.Name())
	content, err = util.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, "<userSettings>here</userSettings>", string(content))

	p = path.Join(dir, "settings-security.xml")
	f, err = os.Stat(p)
	require.NoError(t, err)
	assert.Equal(t, "settings-security.xml", f.Name())
	content, err = util.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, "<settingsSecurity>here</settingsSecurity>", string(content))

	p = path.Join(dir, "some-binary")
	f, err = os.Stat(p)
	require.NoError(t, err)
	assert.Equal(t, "some-binary", f.Name())
	content, err = util.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, "some content converted to binary", string(content))

	p = path.Join(dir, ".mvn", "maven.config")
	f, err = os.Stat(p)
	require.NoError(t, err)
	assert.Equal(t, "maven.config", f.Name())
	content, err = util.ReadFile(p)
	require.NoError(t, err)
	assert.Contains(t, string(content), fmt.Sprintf("--global-settings\n%s/settings.xml", dir))
	assert.Contains(t, string(content), fmt.Sprintf("--settings\n%s/user-settings.xml", dir))
	assert.Contains(t, string(content), fmt.Sprintf("-Dsettings.security=%s/settings-security.xml", dir))
	assert.Contains(t, string(content), fmt.Sprintf("-Dmaven.repo.local=%s", localRepo))
	assert.Contains(t, string(content), "extra")
}
