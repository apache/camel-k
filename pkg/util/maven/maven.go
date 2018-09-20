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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/apache/camel-k/version"

	"github.com/apache/camel-k/pkg/build"
	"gopkg.in/yaml.v1"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	buildDirPrefix    = "maven-"
	artifactDirPrefix = "maven-bin-"
)

// BuildResult --
type BuildResult struct {
	TarFilePath string
	Classpath   []build.ClasspathEntry
}

// Process takes a project description and returns a binary tar with the built artifacts
func Process(project Project) (BuildResult, error) {
	res := BuildResult{}
	buildDir, err := ioutil.TempDir("", buildDirPrefix)
	if err != nil {
		return res, errors.Wrap(err, "could not create temporary dir for maven source files")
	}

	defer os.RemoveAll(buildDir)

	err = createMavenStructure(buildDir, project)
	if err != nil {
		return res, errors.Wrap(err, "could not write maven source files")
	}
	err = runMavenBuild(buildDir, &res)
	if err != nil {
		return res, err
	}

	res.TarFilePath, err = createTar(project, &res)
	if err != nil {
		return res, err
	}

	return res, nil
}

func runMavenBuild(buildDir string, result *BuildResult) error {
	goal := fmt.Sprintf("org.apache.camel.k:camel-k-runtime-dependency-lister:%s:generate-dependency-list", version.Version)
	cmd := exec.Command("mvn", mavenExtraOptions(), goal)
	cmd.Dir = buildDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logrus.Infof("determine classpath: %v", cmd.Args)
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failure while determining classpath")
	}

	dependencies := path.Join(buildDir, "target", "dependencies.yaml")
	content, err := ioutil.ReadFile(dependencies)
	if err != nil {
		return err
	}

	cp := make(map[string][]build.ClasspathEntry)
	if err := yaml.Unmarshal(content, &cp); err != nil {
		return err
	}

	result.Classpath = cp["dependencies"]

	logrus.Info("Maven build completed successfully")
	return nil
}

func mavenExtraOptions() string {
	if _, err := os.Stat("/tmp/artifacts/m2"); err == nil {
		return "-Dmaven.repo.local=/tmp/artifacts/m2"
	}
	return "-Dcamel.noop=true"
}

func createTar(project Project, result *BuildResult) (string, error) {
	artifactDir, err := ioutil.TempDir("", artifactDirPrefix)
	if err != nil {
		return "", errors.Wrap(err, "could not create temporary dir for maven artifacts")
	}

	tarFileName := path.Join(artifactDir, project.ArtifactID+".tar")
	tarFile, err := os.Create(tarFileName)
	if err != nil {
		return "", errors.Wrap(err, "cannot create tar file "+tarFileName)
	}
	defer tarFile.Close()

	writer := tar.NewWriter(tarFile)
	defer writer.Close()

	cp := ""
	for _, entry := range result.Classpath {
		gav, err := ParseGAV(entry.ID)
		if err != nil {
			return "", nil
		}

		tarPath := path.Join("dependencies/", gav.GroupID)
		fileName, err := appendFileToTar(entry.Location, tarPath, writer)
		if err != nil {
			return "", err
		}

		cp += fileName + "\n"
	}

	err = appendDataToTar([]byte(cp), "classpath", writer)
	if err != nil {
		return "", err
	}

	return tarFileName, nil
}

func appendFileToTar(filePath string, tarPath string, writer *tar.Writer) (string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return "", err
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
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(writer, file)
	if err != nil {
		return "", errors.Wrap(err, "cannot add file to the tar archive")
	}

	return fileName, nil
}

func appendDataToTar(data []byte, tarPath string, writer *tar.Writer) error {
	writer.WriteHeader(&tar.Header{
		Name: tarPath,
		Size: int64(len(data)),
		Mode: 0644,
	})

	_, err := writer.Write(data)
	if err != nil {
		return errors.Wrap(err, "cannot add data to the tar archive")
	}
	return nil
}

func createMavenStructure(buildDir string, project Project) error {
	pom, err := GeneratePomFileContent(project)
	if err != nil {
		return err
	}

	err = writeFile(buildDir, "pom.xml", pom)
	if err != nil {
		return err
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
