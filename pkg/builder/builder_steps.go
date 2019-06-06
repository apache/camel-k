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

	"github.com/apache/camel-k/pkg/util/kubernetes"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/scylladb/go-set/strset"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/apache/camel-k/pkg/util/tar"

	yaml2 "gopkg.in/yaml.v2"

	"github.com/pkg/errors"
)

var stepsByID = make(map[string]Step)

func init() {
	RegisterSteps(Steps)
}

type steps struct {
	GenerateProject         Step
	GenerateProjectSettings Step
	InjectDependencies      Step
	SanitizeDependencies    Step
	ComputeDependencies     Step
	StandardPackager        Step
	IncrementalPackager     Step
}

// Steps --
var Steps = steps{
	GenerateProject: NewStep(
		ProjectGenerationPhase,
		generateProject,
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
	ComputeDependencies: NewStep(
		ProjectBuildPhase,
		computeDependencies,
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

// generateProject --
func generateProject(ctx *Context) error {
	p, err := NewMavenProject(ctx)
	if err != nil {
		return err
	}

	ctx.Maven.Project = p
	//
	// set-up dependencies
	//
	ctx.Maven.Project.AddDependencyGAV("org.apache.camel.k", "camel-k-runtime-jvm", ctx.Build.RuntimeVersion)

	for _, d := range ctx.Build.Dependencies {
		switch {
		case strings.HasPrefix(d, "camel:"):
			artifactID := strings.TrimPrefix(d, "camel:")

			if !strings.HasPrefix(artifactID, "camel-") {
				artifactID = "camel-" + artifactID
			}

			ctx.Maven.Project.AddDependencyGAV("org.apache.camel", artifactID, "")
		case strings.HasPrefix(d, "camel-k:"):
			artifactID := strings.TrimPrefix(d, "camel-k:")

			if !strings.HasPrefix(artifactID, "camel-") {
				artifactID = "camel-" + artifactID
			}

			ctx.Maven.Project.AddDependencyGAV("org.apache.camel.k", artifactID, ctx.Build.RuntimeVersion)
		case strings.HasPrefix(d, "mvn:"):
			mid := strings.TrimPrefix(d, "mvn:")
			gav := strings.Replace(mid, "/", ":", -1)

			ctx.Maven.Project.AddEncodedDependencyGAV(gav)
		case strings.HasPrefix(d, "runtime:"):
			artifactID := strings.Replace(d, "runtime:", "camel-k-runtime-", 1)

			ctx.Maven.Project.AddDependencyGAV("org.apache.camel.k", artifactID, ctx.Build.RuntimeVersion)
		case strings.HasPrefix(d, "bom:"):
			// no-op
		default:
			return fmt.Errorf("unknown dependency type: %s", d)
		}
	}

	// Add Log4j 2 SLF4J binding as default logging impl
	ctx.Maven.Project.AddDependency(maven.Dependency{
		GroupID:    "org.apache.logging.log4j",
		ArtifactID: "log4j-slf4j-impl",
		Version:    "2.11.2",
		Scope:      "runtime",
	})

	return nil
}

// generateProjectSettings --
func generateProjectSettings(ctx *Context) error {
	val, err := kubernetes.ResolveValueSource(ctx.C, ctx.Client, ctx.Namespace, &ctx.Build.Platform.Build.Maven.Settings)
	if err != nil {
		return err
	}
	if val != "" {
		ctx.Maven.SettingsData = []byte(val)
	}

	return nil
}

func injectDependencies(ctx *Context) error {
	//
	// Add dependencies from catalog
	//
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
	//
	// post process dependencies
	//
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

		switch dep.GroupID {
		case "org.apache.camel":
			//
			// Remove the version so we force using the one configured by the bom
			//
			ctx.Maven.Project.Dependencies[i].Version = ""
		case "org.apache.camel.k":
			//
			// Force every runtime dependency to have the required version discardin
			// any version eventually set on the catalog
			//
			ctx.Maven.Project.Dependencies[i].Version = ctx.Build.RuntimeVersion
		}
	}

	return nil
}

func computeDependencies(ctx *Context) error {
	mc := maven.NewContext(path.Join(ctx.Path, "maven"), ctx.Maven.Project)
	mc.Settings = ctx.Maven.Settings
	mc.SettingsData = ctx.Maven.SettingsData
	mc.AddArguments(maven.ExtraOptions(ctx.Build.Platform.Build.LocalRepository)...)
	mc.AddArgumentf("org.apache.camel.k:camel-k-maven-plugin:%s:generate-dependency-list", ctx.Build.RuntimeVersion)

	if err := maven.Run(mc); err != nil {
		return errors.Wrap(err, "failure while determining classpath")
	}

	dependencies := path.Join(mc.Path, "target", "dependencies.yaml")
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

		bestImage, commonLibs := findBestImage(images, ctx.Build.Dependencies, ctx.Artifacts)
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

	for _, entry := range ctx.Resources {
		if err := tarAppender.AddData(entry.Content, entry.Target); err != nil {
			return err
		}
	}

	ctx.Archive = tarFileName

	return nil
}

func listPublishedImages(context *Context) ([]publishedImage, error) {
	list := v1alpha1.NewIntegrationKitList()

	err := context.Client.List(context.C, &k8sclient.ListOptions{Namespace: context.Namespace}, &list)
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
		if kit.Status.RuntimeVersion != context.Build.RuntimeVersion {
			continue
		}
		if kit.Status.Phase != v1alpha1.IntegrationKitPhaseReady || kit.Labels == nil {
			continue
		}
		if ctxType, present := kit.Labels["camel.apache.org/kit.type"]; !present || ctxType != v1alpha1.IntegrationKitTypePlatform {
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

func findBestImage(images []publishedImage, dependencies []string, artifacts []v1alpha1.Artifact) (publishedImage, map[string]bool) {
	var bestImage publishedImage

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
