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
	"bytes"
	"encoding/xml"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util"

	"github.com/sirupsen/logrus"
)

// BuildResult --
type BuildResult struct {
	Classpath []v1alpha1.Artifact
}

// GeneratePomContent generate a pom.xml file from the given project definition
func GeneratePomContent(project Project) (string, error) {
	w := &bytes.Buffer{}
	w.WriteString(xml.Header)

	e := xml.NewEncoder(w)
	e.Indent("", "  ")

	err := e.Encode(project)
	if err != nil {
		return "", err
	}

	return w.String(), nil
}

// CreateStructure --
func CreateStructure(buildDir string, project Project) error {
	logrus.Infof("write project: %+v", project)

	pom, err := GeneratePomContent(project)
	if err != nil {
		return err
	}

	err = util.WriteFileWithContent(buildDir, "pom.xml", pom)
	if err != nil {
		return err
	}

	return nil
}

// Run --
func Run(buildDir string, args ...string) error {
	mvnCmd := "mvn"
	if c, ok := os.LookupEnv("MAVEN_CMD"); ok {
		mvnCmd = c
	}

	l := logrus.WithFields(logrus.Fields{
		"logger": "maven",
	})

	cmd := exec.Command(mvnCmd, args...)
	cmd.Dir = buildDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	l.Infof("execute: %s", strings.Join(cmd.Args, " "))

	return cmd.Run()
}

// ParseGAV decode a maven artifact id to a dependency definition.
//
// The artifact id is in the form of:
//
//     <groupId>:<artifactId>[:<packagingType>[:<classifier>]]:(<version>|'?')
//
func ParseGAV(gav string) (Dependency, error) {
	// <groupId>:<artifactId>[:<packagingType>[:<classifier>]]:(<version>|'?')
	dep := Dependency{}
	rex := regexp.MustCompile("([^: ]+):([^: ]+)(:([^: ]*)(:([^: ]+))?)?(:([^: ]+))?")
	res := rex.FindStringSubmatch(gav)

	dep.GroupID = res[1]
	dep.ArtifactID = res[2]
	dep.Type = "jar"

	cnt := strings.Count(gav, ":")
	switch cnt {
	case 2:
		dep.Version = res[4]
	case 3:
		dep.Type = res[4]
		dep.Version = res[6]
	default:
		dep.Type = res[4]
		dep.Classifier = res[6]
		dep.Version = res[8]
	}

	return dep, nil
}

// ExtraOptions --
func ExtraOptions() string {
	if _, err := os.Stat("/tmp/artifacts/m2"); err == nil {
		return "-Dmaven.repo.local=/tmp/artifacts/m2"
	}
	return "-Dcamel.noop=true"
}
