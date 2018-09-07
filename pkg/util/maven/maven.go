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
	"io/ioutil"
	"os"
	"archive/tar"
	"github.com/sirupsen/logrus"
	"io"
	"github.com/pkg/errors"
	"path"
	"strings"
	"os/exec"
)


const (
	buildDirPrefix = "maven-"
	artifactDirPrefix = "maven-bin-"
)

// Takes a project description and returns a binary tar with the built artifacts
func Build(project Project) (string, error) {
	buildDir, err := ioutil.TempDir("", buildDirPrefix)
	if err != nil {
		return "", errors.Wrap(err, "could not create temporary dir for maven source files")
	}
	defer os.RemoveAll(buildDir)

	err = createMavenStructure(buildDir, project)
	if err != nil {
		return "", errors.Wrap(err, "could not write maven source files")
	}
	err = runMavenBuild(buildDir)
	if err != nil {
		return "", err
	}
	tarfile, err := createTar(buildDir, project)
	if err != nil {
		return "", err
	}
	return tarfile, nil
}

func runMavenBuild(buildDir string) error {
	mavenBuild := exec.Command("mvn", "clean", "install", "-DskipTests")
	mavenBuild.Dir = buildDir
	logrus.Info("Starting maven build: mvn clean install -DskipTests")
	err := mavenBuild.Run()
	if err != nil {
		return errors.Wrap(err, "failure while executing maven build")
	}

	mavenDep := exec.Command("mvn", "dependency:copy-dependencies")
	mavenDep.Dir = buildDir
	logrus.Info("Copying maven dependencies: mvn dependency:copy-dependencies")
	err = mavenDep.Run()
	if err != nil {
		return errors.Wrap(err, "failure while extracting maven dependencies")
	}
	logrus.Info("Maven build completed successfully")
	return nil
}

func createTar(buildDir string, project Project) (string, error) {
	artifactDir, err := ioutil.TempDir("", artifactDirPrefix)
	if err != nil {
		return "", errors.Wrap(err, "could not create temporary dir for maven artifacts")
	}

	tarFileName := path.Join(artifactDir, project.ArtifactId + ".tar")
	tarFile, err := os.Create(tarFileName)
	if err != nil {
		return "", errors.Wrap(err, "cannot create tar file " + tarFileName)
	}
	defer tarFile.Close()

	writer := tar.NewWriter(tarFile)
	err = appendToTar(path.Join(buildDir, "target", project.ArtifactId + "-" + project.Version + ".jar"), "", writer)
	if err != nil {
		return "", err
	}

	// Environment variables
	if project.Env != nil {
		err = writeFile(buildDir, "run-env.sh", envFileContent(project.Env))
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
		Name: fileName,
		Size: info.Size(),
		Mode: int64(info.Mode()),
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

func createMavenStructure(buildDir string, project Project) error {
	err := writeFile(buildDir, "pom.xml", pomFileContent(project))
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
		return errors.Wrap(err, "could not create dir for file " + relativePath)
	}
	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return errors.Wrap(err, "could not create file " + relativePath)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		errors.Wrap(err, "could not write to file " + relativePath)
	}
	return nil
}


func envFileContent(env map[string]string) string {
	if env == nil {
		return ""
	}
	content := ""
	for k,v := range env {
		content = content + "export " + k + "=" + v + "\n"
	}
	return content
}


func pomFileContent(project Project) string {
	basePom := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
  <modelVersion>4.0.0</modelVersion>

  <groupId>` + project.GroupId + `</groupId>
  <artifactId>` + project.ArtifactId + `</artifactId>
  <version>` + project.Version + `</version>

  <dependencies>
    #dependencies#
  </dependencies>

</project>
`
	depStr := ""
	for _, dep := range project.Dependencies {
		depStr += "\t\t<dependency>"
		depStr += "\t\t\t<groupId>" + dep.GroupId + "</groupId>"
		depStr += "\t\t\t<artifactId>" + dep.ArtifactId + "</artifactId>"
		if dep.Version != "" {
			depStr += "\t\t\t<version>" + dep.Version + "</version>"
		}
		depStr += "\t\t</dependency>"
		depStr += "\n"
	}

	pom := strings.Replace(basePom, "#dependencies#", depStr, 1)
	return pom
}