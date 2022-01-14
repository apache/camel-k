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
	"bufio"
	"context"
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

var Log = log.WithName("maven")

type Command struct {
	context Context
	project Project
}

func (c *Command) Do(ctx context.Context) error {
	if err := generateProjectStructure(c.context, c.project); err != nil {
		return err
	}

	mvnCmd := "mvn"
	if c, ok := os.LookupEnv("MAVEN_CMD"); ok {
		mvnCmd = c
	}

	args := make([]string, 0)
	args = append(args, c.context.AdditionalArguments...)

	if c.context.LocalRepository != "" {
		if _, err := os.Stat(c.context.LocalRepository); err == nil {
			args = append(args, "-Dmaven.repo.local="+c.context.LocalRepository)
		}
	}

	settingsPath := path.Join(c.context.Path, "settings.xml")
	if settingsExists, err := util.FileExists(settingsPath); err != nil {
		return err
	} else if settingsExists {
		args = append(args, "--global-settings", settingsPath)
	}

	settingsPath = path.Join(c.context.Path, "user-settings.xml")
	if settingsExists, err := util.FileExists(settingsPath); err != nil {
		return err
	} else if settingsExists {
		args = append(args, "--settings", settingsPath)
	}

	cmd := exec.CommandContext(ctx, mvnCmd, args...)
	cmd.Dir = c.context.Path

	var mavenOptions string
	if len(c.context.ExtraMavenOpts) > 0 {
		// Inherit the parent process environment
		env := os.Environ()

		mavenOpts, ok := os.LookupEnv("MAVEN_OPTS")
		if !ok {
			mavenOptions = strings.Join(c.context.ExtraMavenOpts, " ")
			env = append(env, "MAVEN_OPTS="+mavenOptions)
		} else {
			var extraOptions []string
			options := strings.Fields(mavenOpts)
			for _, extraOption := range c.context.ExtraMavenOpts {
				// Basic duplicated key detection, that should be improved
				// to support a wider range of JVM options
				key := strings.SplitN(extraOption, "=", 2)[0]
				exists := false
				for _, opt := range options {
					if strings.HasPrefix(opt, key) {
						exists = true

						break
					}
				}
				if !exists {
					extraOptions = append(extraOptions, extraOption)
				}
			}

			options = append(options, extraOptions...)
			mavenOptions = strings.Join(options, " ")
			for i, e := range env {
				if strings.HasPrefix(e, "MAVEN_OPTS=") {
					env[i] = "MAVEN_OPTS=" + mavenOptions
					break
				}
			}
		}

		cmd.Env = env
	}

	Log.WithValues("MAVEN_OPTS", mavenOptions).Infof("executing: %s", strings.Join(cmd.Args, " "))

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()

	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdOut)

	Log.Debug("About to start parsing the Maven output")
	for scanner.Scan() {
		line := scanner.Text()
		mavenLog, parseError := parseLog(line)
		if parseError == nil {
			normalizeLog(mavenLog)
		} else {
			// Why we are ignoring the parsing errors here: there are a few scenarios where this would likely occur.
			// For example, if something outside of Maven outputs something (i.e.: the JDK, a misbehaved plugin,
			// etc). The build may still have succeeded, though.
			nonNormalizedLog(line)
		}
	}
	Log.Debug("Finished parsing Maven output")

	return cmd.Wait()
}

func NewContext(buildDir string) Context {
	return Context{
		Path:                buildDir,
		AdditionalArguments: make([]string, 0),
		AdditionalEntries:   make(map[string]interface{}),
	}
}

type Context struct {
	Path                string
	ExtraMavenOpts      []string
	GlobalSettings      []byte
	UserSettings        []byte
	AdditionalArguments []string
	AdditionalEntries   map[string]interface{}
	LocalRepository     string
}

func (c *Context) AddEntry(id string, entry interface{}) {
	if c.AdditionalEntries == nil {
		c.AdditionalEntries = make(map[string]interface{})
	}

	c.AdditionalEntries[id] = entry
}

func (c *Context) AddArgument(argument string) {
	c.AdditionalArguments = append(c.AdditionalArguments, argument)
}

func (c *Context) AddArgumentf(format string, args ...interface{}) {
	c.AdditionalArguments = append(c.AdditionalArguments, fmt.Sprintf(format, args...))
}

func (c *Context) AddArguments(arguments ...string) {
	c.AdditionalArguments = append(c.AdditionalArguments, arguments...)
}

func (c *Context) AddSystemProperty(name string, value string) {
	c.AddArgumentf("-D%s=%s", name, value)
}

func generateProjectStructure(context Context, project Project) error {
	if err := util.WriteFileWithBytesMarshallerContent(context.Path, "pom.xml", project); err != nil {
		return err
	}

	if context.GlobalSettings != nil {
		if err := util.WriteFileWithContent(path.Join(context.Path, "settings.xml"), context.GlobalSettings); err != nil {
			return err
		}
	}

	if context.UserSettings != nil {
		if err := util.WriteFileWithContent(path.Join(context.Path, "user-settings.xml"), context.UserSettings); err != nil {
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

			err = util.WriteFileWithContent(path.Join(context.Path, k), bytes)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ParseGAV decodes the provided Maven GAV into the corresponding Dependency.
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
