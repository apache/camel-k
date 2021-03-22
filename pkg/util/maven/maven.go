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
	"io"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/log"
)

var Log = log.WithName("maven")

func GenerateProjectStructure(context Context) error {
	if err := util.WriteFileWithBytesMarshallerContent(context.Path, "pom.xml", context.Project); err != nil {
		return err
	}

	if context.SettingsContent != nil {
		if err := util.WriteFileWithContent(context.Path, "settings.xml", context.SettingsContent); err != nil {
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

func Run(ctx Context) error {
	if err := GenerateProjectStructure(ctx); err != nil {
		return err
	}

	mvnCmd := "mvn"
	if c, ok := os.LookupEnv("MAVEN_CMD"); ok {
		mvnCmd = c
	}

	args := make([]string, 0)
	args = append(args, "--batch-mode")

	if ctx.LocalRepository == "" {
		args = append(args, "-Dcamel.noop=true")
	} else if _, err := os.Stat(ctx.LocalRepository); err == nil {
		args = append(args, "-Dmaven.repo.local="+ctx.LocalRepository)
	}

	settingsPath := path.Join(ctx.Path, "settings.xml")
	settingsExists, err := util.FileExists(settingsPath)
	if err != nil {
		return err
	}

	if settingsExists {
		args = append(args, "--settings", settingsPath)
	}

	args = append(args, ctx.AdditionalArguments...)

	timeout := ctx.Timeout
	if timeout == 0 {
		timeout = math.MaxInt64
	}

	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(c, mvnCmd, args...)
	cmd.Dir = ctx.Path
	cmd.Stderr = os.Stderr
	if ctx.Stdout != nil {
		cmd.Stdout = ctx.Stdout
	} else {
		cmd.Stdout = os.Stdout
	}

	mavenOpts, ok := os.LookupEnv("MAVEN_OPTS")
	// FIXME: do not override duplicated Maven options
	mavenOpts = strings.Join(append(strings.Fields(mavenOpts), ctx.ExtraMavenOpts...), " ")

	// Inherit the parent process environment
	env := os.Environ()
	if !ok {
		env = append(env, mavenOpts)
	} else {
		for i, e := range env {
			if strings.HasPrefix(e, "MAVEN_OPTS=") {
				env[i] = mavenOpts
				break
			}
		}
	}

	cmd.Env = env

	Log.WithValues("timeout", timeout.String(), "env", env).
		Infof("executing: %s", strings.Join(cmd.Args, " "))

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

	if res == nil || len(res) < 9 {
		return Dependency{}, errors.New("GAV must match <groupId>:<artifactId>[:<packagingType>[:<classifier>]]:(<version>|'?')")
	}

	dep.GroupID = res[1]
	dep.ArtifactID = res[2]

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
