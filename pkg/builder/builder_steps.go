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

package builder

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/rs/xid"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"

	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/tar"

	"gopkg.in/yaml.v2"

	"github.com/pkg/errors"

	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/apache/camel-k/version"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GenerateProject --
func GenerateProject(ctx *Context) error {
	ctx.Project = maven.Project{
		XMLName:           xml.Name{Local: "project"},
		XMLNs:             "http://maven.apache.org/POM/4.0.0",
		XMLNsXsi:          "http://www.w3.org/2001/XMLSchema-instance",
		XsiSchemaLocation: "http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd",
		ModelVersion:      "4.0.0",
		GroupID:           "org.apache.camel.k.integration",
		ArtifactID:        "camel-k-integration",
		Version:           version.Version,
		DependencyManagement: maven.DependencyManagement{
			Dependencies: maven.Dependencies{
				Dependencies: []maven.Dependency{
					{
						//TODO: camel version should be retrieved from an external request or provided as static version
						GroupID:    "org.apache.camel",
						ArtifactID: "camel-bom",
						Version:    camel.Version,
						Type:       "pom",
						Scope:      "import",
					},
				},
			},
		},
		Dependencies: maven.Dependencies{
			Dependencies: make([]maven.Dependency, 0),
		},
	}

	//
	// Repositories
	//

	ctx.Project.Repositories = maven.Repositories{
		Repositories: make([]maven.Repository, 0, len(ctx.Request.Repositories)),
	}

	for i, r := range ctx.Request.Repositories {
		repo := maven.NewRepository(r)
		if repo.ID == "" {
			repo.ID = fmt.Sprintf("repo-%03d", i)
		}

		ctx.Project.Repositories.Repositories = append(ctx.Project.Repositories.Repositories, repo)
	}

	//
	// set-up dependencies
	//

	deps := &ctx.Project.Dependencies
	deps.AddGAV("org.apache.camel.k", "camel-k-runtime-jvm", version.Version)

	for _, d := range ctx.Request.Dependencies {
		if strings.HasPrefix(d, "camel:") {
			artifactID := strings.TrimPrefix(d, "camel:")

			if !strings.HasPrefix(artifactID, "camel-") {
				artifactID = "camel-" + artifactID
			}

			deps.AddGAV("org.apache.camel", artifactID, "")
		} else if strings.HasPrefix(d, "mvn:") {
			mid := strings.TrimPrefix(d, "mvn:")
			gav := strings.Replace(mid, "/", ":", -1)

			deps.AddEncodedGAV(gav)
		} else if strings.HasPrefix(d, "runtime:") {
			artifactID := strings.Replace(d, "runtime:", "camel-k-runtime-", 1)

			deps.AddGAV("org.apache.camel.k", artifactID, version.Version)
		} else {
			return fmt.Errorf("unknown dependency type: %s", d)
		}
	}

	return nil
}

// ComputeDependencies --
func ComputeDependencies(ctx *Context) error {
	p := path.Join(ctx.Path, "maven")

	err := maven.CreateStructure(p, ctx.Project)
	if err != nil {
		return err
	}

	goal := fmt.Sprintf("org.apache.camel.k:camel-k-runtime-dependency-lister:%s:generate-dependency-list", version.Version)

	err = maven.Run(p, MavenExtraOptions(), goal)
	if err != nil {
		return errors.Wrap(err, "failure while determining classpath")
	}

	dependencies := path.Join(p, "target", "dependencies.yaml")
	content, err := ioutil.ReadFile(dependencies)
	if err != nil {
		return err
	}

	cp := make(map[string][]v1alpha1.Artifact)
	err = yaml.Unmarshal(content, &cp)
	if err != nil {
		return err
	}

	for _, e := range cp["dependencies"] {
		ctx.Artifacts = append(ctx.Artifacts, v1alpha1.Artifact{
			ID:       e.ID,
			Location: e.Location,
			Target:   "dependencies",
		})
	}

	return nil
}

// ArtifactsSelector --
type ArtifactsSelector func([]v1alpha1.Artifact) (string, []v1alpha1.Artifact, error)

// StandardPackager --
func StandardPackager(ctx *Context) error {
	return packager(ctx, func(libraries []v1alpha1.Artifact) (string, []v1alpha1.Artifact, error) {
		return ctx.Image, libraries, nil
	})
}

// IncrementalPackager --
func IncrementalPackager(ctx *Context) error {
	images, err := ListPublishedImages(ctx.Namespace)
	if err != nil {
		return err
	}

	return packager(ctx, func(libraries []v1alpha1.Artifact) (string, []v1alpha1.Artifact, error) {
		bestImage, commonLibs := FindBestImage(images, libraries)
		if bestImage != nil {
			selectedClasspath := make([]v1alpha1.Artifact, 0)
			for _, entry := range libraries {
				if _, isCommon := commonLibs[entry.ID]; !isCommon {
					selectedClasspath = append(selectedClasspath, entry)
				}
			}

			return bestImage.Image, selectedClasspath, nil
		}

		// return default selection
		return ctx.Image, libraries, nil
	})
}

// ClassPathPackager --
func packager(ctx *Context, selector ArtifactsSelector) error {
	imageName, selectedArtifacts, err := selector(ctx.Artifacts)
	if err != nil {
		return err
	}
	if imageName == "" {
		imageName = ctx.Image
	}

	tarFileName := path.Join(ctx.Path, "package", "occi.tar")
	tarFileDir := path.Dir(tarFileName)

	err = os.MkdirAll(tarFileDir, 0777)
	if err != nil {
		return err
	}

	tarAppender, err := tar.NewAppender(tarFileName)
	if err != nil {
		return err
	}
	defer tarAppender.Close()

	for _, entry := range selectedArtifacts {
		gav, err := maven.ParseGAV(entry.ID)
		if err != nil {
			return err
		}

		_, fileName := path.Split(entry.Location)

		_, err = tarAppender.AddFileWithName(gav.GroupID+"."+fileName, entry.Location, entry.Target)
		if err != nil {
			return err
		}
	}

	if ctx.ComputeClasspath {
		cp := ""
		for _, entry := range ctx.Artifacts {
			gav, err := maven.ParseGAV(entry.ID)
			if err != nil {
				return nil
			}
			_, fileName := path.Split(entry.Location)
			cp += path.Join(entry.Target, gav.GroupID+"."+fileName) + "\n"
		}

		err = tarAppender.AppendData([]byte(cp), "classpath")
		if err != nil {
			return err
		}
	}

	ctx.Image = imageName
	ctx.Archive = tarFileName

	return nil
}

// ListPublishedImages --
func ListPublishedImages(namespace string) ([]PublishedImage, error) {
	list := v1alpha1.NewIntegrationContextList()

	err := sdk.List(namespace, &list, sdk.WithListOptions(&metav1.ListOptions{}))
	if err != nil {
		return nil, err
	}
	images := make([]PublishedImage, 0)
	for _, ctx := range list.Items {
		if ctx.Status.Phase != v1alpha1.IntegrationContextPhaseReady || ctx.Labels == nil {
			continue
		}
		if ctxType, present := ctx.Labels["camel.apache.org/context.type"]; !present || ctxType != "platform" {
			continue
		}

		images = append(images, PublishedImage{
			Image:     ctx.Status.Image,
			Classpath: ctx.Status.Classpath,
		})
	}
	return images, nil
}

// FindBestImage --
func FindBestImage(images []PublishedImage, entries []v1alpha1.Artifact) (*PublishedImage, map[string]bool) {
	if len(images) == 0 {
		return nil, nil
	}
	requiredLibs := make(map[string]bool, len(entries))
	for _, entry := range entries {
		requiredLibs[entry.ID] = true
	}

	var bestImage PublishedImage
	bestImageCommonLibs := make(map[string]bool, 0)
	bestImageSurplusLibs := 0
	for _, image := range images {
		common := make(map[string]bool)
		for _, id := range image.Classpath {
			if _, ok := requiredLibs[id]; ok {
				common[id] = true
			}
		}
		numCommonLibs := len(common)
		surplus := len(image.Classpath) - numCommonLibs

		if numCommonLibs != len(image.Classpath) && surplus >= numCommonLibs/3 {
			// Heuristic approach: if there are too many unrelated libraries, just use the base image
			continue
		}

		if numCommonLibs > len(bestImageCommonLibs) || (numCommonLibs == len(bestImageCommonLibs) && surplus < bestImageSurplusLibs) {
			bestImage = image
			bestImageCommonLibs = common
			bestImageSurplusLibs = surplus
		}
	}

	return &bestImage, bestImageCommonLibs
}

// Notify --
func Notify(ctx *Context) error {
	c := v1alpha1.IntegrationContext{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.IntegrationContextKind,
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ctx.Namespace,
			Name:      ctx.Request.Meta.Name,
		},
	}

	if err := sdk.Get(&c); err != nil {
		return err
	}

	t := c.DeepCopy()
	if t.Annotations == nil {
		t.Annotations = make(map[string]string)
	}

	// Add a random ID to trigger update
	t.Annotations["camel.apache.org/build.id"] = xid.New().String()

	if err := sdk.Update(t); err != nil {
		return err
	}

	return nil
}
