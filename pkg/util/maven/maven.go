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
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/log"
)

// Log --
var Log = log.WithName("maven")

// CreateStructure --
func CreateStructure(buildDir string, project Project, settings Settings) error {
	Log.Infof("write project: %+v", project)

	pomContent, err := util.EncodeXML(project)
	if err != nil {
		return err
	}

	err = util.WriteFileWithContent(buildDir, "pom.xml", []byte(pomContent))
	if err != nil {
		return err
	}

	if len(settings.Content) > 0 {
		err = util.WriteFileWithContent(buildDir, "settings.xml", settings.Content)
		if err != nil {
			return err
		}
	}

	return nil
}

// Run --
func Run(buildDir string, args ...string) error {
	mvnCmd := "mvn"
	if c, ok := os.LookupEnv("MAVEN_CMD"); ok {
		mvnCmd = c
	}

	args = append(args, "--batch-mode")

	settingsPath := path.Join(buildDir, "settings.xml")
	settingsExists, err := util.FileExists(settingsPath)
	if err != nil {
		return err
	}

	if settingsExists {
		args = append(args, "--settings", settingsPath)
	}

	cmd := exec.Command(mvnCmd, args...)
	cmd.Dir = buildDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	Log.Infof("execute: %s", strings.Join(cmd.Args, " "))

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

	fmt.Println(res, len(res))

	if res == nil || len(res) < 9 {
		return Dependency{}, errors.New("GAV must match <groupId>:<artifactId>[:<packagingType>[:<classifier>]]:(<version>|'?')")
	}

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
func ExtraOptions(localRepo string) []string {
	if _, err := os.Stat(localRepo); err == nil {
		return []string{"-Dmaven.repo.local=" + localRepo}
	}
	return []string{"-Dcamel.noop=true"}
}
