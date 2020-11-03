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
	"reflect"
	"strings"

	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/jitpack"

	"github.com/rs/xid"

	"github.com/apache/camel-k/pkg/util/controller"
	"k8s.io/apimachinery/pkg/selection"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/maven"
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
	StandardImageContext    Step
	IncrementalImageContext Step
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
	StandardImageContext: NewStep(
		ApplicationPackagePhase,
		standardImageContext,
	),
	IncrementalImageContext: NewStep(
		ApplicationPackagePhase,
		incrementalImageContext,
	),
}

// DefaultSteps --
var DefaultSteps = []Step{
	Steps.CleanBuildDir,
	Steps.GenerateProjectSettings,
	Steps.InjectDependencies,
	Steps.SanitizeDependencies,
	Steps.IncrementalImageContext,
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
	return InjectDependenciesCommon(&ctx.Maven.Project, ctx.Build.Dependencies, ctx.Catalog)
}

// InjectDependenciesCommon --
func InjectDependenciesCommon(
	project *maven.Project,
	dependencies []string,
	catalog *camel.RuntimeCatalog) error {

	// Add dependencies from build
	for _, d := range dependencies {
		switch {
		case strings.HasPrefix(d, "bom:"):
			mid := strings.TrimPrefix(d, "bom:")
			gav := strings.Replace(mid, "/", ":", -1)

			d, err := maven.ParseGAV(gav)
			if err != nil {
				return err
			}

			project.DependencyManagement.Dependencies = append(project.DependencyManagement.Dependencies, maven.Dependency{
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

			project.AddDependencyGAV("org.apache.camel", artifactID, "")
		case strings.HasPrefix(d, "camel-k:"):
			artifactID := strings.TrimPrefix(d, "camel-k:")

			if !strings.HasPrefix(artifactID, "camel-k-") {
				artifactID = "camel-k-" + artifactID
			}

			project.AddDependencyGAV("org.apache.camel.k", artifactID, "")
		case strings.HasPrefix(d, "camel-quarkus:"):
			artifactID := strings.TrimPrefix(d, "camel-quarkus:")

			if !strings.HasPrefix(artifactID, "camel-quarkus-") {
				artifactID = "camel-quarkus-" + artifactID
			}

			project.AddDependencyGAV("org.apache.camel.quarkus", artifactID, "")
		case strings.HasPrefix(d, "mvn:"):
			mid := strings.TrimPrefix(d, "mvn:")
			gav := strings.Replace(mid, "/", ":", -1)

			project.AddEncodedDependencyGAV(gav)
		default:
			if dep := jitpack.ToDependency(d); dep != nil {
				project.AddDependency(*dep)

				addRepo := true
				for _, repo := range project.Repositories {
					if repo.URL == jitpack.RepoURL {
						addRepo = false
						break
					}
				}
				if addRepo {
					project.Repositories = append(project.Repositories, maven.Repository{
						ID:  "jitpack.io-" + xid.New().String(),
						URL: jitpack.RepoURL,
						Releases: maven.RepositoryPolicy{
							Enabled:        true,
							ChecksumPolicy: "fail",
						},
						Snapshots: maven.RepositoryPolicy{
							Enabled:        true,
							ChecksumPolicy: "fail",
						},
					})
				}
			} else {
				return fmt.Errorf("unknown dependency type: %s", d)
			}
		}
	}

	// Add dependencies from catalog
	deps := make([]maven.Dependency, len(project.Dependencies))
	copy(deps, project.Dependencies)

	for _, d := range deps {
		if a, ok := catalog.Artifacts[d.ArtifactID]; ok {
			for _, dep := range a.Dependencies {
				md := maven.Dependency{
					GroupID:    dep.GroupID,
					ArtifactID: dep.ArtifactID,
				}

				project.AddDependency(md)

				for _, e := range dep.Exclusions {
					me := maven.Exclusion{
						GroupID:    e.GroupID,
						ArtifactID: e.ArtifactID,
					}

					project.AddDependencyExclusion(md, me)
				}
			}
		}
	}

	// Post process dependencies
	deps = make([]maven.Dependency, len(project.Dependencies))
	copy(deps, project.Dependencies)

	for _, d := range deps {
		if a, ok := catalog.Artifacts[d.ArtifactID]; ok {
			md := maven.Dependency{
				GroupID:    a.GroupID,
				ArtifactID: a.ArtifactID,
			}

			for _, e := range a.Exclusions {
				me := maven.Exclusion{
					GroupID:    e.GroupID,
					ArtifactID: e.ArtifactID,
				}

				project.AddDependencyExclusion(md, me)
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

func standardImageContext(ctx *Context) error {
	return imageContext(ctx, func(ctx *Context) error {
		ctx.SelectedArtifacts = ctx.Artifacts

		return nil
	})
}

func incrementalImageContext(ctx *Context) error {
	if ctx.HasRequiredImage() {
		//
		// If the build requires a specific image, don't try to determine the
		// base image using artifact so just use the standard packages
		//
		return standardImageContext(ctx)
	}

	images, err := listPublishedImages(ctx)
	if err != nil {
		return err
	}

	return imageContext(ctx, func(ctx *Context) error {
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
		}

		return nil
	})
}

func imageContext(ctx *Context, selector artifactsSelector) error {
	err := selector(ctx)
	if err != nil {
		return err
	}

	contextDir := path.Join(ctx.Path, "context")

	err = os.MkdirAll(contextDir, 0777)
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
			return nil
		}
	}

	// #nosec G202
	dockerfile := []byte(`
		FROM ` + ctx.BaseImage + `
		ADD . /deployments
		USER 1000
	`)

	err = ioutil.WriteFile(path.Join(contextDir, "Dockerfile"), dockerfile, 0777)
	if err != nil {
		return err
	}

	return nil
}

func listPublishedImages(context *Context) ([]v1.IntegrationKitStatus, error) {
	options := []k8sclient.ListOption{
		k8sclient.InNamespace(context.Namespace),
		k8sclient.MatchingLabels{
			"camel.apache.org/runtime.version":  context.Catalog.Runtime.Version,
			"camel.apache.org/runtime.provider": string(context.Catalog.Runtime.Provider),
		},
		controller.NewLabelSelector("camel.apache.org/kit.type", selection.Equals, []string{
			v1.IntegrationKitTypePlatform,
		}),
	}

	list := v1.NewIntegrationKitList()
	err := context.Client.List(context.C, &list, options...)
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
			//
			// If the Artifact's checksum is not defined we can't reliably determine if for some
			// reason the artifact has been changed but not the ID (as example for snapshots or
			// other generated jar) thus we do not take this artifact into account.
			//
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
