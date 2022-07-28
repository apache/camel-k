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

package cmd

import (
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/apache/camel-k/pkg/util"
)

// MavenWorkingDirectory is the directory used by Maven for an invocation of the kamel local command.
// By default, a temporary folder will be used.
var MavenWorkingDirectory = ""

// createMavenWorkingDirectory creates local Maven working directory.
func createMavenWorkingDirectory() error {
	temporaryDirectory, err := ioutil.TempDir(os.TempDir(), "maven-")
	if err != nil {
		return err
	}

	// Set the Maven directory to the default value
	MavenWorkingDirectory = temporaryDirectory

	return nil
}

// deleteMavenWorkingDirectory removes local Maven working directory.
func deleteMavenWorkingDirectory() error {
	return os.RemoveAll(MavenWorkingDirectory)
}

// getLocalDependenciesDir returns <mavenWorkingDirectory>/dependencies.
func getLocalDependenciesDir() string {
	return path.Join(MavenWorkingDirectory, util.DefaultDependenciesDirectoryName)
}

func createLocalDependenciesDirectory() error {
	// Do not create a directory unless the maven directory contains a valid value.
	if MavenWorkingDirectory == "" {
		return nil
	}

	directoryExists, err := util.DirectoryExists(getLocalDependenciesDir())
	if err != nil {
		return err
	}

	if !directoryExists {
		if err := os.MkdirAll(getLocalDependenciesDir(), 0o700); err != nil {
			return err
		}
	}

	return nil
}

// getLocalPropertiesDir returns <mavenWorkingDirectory>/properties.
func getLocalPropertiesDir() string {
	return path.Join(MavenWorkingDirectory, util.DefaultPropertiesDirectoryName)
}

func createLocalPropertiesDirectory() error {
	// Do not create a directory unless the maven directory contains a valid value.
	if MavenWorkingDirectory == "" {
		return nil
	}

	directoryExists, err := util.DirectoryExists(getLocalPropertiesDir())
	if err != nil {
		return err
	}

	if !directoryExists {
		err := os.MkdirAll(getLocalPropertiesDir(), 0o700)
		if err != nil {
			return err
		}
	}

	return nil
}

// getLocalRoutesDir returns <mavenWorkingDirectory>/routes.
func getLocalRoutesDir() string {
	return path.Join(MavenWorkingDirectory, util.DefaultRoutesDirectoryName)
}

func createLocalRoutesDirectory() error {
	// Do not create a directory unless the maven directory contains a valid value.
	if MavenWorkingDirectory == "" {
		return nil
	}

	directoryExists, err := util.DirectoryExists(getLocalRoutesDir())
	if err != nil {
		return err
	}

	if !directoryExists {
		if err := os.MkdirAll(getLocalRoutesDir(), 0o700); err != nil {
			return err
		}
	}

	return nil
}

// getLocalQuarkusDir returns <mavenWorkingDirectory>/quarkus.
func getLocalQuarkusDir() string {
	return path.Join(MavenWorkingDirectory, util.CustomQuarkusDirectoryName)
}

func createLocalQuarkusDirectory() error {
	// Do not create a directory unless the maven directory contains a valid value.
	if MavenWorkingDirectory == "" {
		return nil
	}

	directoryExists, err := util.DirectoryExists(getLocalQuarkusDir())
	if err != nil {
		return err
	}

	if !directoryExists {
		if err := os.MkdirAll(getLocalQuarkusDir(), 0o700); err != nil {
			return err
		}
	}

	return nil
}

// getLocalAppDir returns <mavenWorkingDirectory>/app.
func getLocalAppDir() string {
	return path.Join(MavenWorkingDirectory, util.CustomAppDirectoryName)
}

func createLocalAppDirectory() error {
	// Do not create a directory unless the maven directory contains a valid value.
	if MavenWorkingDirectory == "" {
		return nil
	}

	directoryExists, err := util.DirectoryExists(getLocalAppDir())
	if err != nil {
		return err
	}

	if !directoryExists {
		if err := os.MkdirAll(getLocalAppDir(), 0o700); err != nil {
			return err
		}
	}

	return nil
}

// getLocalLibDir returns <mavenWorkingDirectory>/lib/main.
func getLocalLibDir() string {
	return path.Join(MavenWorkingDirectory, util.CustomLibDirectoryName)
}

func createLocalLibDirectory() error {
	// Do not create a directory unless the maven directory contains a valid value.
	if MavenWorkingDirectory == "" {
		return nil
	}

	directoryExists, err := util.DirectoryExists(getLocalLibDir())
	if err != nil {
		return err
	}

	if !directoryExists {
		if err := os.MkdirAll(getLocalLibDir(), 0o700); err != nil {
			return err
		}
	}

	return nil
}

func getCustomDependenciesDir(dir string) string {
	return path.Join(dir, util.DefaultDependenciesDirectoryName)
}

func getCustomPropertiesDir(dir string) string {
	return path.Join(dir, util.DefaultPropertiesDirectoryName)
}

func getCustomRoutesDir(dir string) string {
	return path.Join(dir, util.DefaultRoutesDirectoryName)
}

func getCustomQuarkusDir(dir string) string {
	parentDir := path.Dir(strings.TrimSuffix(dir, "/"))
	return path.Join(parentDir, util.CustomQuarkusDirectoryName)
}

func getCustomLibDir(dir string) string {
	parentDir := path.Dir(strings.TrimSuffix(dir, "/"))
	return path.Join(parentDir, util.CustomLibDirectoryName)
}

func getCustomAppDir(dir string) string {
	parentDir := path.Dir(strings.TrimSuffix(dir, "/"))
	return path.Join(parentDir, "app")
}

func deleteLocalIntegrationDirs(dir string) error {
	dirs := []string{
		getCustomQuarkusDir(dir),
		getCustomLibDir(dir),
		getCustomAppDir(dir),
	}

	for _, dir := range dirs {
		err := os.RemoveAll(dir)
		if err != nil {
			return err
		}
	}

	return nil
}
