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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/scylladb/go-set/strset"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/tar"

	yaml2 "gopkg.in/yaml.v2"

	"github.com/pkg/errors"

	"github.com/apache/camel-k/pkg/util/maven"
)

// GenerateProject --
func GenerateProject(ctx *Context) error {
	p, err := NewProject(ctx)
	if err != nil {
		return err
	}

	ctx.Project = p

	//
	// set-up dependencies
	//

	ctx.Project.AddDependencyGAV("org.apache.camel.k", "camel-k-runtime-jvm", ctx.Request.RuntimeVersion)

	for _, d := range ctx.Request.Dependencies {
		switch {
		case strings.HasPrefix(d, "camel:"):
			artifactID := strings.TrimPrefix(d, "camel:")

			if !strings.HasPrefix(artifactID, "camel-") {
				artifactID = "camel-" + artifactID
			}

			ctx.Project.AddDependencyGAV("org.apache.camel", artifactID, "")
		case strings.HasPrefix(d, "camel-k:"):
			artifactID := strings.TrimPrefix(d, "camel-k:")

			if !strings.HasPrefix(artifactID, "camel-") {
				artifactID = "camel-" + artifactID
			}

			ctx.Project.AddDependencyGAV("org.apache.camel.k", artifactID, ctx.Request.RuntimeVersion)
		case strings.HasPrefix(d, "mvn:"):
			mid := strings.TrimPrefix(d, "mvn:")
			gav := strings.Replace(mid, "/", ":", -1)

			ctx.Project.AddEncodedDependencyGAV(gav)
		case strings.HasPrefix(d, "runtime:"):
			artifactID := strings.Replace(d, "runtime:", "camel-k-runtime-", 1)

			ctx.Project.AddDependencyGAV("org.apache.camel.k", artifactID, ctx.Request.RuntimeVersion)
		case strings.HasPrefix(d, "bom:"):
			// no-op
		default:
			return fmt.Errorf("unknown dependency type: %s", d)
		}
	}

	// Add Log4j 2 SLF4J binding as default logging impl
	ctx.Project.AddDependency(maven.Dependency{
		GroupID:    "org.apache.logging.log4j",
		ArtifactID: "log4j-slf4j-impl",
		Version:    "2.11.2",
		Scope:      "runtime",
	})

	return nil
}

// InjectDependencies --
func InjectDependencies(ctx *Context) error {
	//
	// Add dependencies from catalog
	//

	deps := make([]maven.Dependency, len(ctx.Project.Dependencies))
	copy(deps, ctx.Project.Dependencies)

	for _, d := range deps {
		if a, ok := ctx.Request.Catalog.Artifacts[d.ArtifactID]; ok {
			for _, dep := range a.Dependencies {
				md := maven.Dependency{
					GroupID:    dep.GroupID,
					ArtifactID: dep.ArtifactID,
				}

				ctx.Project.AddDependency(md)

				for _, e := range dep.Exclusions {
					me := maven.Exclusion{
						GroupID:    e.GroupID,
						ArtifactID: e.ArtifactID,
					}

					ctx.Project.AddDependencyExclusion(md, me)
				}
			}
		}
	}

	//
	// post process dependencies
	//

	deps = make([]maven.Dependency, len(ctx.Project.Dependencies))
	copy(deps, ctx.Project.Dependencies)

	for _, d := range deps {
		if a, ok := ctx.Request.Catalog.Artifacts[d.ArtifactID]; ok {
			md := maven.Dependency{
				GroupID:    a.GroupID,
				ArtifactID: a.ArtifactID,
			}

			for _, e := range a.Exclusions {
				me := maven.Exclusion{
					GroupID:    e.GroupID,
					ArtifactID: e.ArtifactID,
				}

				ctx.Project.AddDependencyExclusion(md, me)
			}
		}
	}

	return nil
}

// SanitizeDependencies --
func SanitizeDependencies(ctx *Context) error {
	for i := 0; i < len(ctx.Project.Dependencies); i++ {
		dep := ctx.Project.Dependencies[i]

		switch dep.GroupID {
		case "org.apache.camel":
			//
			// Remove the version so we force using the one configured by the bom
			//
			ctx.Project.Dependencies[i].Version = ""
		case "org.apache.camel.k":
			//
			// Force every runtime dependency to have the required version discardin
			// any version eventually set on the catalog
			//
			ctx.Project.Dependencies[i].Version = ctx.Request.RuntimeVersion
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

	opts := make([]string, 0, 2)
	opts = append(opts, maven.ExtraOptions(ctx.Request.Platform.Build.LocalRepository)...)
	opts = append(opts, fmt.Sprintf("org.apache.camel.k:camel-k-maven-plugin:%s:generate-dependency-list", ctx.Request.RuntimeVersion))

	err = maven.Run(p, opts...)
	if err != nil {
		return errors.Wrap(err, "failure while determining classpath")
	}

	dependencies := path.Join(p, "target", "dependencies.yaml")
	content, err := ioutil.ReadFile(dependencies)
	if err != nil {
		return err
	}

	cp := make(map[string][]v1alpha1.Artifact)
	err = yaml2.Unmarshal(content, &cp)
	if err != nil {
		return err
	}

	for _, e := range cp["dependencies"] {
		_, fileName := path.Split(e.Location)

		gav, err := maven.ParseGAV(e.ID)
		if err != nil {
			return nil
		}

		ctx.Artifacts = append(ctx.Artifacts, v1alpha1.Artifact{
			ID:       e.ID,
			Location: e.Location,
			Target:   path.Join("dependencies", gav.GroupID+"."+fileName),
		})
	}

	return nil
}

// ArtifactsSelector --
type ArtifactsSelector func(ctx *Context) error

// StandardPackager --
func StandardPackager(ctx *Context) error {
	return packager(ctx, func(ctx *Context) error {
		ctx.SelectedArtifacts = ctx.Artifacts

		return nil
	})
}

// IncrementalPackager --
func IncrementalPackager(ctx *Context) error {
	if ctx.HasRequiredImage() {
		//
		// If the build requires a specific image, don't try to determine the
		// base image using artifact so just use the standard packages
		//
		return StandardPackager(ctx)
	}

	images, err := ListPublishedImages(ctx)
	if err != nil {
		return err
	}

	return packager(ctx, func(ctx *Context) error {
		ctx.SelectedArtifacts = ctx.Artifacts

		bestImage, commonLibs := FindBestImage(images, ctx.Request.Dependencies, ctx.Artifacts)
		if bestImage.Image != "" {
			selectedArtifacts := make([]v1alpha1.Artifact, 0)
			for _, entry := range ctx.Artifacts {
				if _, isCommon := commonLibs[entry.ID]; !isCommon {
					selectedArtifacts = append(selectedArtifacts, entry)
				}
			}

			ctx.BaseImage = bestImage.Image
			ctx.Image = bestImage.Image
			ctx.SelectedArtifacts = selectedArtifacts
		}

		return nil
	})
}

// ClassPathPackager --
func packager(ctx *Context, selector ArtifactsSelector) error {
	err := selector(ctx)
	if err != nil {
		return err
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

	for _, entry := range ctx.SelectedArtifacts {
		_, tarFileName := path.Split(entry.Target)
		tarFilePath := path.Dir(entry.Target)

		_, err = tarAppender.AddFileWithName(tarFileName, entry.Location, tarFilePath)
		if err != nil {
			return err
		}
	}

	for _, entry := range ctx.Request.Resources {
		if err := tarAppender.AddData(entry.Content, entry.Target); err != nil {
			return err
		}
	}

	ctx.Archive = tarFileName

	return nil
}

// ListPublishedImages --
func ListPublishedImages(context *Context) ([]PublishedImage, error) {
	list := v1alpha1.NewIntegrationContextList()

	err := context.Client.List(context.Request.C, &k8sclient.ListOptions{Namespace: context.Namespace}, &list)
	if err != nil {
		return nil, err
	}

	images := make([]PublishedImage, 0)
	for _, item := range list.Items {
		ctx := item

		if ctx.Status.CamelVersion != context.Catalog.Version {
			continue
		}
		if ctx.Status.RuntimeVersion != context.Request.RuntimeVersion {
			continue
		}
		if ctx.Status.Phase != v1alpha1.IntegrationContextPhaseReady || ctx.Labels == nil {
			continue
		}
		if context.ContextFilter != nil && !context.ContextFilter(&ctx) {
			continue
		}
		if ctxType, present := ctx.Labels["camel.apache.org/context.type"]; !present || ctxType != v1alpha1.IntegrationContextTypePlatform {
			continue
		}

		images = append(images, PublishedImage{
			Image:        ctx.Status.Image,
			Artifacts:    ctx.Status.Artifacts,
			Dependencies: ctx.Spec.Dependencies,
		})
	}
	return images, nil
}

// FindBestImage --
func FindBestImage(images []PublishedImage, dependencies []string, artifacts []v1alpha1.Artifact) (PublishedImage, map[string]bool) {
	var bestImage PublishedImage

	if len(images) == 0 {
		return bestImage, nil
	}

	requiredLibs := make(map[string]bool, len(artifacts))
	for _, entry := range artifacts {
		requiredLibs[entry.ID] = true
	}

	requiredRuntimes := strset.New()
	for _, entry := range dependencies {
		if strings.HasPrefix(entry, "runtime:") {
			requiredRuntimes.Add(entry)
		}
	}

	bestImageCommonLibs := make(map[string]bool)
	bestImageSurplusLibs := 0

	for _, image := range images {
		runtimes := strset.New()
		for _, entry := range image.Dependencies {
			if strings.HasPrefix(entry, "runtime:") {
				runtimes.Add(entry)
			}
		}

		//
		// check if the image has the same runtime requirements to avoid the heuristic
		// selector to include unwanted runtime bits such as spring-boot (which may have
		// an additional artifact only thus it may match)
		//
		if !requiredRuntimes.IsSubset(runtimes) {
			continue
		}

		common := make(map[string]bool)
		for _, artifact := range image.Artifacts {
			if _, ok := requiredLibs[artifact.ID]; ok {
				common[artifact.ID] = true
			}
		}

		numCommonLibs := len(common)
		surplus := len(image.Artifacts) - numCommonLibs

		if numCommonLibs != len(image.Artifacts) && surplus >= numCommonLibs/3 {
			// Heuristic approach: if there are too many unrelated libraries, just use
			// the base image
			continue
		}

		if numCommonLibs > len(bestImageCommonLibs) || (numCommonLibs == len(bestImageCommonLibs) && surplus < bestImageSurplusLibs) {
			bestImage = image
			bestImageCommonLibs = common
			bestImageSurplusLibs = surplus
		}
	}

	return bestImage, bestImageCommonLibs
}
