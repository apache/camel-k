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

	"github.com/scylladb/go-set/strset"

	"github.com/rs/xid"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"

	"github.com/apache/camel-k/pkg/util/tar"

	"gopkg.in/yaml.v2"

	"github.com/pkg/errors"

	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/apache/camel-k/version"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GenerateProject --
func GenerateProject(ctx *Context) error {
	ctx.Project = NewProject(ctx)

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
		switch {
		case strings.HasPrefix(d, "camel:"):
			artifactID := strings.TrimPrefix(d, "camel:")

			if !strings.HasPrefix(artifactID, "camel-") {
				artifactID = "camel-" + artifactID
			}

			deps.AddGAV("org.apache.camel", artifactID, "")
		case strings.HasPrefix(d, "mvn:"):
			mid := strings.TrimPrefix(d, "mvn:")
			gav := strings.Replace(mid, "/", ":", -1)

			deps.AddEncodedGAV(gav)
		case strings.HasPrefix(d, "runtime:"):
			artifactID := strings.Replace(d, "runtime:", "camel-k-runtime-", 1)

			deps.AddGAV("org.apache.camel.k", artifactID, version.Version)
		default:
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

	err := sdk.List(context.Namespace, &list, sdk.WithListOptions(&metav1.ListOptions{}))
	if err != nil {
		return nil, err
	}
	images := make([]PublishedImage, 0)
	for _, ctx := range list.Items {
		if ctx.Status.Phase != v1alpha1.IntegrationContextPhaseReady || ctx.Labels == nil {
			continue
		}
		if context.ContextFiler != nil && !context.ContextFiler(&ctx) {
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

// NotifyIntegrationContext --
func NotifyIntegrationContext(ctx *Context) error {
	target := v1alpha1.IntegrationContext{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.IntegrationContextKind,
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ctx.Namespace,
			Name:      ctx.Request.Meta.Name,
		},
	}

	if err := sdk.Get(&target); err != nil {
		return err
	}

	t := target.DeepCopy()
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

// NotifyIntegration --
func NotifyIntegration(ctx *Context) error {
	target := v1alpha1.Integration{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.IntegrationKind,
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ctx.Namespace,
			Name:      ctx.Request.Meta.Name,
		},
	}

	if err := sdk.Get(&target); err != nil {
		return err
	}

	t := target.DeepCopy()
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
