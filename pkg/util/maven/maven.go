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
	"archive/tar"
	"bytes"
	"encoding/xml"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	buildDirPrefix    = "maven-"
	artifactDirPrefix = "maven-bin-"
)

// Build takes a project description and returns a binary tar with the built artifacts
func Build(integration Integration) (string, error) {
	buildDir, err := ioutil.TempDir("", buildDirPrefix)
	if err != nil {
		return "", errors.Wrap(err, "could not create temporary dir for maven source files")
	}
	defer os.RemoveAll(buildDir)

	err = createMavenStructure(buildDir, integration)
	if err != nil {
		return "", errors.Wrap(err, "could not write maven source files")
	}
	err = runMavenBuild(buildDir)
	if err != nil {
		return "", err
	}
	tarfile, err := createTar(buildDir, integration)
	if err != nil {
		return "", err
	}
	return tarfile, nil
}

func runMavenBuild(buildDir string) error {
	mavenBuild := exec.Command("mvn", mavenExtraOptions(), "clean", "install", "-DskipTests")
	mavenBuild.Dir = buildDir
	mavenBuild.Stdout = os.Stdout
	mavenBuild.Stderr = os.Stderr
	logrus.Info("Starting maven build: mvn " + mavenExtraOptions() + " clean install -DskipTests")
	err := mavenBuild.Run()
	if err != nil {
		return errors.Wrap(err, "failure while executing maven build")
	}

	mavenDep := exec.Command("mvn", mavenExtraOptions(), "dependency:copy-dependencies")
	mavenDep.Dir = buildDir
	logrus.Info("Copying maven dependencies: mvn " + mavenExtraOptions() + " dependency:copy-dependencies")
	err = mavenDep.Run()
	if err != nil {
		return errors.Wrap(err, "failure while extracting maven dependencies")
	}
	logrus.Info("Maven build completed successfully")
	return nil
}

func mavenExtraOptions() string {
	if _, err := os.Stat("/tmp/artifacts/m2"); err == nil {
		return "-Dmaven.repo.local=/tmp/artifacts/m2"
	}
	return "-Dcamel.noop=true"
}

func createTar(buildDir string, integration Integration) (string, error) {
	artifactDir, err := ioutil.TempDir("", artifactDirPrefix)
	if err != nil {
		return "", errors.Wrap(err, "could not create temporary dir for maven artifacts")
	}

	tarFileName := path.Join(artifactDir, integration.Project.ArtifactID+".tar")
	tarFile, err := os.Create(tarFileName)
	if err != nil {
		return "", errors.Wrap(err, "cannot create tar file "+tarFileName)
	}
	defer tarFile.Close()

	writer := tar.NewWriter(tarFile)
	err = appendToTar(path.Join(buildDir, "target", integration.Project.ArtifactID+"-"+integration.Project.Version+".jar"), "", writer)
	if err != nil {
		return "", err
	}

	// Environment variables
	if integration.Env != nil {
		err = writeFile(buildDir, "run-env.sh", envFileContent(integration.Env))
		if err != nil {
			return "", err
		}
		err = appendToTar(path.Join(buildDir, "run-env.sh"), "", writer)
		if err != nil {
			return "", err
		}
	}

	dependenciesDir := path.Join(buildDir, "target", "dependency")
	dependencies, err := ioutil.ReadDir(dependenciesDir)
	if err != nil {
		return "", err
	}

	for _, dep := range dependencies {
		err = appendToTar(path.Join(dependenciesDir, dep.Name()), "", writer)
		if err != nil {
			return "", err
		}
	}

	writer.Close()

	return tarFileName, nil
}

func appendToTar(filePath string, tarPath string, writer *tar.Writer) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	_, fileName := path.Split(filePath)
	if tarPath != "" {
		fileName = path.Join(tarPath, fileName)
	}

	writer.WriteHeader(&tar.Header{
		Name:    fileName,
		Size:    info.Size(),
		Mode:    int64(info.Mode()),
		ModTime: info.ModTime(),
	})

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(writer, file)
	if err != nil {
		return errors.Wrap(err, "cannot add file to the tar archive")
	}
	return nil
}

func createMavenStructure(buildDir string, project Integration) error {
	pom, err := GeneratePomFileContent(project.Project)
	if err != nil {
		return err
	}
	err = writeFile(buildDir, "pom.xml", pom)
	if err != nil {
		return err
	}
	err = writeFiles(path.Join(buildDir, "src", "main", "java"), project.JavaSources)
	if err != nil {
		return err
	}
	err = writeFiles(path.Join(buildDir, "src", "main", "resources"), project.Resources)
	if err != nil {
		return err
	}

	return nil
}

func writeFiles(buildDir string, files map[string]string) error {
	if files == nil {
		return nil
	}
	for fileName, fileContent := range files {
		err := writeFile(buildDir, fileName, fileContent)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeFile(buildDir string, relativePath string, content string) error {
	filePath := path.Join(buildDir, relativePath)
	fileDir := path.Dir(filePath)
	// Create dir if not present
	err := os.MkdirAll(fileDir, 0777)
	if err != nil {
		return errors.Wrap(err, "could not create dir for file "+relativePath)
	}
	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return errors.Wrap(err, "could not create file "+relativePath)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		errors.Wrap(err, "could not write to file "+relativePath)
	}
	return nil
}

func envFileContent(env map[string]string) string {
	if env == nil {
		return ""
	}
	content := ""
	for k, v := range env {
		content = content + "export " + k + "=" + v + "\n"
	}
	return content
}

// GeneratePomFileContent generate a pom.xml file from the given project definition
func GeneratePomFileContent(project Project) (string, error) {
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
	if cnt == 2 {
		dep.Version = res[4]
	} else if cnt == 3 {
		dep.Type = res[4]
		dep.Version = res[6]
	} else {
		dep.Type = res[4]
		dep.Classifier = res[6]
		dep.Version = res[8]
	}

	return dep, nil
}
