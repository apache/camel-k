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
	"os"
	"path"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/controller"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/apache/camel-k/pkg/util/tar"
)

var stepsByID = make(map[string]Step)

func init() {
	RegisterSteps(Steps)
}

type steps struct {
	CleanBuildDir           Step
	GenerateProjectSettings Step
	InjectDependencies      Step
	SanitizeDependencies    Step
	StandardPackager        Step
	IncrementalPackager     Step
}

// Steps --
var Steps = steps{
	CleanBuildDir: NewStep(
		ProjectGenerationPhase-1,
		cleanBuildDir,
	),
	GenerateProjectSettings: NewStep(
		ProjectGenerationPhase+1,
		generateProjectSettings,
	),
	InjectDependencies: NewStep(
		ProjectGenerationPhase+2,
		injectDependencies,
	),
	SanitizeDependencies: NewStep(
		ProjectGenerationPhase+3,
		sanitizeDependencies,
	),
	StandardPackager: NewStep(
		ApplicationPackagePhase,
		standardPackager,
	),
	IncrementalPackager: NewStep(
		ApplicationPackagePhase,
		incrementalPackager,
	),
}

// DefaultSteps --
var DefaultSteps = []Step{
	Steps.CleanBuildDir,
	Steps.GenerateProjectSettings,
	Steps.InjectDependencies,
	Steps.SanitizeDependencies,
	Steps.IncrementalPackager,
}

// RegisterSteps --
func RegisterSteps(steps interface{}) {
	v := reflect.ValueOf(steps)
	t := reflect.TypeOf(steps)

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if step, ok := v.Field(i).Interface().(Step); ok {
			id := t.PkgPath() + "/" + field.Name
			// Set the fully qualified step ID
			reflect.Indirect(v.Field(i).Elem()).FieldByName("StepID").SetString(id)

			registerStep(step)
		}
	}
}

func registerStep(steps ...Step) {
	for _, step := range steps {
		if _, exists := stepsByID[step.ID()]; exists {
			panic(fmt.Errorf("the build step is already registered: %s", step.ID()))
		}
		stepsByID[step.ID()] = step
	}
}

func cleanBuildDir(ctx *Context) error {
	if ctx.Build.BuildDir == "" {
		return nil
	}

	return os.RemoveAll(ctx.Build.BuildDir)
}

func generateProjectSettings(ctx *Context) error {
	val, err := kubernetes.ResolveValueSource(ctx.C, ctx.Client, ctx.Namespace, &ctx.Build.Maven.Settings)
	if err != nil {
		return err
	}
	if val != "" {
		ctx.Maven.SettingsData = []byte(val)
	}

	return nil
}

func injectDependencies(ctx *Context) error {
	// Add dependencies from build
	for _, d := range ctx.Build.Dependencies {
		switch {
		case strings.HasPrefix(d, "bom:"):
			mid := strings.TrimPrefix(d, "bom:")
			gav := strings.Replace(mid, "/", ":", -1)

			d, err := maven.ParseGAV(gav)
			if err != nil {
				return err
			}

			ctx.Maven.Project.DependencyManagement.Dependencies = append(ctx.Maven.Project.DependencyManagement.Dependencies, maven.Dependency{
				GroupID:    d.GroupID,
				ArtifactID: d.ArtifactID,
				Version:    d.Version,
				Type:       "pom",
				Scope:      "import",
			})
		case strings.HasPrefix(d, "camel:"):
			artifactID := strings.TrimPrefix(d, "camel:")

			if !strings.HasPrefix(artifactID, "camel-") {
				artifactID = "camel-" + artifactID
			}

			ctx.Maven.Project.AddDependencyGAV("org.apache.camel", artifactID, "")
		case strings.HasPrefix(d, "camel-k:"):
			artifactID := strings.TrimPrefix(d, "camel-k:")

			if !strings.HasPrefix(artifactID, "camel-k-") {
				artifactID = "camel-k-" + artifactID
			}

			ctx.Maven.Project.AddDependencyGAV("org.apache.camel.k", artifactID, "")
		case strings.HasPrefix(d, "camel-quarkus:"):
			artifactID := strings.TrimPrefix(d, "camel-quarkus:")

			if !strings.HasPrefix(artifactID, "camel-quarkus-") {
				artifactID = "camel-quarkus-" + artifactID
			}

			ctx.Maven.Project.AddDependencyGAV("org.apache.camel.quarkus", artifactID, "")
		case strings.HasPrefix(d, "mvn:"):
			mid := strings.TrimPrefix(d, "mvn:")
			gav := strings.Replace(mid, "/", ":", -1)

			ctx.Maven.Project.AddEncodedDependencyGAV(gav)
		default:
			return fmt.Errorf("unknown dependency type: %s", d)
		}
	}

	// Add dependencies from catalog
	deps := make([]maven.Dependency, len(ctx.Maven.Project.Dependencies))
	copy(deps, ctx.Maven.Project.Dependencies)

	for _, d := range deps {
		if a, ok := ctx.Catalog.Artifacts[d.ArtifactID]; ok {
			for _, dep := range a.Dependencies {
				md := maven.Dependency{
					GroupID:    dep.GroupID,
					ArtifactID: dep.ArtifactID,
				}

				ctx.Maven.Project.AddDependency(md)

				for _, e := range dep.Exclusions {
					me := maven.Exclusion{
						GroupID:    e.GroupID,
						ArtifactID: e.ArtifactID,
					}

					ctx.Maven.Project.AddDependencyExclusion(md, me)
				}
			}
		}
	}

	// Post process dependencies
	deps = make([]maven.Dependency, len(ctx.Maven.Project.Dependencies))
	copy(deps, ctx.Maven.Project.Dependencies)

	for _, d := range deps {
		if a, ok := ctx.Catalog.Artifacts[d.ArtifactID]; ok {
			md := maven.Dependency{
				GroupID:    a.GroupID,
				ArtifactID: a.ArtifactID,
			}

			for _, e := range a.Exclusions {
				me := maven.Exclusion{
					GroupID:    e.GroupID,
					ArtifactID: e.ArtifactID,
				}

				ctx.Maven.Project.AddDependencyExclusion(md, me)
			}
		}
	}

	return nil
}

func sanitizeDependencies(ctx *Context) error {
	for i := 0; i < len(ctx.Maven.Project.Dependencies); i++ {
		dep := ctx.Maven.Project.Dependencies[i]

		// It may be externalized into runtime provider specific steps
		switch dep.GroupID {
		case "org.apache.camel":
			fallthrough
		case "org.apache.camel.k":
			fallthrough
		case "org.apache.camel.quarkus":
			//
			// Remove the version so we force using the one configured by the bom
			//
			ctx.Maven.Project.Dependencies[i].Version = ""
		}
	}

	return nil
}

type artifactsSelector func(ctx *Context) error

func standardPackager(ctx *Context) error {
	return packager(ctx, func(ctx *Context) error {
		ctx.SelectedArtifacts = ctx.Artifacts

		return nil
	})
}

func incrementalPackager(ctx *Context) error {
	if ctx.HasRequiredImage() {
		//
		// If the build requires a specific image, don't try to determine the
		// base image using artifact so just use the standard packages
		//
		return standardPackager(ctx)
	}

	images, err := listPublishedImages(ctx)
	if err != nil {
		return err
	}

	return packager(ctx, func(ctx *Context) error {
		ctx.SelectedArtifacts = ctx.Artifacts

		bestImage, commonLibs := findBestImage(images, ctx.Artifacts)
		if bestImage.Image != "" {
			selectedArtifacts := make([]v1alpha1.Artifact, 0)
			for _, entry := range ctx.Artifacts {
				if _, isCommon := commonLibs[entry.ID]; !isCommon {
					selectedArtifacts = append(selectedArtifacts, entry)
				}
			}

			ctx.BaseImage = bestImage.Image
			ctx.SelectedArtifacts = selectedArtifacts
		}

		return nil
	})
}

func packager(ctx *Context, selector artifactsSelector) error {
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

	for _, entry := range ctx.Resources {
		if err := tarAppender.AddData(entry.Content, entry.Target); err != nil {
			return err
		}
	}

	ctx.Archive = tarFileName

	return nil
}

func listPublishedImages(context *Context) ([]publishedImage, error) {
	options := []k8sclient.ListOption{
		k8sclient.InNamespace(context.Namespace),
	}

	if context.Catalog.RuntimeProvider != nil && context.Catalog.RuntimeProvider.Quarkus != nil {
		options = append(options, k8sclient.MatchingLabels{
			"camel.apache.org/runtime.provider": "quarkus",
		})
	} else {
		provider, _ := labels.NewRequirement("camel.apache.org/runtime.provider", selection.DoesNotExist, []string{})
		selector := labels.NewSelector().Add(*provider)
		options = append(options, controller.MatchingSelector{Selector: selector})
	}

	list := v1alpha1.NewIntegrationKitList()
	err := context.Client.List(context.C, &list, options...)
	if err != nil {
		return nil, err
	}

	images := make([]publishedImage, 0)
	for _, item := range list.Items {
		kit := item

		if kit.Status.Phase != v1alpha1.IntegrationKitPhaseReady {
			continue
		}
		if kit.Status.CamelVersion != context.Catalog.Version {
			continue
		}
		if kit.Status.RuntimeVersion != context.Catalog.RuntimeVersion {
			continue
		}

		// TODO: should ideally be made generic from the runtime providers
		if kit.Status.RuntimeProvider == nil && context.Catalog.RuntimeProvider != nil ||
			kit.Status.RuntimeProvider != nil && context.Catalog.RuntimeProvider == nil ||
			kit.Status.RuntimeProvider != nil && context.Catalog.RuntimeProvider != nil &&
				(kit.Status.RuntimeProvider.Quarkus != nil && context.Catalog.RuntimeProvider.Quarkus == nil ||
					kit.Status.RuntimeProvider.Quarkus == nil && context.Catalog.RuntimeProvider.Quarkus != nil ||
					*kit.Status.RuntimeProvider.Quarkus != *context.Catalog.RuntimeProvider.Quarkus) {
			continue
		}

		if kit.Status.Phase != v1alpha1.IntegrationKitPhaseReady || kit.Labels == nil {
			continue
		}
		if kitType, present := kit.Labels["camel.apache.org/kit.type"]; !present || kitType != v1alpha1.IntegrationKitTypePlatform {
			continue
		}

		images = append(images, publishedImage{
			Image:        kit.Status.Image,
			Artifacts:    kit.Status.Artifacts,
			Dependencies: kit.Spec.Dependencies,
		})
	}
	return images, nil
}

func findBestImage(images []publishedImage, artifacts []v1alpha1.Artifact) (publishedImage, map[string]bool) {
	var bestImage publishedImage

	if len(images) == 0 {
		return bestImage, nil
	}

	requiredLibs := make(map[string]bool, len(artifacts))
	for _, entry := range artifacts {
		requiredLibs[entry.ID] = true
	}

	bestImageCommonLibs := make(map[string]bool)
	bestImageSurplusLibs := 0

	for _, image := range images {
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
