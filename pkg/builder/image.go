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
	"io/ioutil"
	"os"
	"path"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/defaults"
)

const (
	ContextDir      = "context"
	DeploymentDir   = "/deployments"
	DependenciesDir = "dependencies"
)

func init() {
	registerSteps(Image)
}

type imageSteps struct {
	IncrementalImageContext Step
	NativeImageContext      Step
	StandardImageContext    Step
	ExecutableDockerfile    Step
	JvmDockerfile           Step
}

var Image = imageSteps{
	IncrementalImageContext: NewStep(ApplicationPackagePhase, incrementalImageContext),
	NativeImageContext:      NewStep(ApplicationPackagePhase, nativeImageContext),
	StandardImageContext:    NewStep(ApplicationPackagePhase, standardImageContext),
	ExecutableDockerfile:    NewStep(ApplicationPackagePhase+1, executableDockerfile),
	JvmDockerfile:           NewStep(ApplicationPackagePhase+1, jvmDockerfile),
}

type artifactsSelector func(ctx *builderContext) error

func nativeImageContext(ctx *builderContext) error {
	return imageContext(ctx, func(ctx *builderContext) error {
		runner := "camel-k-integration-" + defaults.Version + "-runner"

		ctx.BaseImage = "quay.io/quarkus/quarkus-distroless-image:1.0"
		ctx.Artifacts = []v1.Artifact{
			{
				ID:       runner,
				Location: path.Join(ctx.Path, "maven", "target", runner),
				Target:   runner,
			},
		}
		ctx.SelectedArtifacts = ctx.Artifacts

		return nil
	})
}

func executableDockerfile(ctx *builderContext) error {
	// #nosec G202
	dockerfile := []byte(`
		FROM ` + ctx.BaseImage + `
		WORKDIR ` + DeploymentDir + `
		COPY --chown=nonroot:root . ` + DeploymentDir + `
		USER nonroot
	`)

	err := ioutil.WriteFile(path.Join(ctx.Path, ContextDir, "Dockerfile"), dockerfile, 0o400)
	if err != nil {
		return err
	}

	return nil
}

func standardImageContext(ctx *builderContext) error {
	return imageContext(ctx, func(ctx *builderContext) error {
		ctx.SelectedArtifacts = ctx.Artifacts

		return nil
	})
}

func jvmDockerfile(ctx *builderContext) error {
	// #nosec G202
	dockerfile := []byte(`
		FROM ` + ctx.BaseImage + `
		ADD . ` + DeploymentDir + `
		USER 1000
	`)

	err := ioutil.WriteFile(path.Join(ctx.Path, ContextDir, "Dockerfile"), dockerfile, 0o400)
	if err != nil {
		return err
	}

	return nil
}

func incrementalImageContext(ctx *builderContext) error {
	if ctx.Build.BaseImage != "" {
		// If the build requires a specific image, don't try to determine the
		// base image using artifact so just use the standard packages
		return standardImageContext(ctx)
	}

	images, err := listPublishedImages(ctx)
	if err != nil {
		return err
	}

	return imageContext(ctx, func(ctx *builderContext) error {
		ctx.SelectedArtifacts = ctx.Artifacts

		bestImage, commonLibs := findBestImage(images, ctx.Artifacts)
		if bestImage.Image != "" {
			ctx.BaseImage = bestImage.Image
			ctx.SelectedArtifacts = make([]v1.Artifact, 0)

			for _, entry := range ctx.Artifacts {
				if _, isCommon := commonLibs[entry.ID]; !isCommon {
					ctx.SelectedArtifacts = append(ctx.SelectedArtifacts, entry)
				}
			}
		} else if ctx.BaseImage == "" {
			// TODO: transient workaround to be removed in 1.8.x
			ctx.BaseImage = defaults.BaseImage()
		}

		return nil
	})
}

func imageContext(ctx *builderContext, selector artifactsSelector) error {
	err := selector(ctx)
	if err != nil {
		return err
	}

	contextDir := path.Join(ctx.Path, ContextDir)

	err = os.MkdirAll(contextDir, 0o777)
	if err != nil {
		return err
	}

	for _, entry := range ctx.SelectedArtifacts {
		_, err := util.CopyFile(entry.Location, path.Join(contextDir, entry.Target))
		if err != nil {
			return err
		}
	}

	for _, entry := range ctx.Resources {
		filePath, fileName := path.Split(entry.Target)
		if err := util.WriteFileWithContent(path.Join(contextDir, filePath), fileName, entry.Content); err != nil {
			return err
		}
	}

	return nil
}

func listPublishedImages(context *builderContext) ([]v1.IntegrationKitStatus, error) {
	excludeNativeImages, err := labels.NewRequirement(v1.IntegrationKitLayoutLabel, selection.NotEquals, []string{
		v1.IntegrationKitLayoutNative,
	})
	if err != nil {
		return nil, err
	}

	options := []ctrl.ListOption{
		ctrl.InNamespace(context.Namespace),
		ctrl.MatchingLabels{
			v1.IntegrationKitTypeLabel:          v1.IntegrationKitTypePlatform,
			"camel.apache.org/runtime.version":  context.Catalog.Runtime.Version,
			"camel.apache.org/runtime.provider": string(context.Catalog.Runtime.Provider),
		},
		ctrl.MatchingLabelsSelector{
			Selector: labels.NewSelector().Add(*excludeNativeImages),
		},
	}

	list := v1.NewIntegrationKitList()
	err = context.Client.List(context.C, &list, options...)
	if err != nil {
		return nil, err
	}

	images := make([]v1.IntegrationKitStatus, 0)
	for _, item := range list.Items {
		kit := item

		if kit.Status.Phase != v1.IntegrationKitPhaseReady {
			continue
		}

		images = append(images, kit.Status)
	}
	return images, nil
}

func findBestImage(images []v1.IntegrationKitStatus, artifacts []v1.Artifact) (v1.IntegrationKitStatus, map[string]bool) {
	var bestImage v1.IntegrationKitStatus

	if len(images) == 0 {
		return bestImage, nil
	}

	requiredLibs := make(map[string]string, len(artifacts))
	for _, entry := range artifacts {
		requiredLibs[entry.ID] = entry.Checksum
	}

	bestImageCommonLibs := make(map[string]bool)
	bestImageSurplusLibs := 0

	for _, image := range images {
		common := make(map[string]bool)
		for _, artifact := range image.Artifacts {
			// If the Artifact's checksum is not defined we can't reliably determine if for some
			// reason the artifact has been changed but not the ID (as example for snapshots or
			// other generated jar) thus we do not take this artifact into account.
			if artifact.Checksum == "" {
				continue
			}
			if requiredLibs[artifact.ID] == artifact.Checksum {
				common[artifact.ID] = true
			}
		}

		numCommonLibs := len(common)
		surplus := len(image.Artifacts) - numCommonLibs

		if numCommonLibs != len(image.Artifacts) && surplus >= numCommonLibs/3 {
			// Heuristic approach: if there are too many unrelated libraries then this image is
			// not suitable to be used as base image
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
