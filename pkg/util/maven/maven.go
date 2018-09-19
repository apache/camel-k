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
	"bufio"
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

// BuildResult --
type BuildResult struct {
	TarFilePath string
	Classpath   []string
}

// Build takes a project description and returns a binary tar with the built artifacts
func Build(project Project) (BuildResult, error) {
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
	// file where the classpath is listed
	out := path.Join(buildDir, "integration.classpath")

	cmd := exec.Command("mvn", mavenExtraOptions(), "-Dmdep.outputFile="+out, "dependency:build-classpath")
	cmd.Dir = buildDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logrus.Infof("determine classpath: mvn: %v", cmd.Args)
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failure while determining classpath")
	}

	lines, err := readLines(out)
	if err != nil {
		return err
	}

	result.Classpath = make([]string, 0)
	for _, line := range lines {
		for _, item := range strings.Split(line, string(os.PathListSeparator)) {
			result.Classpath = append(result.Classpath, item)
		}
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

	for _, path := range result.Classpath {
		err = appendToTarWithPath(path, writer)
		if err != nil {
			return "", err
		}
	}

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

func appendToTarWithPath(path string, writer *tar.Writer) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	//TODO: we ned some constants
	relocatedPath := strings.TrimPrefix(path, "/tmp/artifacts")

	writer.WriteHeader(&tar.Header{
		Name:    relocatedPath,
		Size:    info.Size(),
		Mode:    int64(info.Mode()),
		ModTime: info.ModTime(),
	})

	file, err := os.Open(path)
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

//TODO: move to a file utility package
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
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
