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
	"io"
	"io/ioutil"
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

// GenerateProjectStructure --
func GenerateProjectStructure(context Context) error {
	if err := util.WriteFileWithBytesMarshallerContent(context.Path, "pom.xml", context.Project); err != nil {
		return err
	}

	if context.SettingsData != nil {
		if err := util.WriteFileWithContent(context.Path, "settings.xml", context.SettingsData); err != nil {
			return err
		}
	} else if context.Settings != nil {
		if err := util.WriteFileWithBytesMarshallerContent(context.Path, "settings.xml", context.Settings); err != nil {
			return err
		}
	}

	for k, v := range context.AdditionalEntries {
		var bytes []byte
		var err error

		if dc, ok := v.([]byte); ok {
			bytes = dc
		} else if dc, ok := v.(io.Reader); ok {
			bytes, err = ioutil.ReadAll(dc)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unknown content type: name=%s, content=%+v", k, v)
		}

		if len(bytes) > 0 {
			Log.Infof("write entry: %s (%d bytes)", k, len(bytes))

			err = util.WriteFileWithContent(context.Path, k, bytes)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Run --
func Run(context Context) error {
	if err := GenerateProjectStructure(context); err != nil {
		return err
	}

	mvnCmd := "mvn"
	if c, ok := os.LookupEnv("MAVEN_CMD"); ok {
		mvnCmd = c
	}

	args := append(context.AdditionalArguments, "--batch-mode")

	settingsPath := path.Join(context.Path, "settings.xml")
	settingsExists, err := util.FileExists(settingsPath)
	if err != nil {
		return err
	}

	if settingsExists {
		args = append(args, "--settings", settingsPath)
	}

	cmd := exec.Command(mvnCmd, args...)
	cmd.Dir = context.Path
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
