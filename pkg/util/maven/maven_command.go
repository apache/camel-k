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
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/log"
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

	mvnCmd := ""
	if c, ok := os.LookupEnv("MAVEN_CMD"); ok {
		mvnCmd = c
	}

	if mvnCmd == "" {
		if e, ok := os.LookupEnv("MAVEN_WRAPPER"); (ok && e == "true") || !ok {
			// Prepare maven wrapper helps when running the builder as Pod as it makes
			// the builder container, Maven agnostic
			if err := c.prepareMavenWrapper(ctx); err != nil {
				return err
			}
		}

		mvnCmd = "./mvnw"
	}

	args := make([]string, 0)
	args = append(args, c.context.AdditionalArguments...)

	if c.context.LocalRepository != "" {
		if _, err := os.Stat(c.context.LocalRepository); err == nil {
			args = append(args, "-Dmaven.repo.local="+c.context.LocalRepository)
		}
	}

	settingsPath := filepath.Join(c.context.Path, "settings.xml")
	if settingsExists, err := util.FileExists(settingsPath); err != nil {
		return err
	} else if settingsExists {
		args = append(args, "--global-settings", settingsPath)
	}

	settingsPath = filepath.Join(c.context.Path, "user-settings.xml")
	if settingsExists, err := util.FileExists(settingsPath); err != nil {
		return err
	} else if settingsExists {
		args = append(args, "--settings", settingsPath)
	}

	settingsSecurityPath := filepath.Join(c.context.Path, "settings-security.xml")
	if settingsSecurityExists, err := util.FileExists(settingsSecurityPath); err != nil {
		return err
	} else if settingsSecurityExists {
		args = append(args, "-Dsettings.security="+settingsSecurityPath)
	}

	mavenOptions, env := c.optionsFromEnv()

	cmd := exec.CommandContext(ctx, mvnCmd, args...)
	cmd.Dir = c.context.Path
	cmd.Env = env

	Log.WithValues("MAVEN_OPTS", mavenOptions).Infof("executing: %s", strings.Join(cmd.Args, " "))

	// generate maven file
	if err := generateMavenContext(c.context.Path, args, mavenOptions); err != nil {
		return err
	}

	return util.RunAndLog(ctx, cmd, LogHandler, LogHandler)
}

func (c *Command) optionsFromEnv() (string, []string) {
	if len(c.context.ExtraMavenOpts) == 0 {
		return "", nil
	}

	var mavenOptions string

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

	return mavenOptions, env
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
	SettingsSecurity    []byte
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
	if err := util.WriteFileWithBytesMarshallerContent(context.Path, "pom.xml", &project); err != nil {
		return err
	}

	if context.GlobalSettings != nil {
		if err := util.WriteFileWithContent(filepath.Join(context.Path, "settings.xml"), context.GlobalSettings); err != nil {
			return err
		}
	}

	if context.UserSettings != nil {
		if err := util.WriteFileWithContent(filepath.Join(context.Path, "user-settings.xml"), context.UserSettings); err != nil {
			return err
		}
	}

	if context.SettingsSecurity != nil {
		if err := util.WriteFileWithContent(filepath.Join(context.Path, "settings-security.xml"), context.SettingsSecurity); err != nil {
			return err
		}
	}

	for k, v := range context.AdditionalEntries {
		var bytes []byte
		var err error

		if dc, ok := v.([]byte); ok {
			bytes = dc
		} else if dc, ok := v.(io.Reader); ok {
			bytes, err = io.ReadAll(dc)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unknown content type: name=%s, content=%+v", k, v)
		}

		if len(bytes) > 0 {
			Log.Infof("write entry: %s (%d bytes)", k, len(bytes))

			err = util.WriteFileWithContent(filepath.Join(context.Path, k), bytes)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// We expect a maven wrapper under /usr/share/maven/mvnw.
func (c *Command) prepareMavenWrapper(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "cp", "--recursive", "/usr/share/maven/mvnw/.", ".")
	cmd.Dir = c.context.Path
	return util.RunAndLog(ctx, cmd, LogHandler, LogHandler)
}

// ParseGAV decodes the provided Maven GAV into the corresponding Dependency.
//
// The artifact id is in the form of:
//
//	<groupId>:<artifactId>[:<packagingType>]:(<version>)[:<classifier>]
//
//nolint:mnd
func ParseGAV(gav string) (Dependency, error) {
	dep := Dependency{}
	res := strings.Split(gav, ":")
	count := len(res)
	if res == nil || count < 2 {
		return Dependency{}, errors.New("GAV must match <groupId>:<artifactId>[:<packagingType>]:(<version>)[:<classifier>]")
	}
	dep.GroupID = res[0]
	dep.ArtifactID = res[1]
	switch {
	case count == 3:
		// gav is: org:artifact:<type:version>
		numeric := regexp.MustCompile(`\d`)
		if numeric.MatchString(res[2]) {
			dep.Version = res[2]
		} else {
			dep.Type = res[2]
		}
	case count == 4:
		// gav is: org:artifact:type:version
		dep.Type = res[2]
		dep.Version = res[3]
	case count == 5:
		// gav is: org:artifact:<type>:<version>:classifier
		dep.Type = res[2]
		dep.Version = res[3]
		dep.Classifier = res[4]
	}
	return dep, nil
}

// Create a MAVEN_CONTEXT file containing all arguments for a maven command.
func generateMavenContext(path string, args []string, options string) error {
	// TODO refactor maven code to avoid creating a file to pass command args
	return util.WriteToFile(filepath.Join(path, "MAVEN_CONTEXT"), getMavenContext(args, options))
}

func getMavenContext(args []string, options string) string {
	commandArgs := make([]string, 0)
	for _, arg := range args {
		if arg != "package" && len(strings.TrimSpace(arg)) != 0 {
			commandArgs = append(commandArgs, strings.TrimSpace(arg))
		}
	}

	mavenContext := strings.Join(commandArgs, " ")
	if options != "" {
		mavenContext += " " + options
	}

	return mavenContext
}
