package runtime

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/maven"
)

// QuarkusSteps --
var QuarkusSteps = []builder.Step{
	Steps.LoadCamelQuarkusCatalog,
	Steps.GenerateQuarkusProject,
	Steps.ComputeQuarkusDependencies,
}

func loadCamelQuarkusCatalog(ctx *builder.Context) error {
	catalog, err := camel.LoadCatalog(ctx.C, ctx.Client, ctx.Build.Meta.Namespace, ctx.Build.CamelVersion, ctx.Build.RuntimeVersion, ctx.Build.RuntimeProvider.Quarkus)
	if err != nil {
		return err
	}

	if catalog == nil {
		return fmt.Errorf("unable to find catalog matching version requirement: camel=%s, runtime=%s, camel-quarkus=%s, quarkus=%s",
			ctx.Build.CamelVersion, ctx.Build.RuntimeVersion, ctx.Build.RuntimeProvider.Quarkus.CamelQuarkusVersion, ctx.Build.RuntimeProvider.Quarkus.QuarkusVersion)
	}

	ctx.Catalog = catalog

	return nil
}

func generateQuarkusProject(ctx *builder.Context) error {
	p := maven.NewProjectWithGAV("org.apache.camel.k.integration", "camel-k-integration", defaults.Version)
	p.Properties = ctx.Build.Platform.Build.Properties
	p.DependencyManagement = &maven.DependencyManagement{Dependencies: make([]maven.Dependency, 0)}
	p.Dependencies = make([]maven.Dependency, 0)
	p.Build = &maven.Build{Plugins: make([]maven.Plugin, 0)}

	// DependencyManagement
	p.DependencyManagement.Dependencies = append(p.DependencyManagement.Dependencies,
		maven.Dependency{
			GroupID:    "org.apache.camel.quarkus",
			ArtifactID: "camel-quarkus-bom",
			Version:    ctx.Build.RuntimeProvider.Quarkus.CamelQuarkusVersion,
			Type:       "pom",
			Scope:      "import",
		},
		maven.Dependency{
			GroupID:    "org.apache.camel.k",
			ArtifactID: "camel-k-runtime-bom",
			Version:    ctx.Build.RuntimeVersion,
			Type:       "pom",
			Scope:      "import",
		},
	)

	// Plugins
	p.Build.Plugins = append(p.Build.Plugins,
		maven.Plugin{
			GroupID:    "io.quarkus",
			ArtifactID: "quarkus-bootstrap-maven-plugin",
			Version:    ctx.Build.RuntimeProvider.Quarkus.QuarkusVersion,
		},
		maven.Plugin{
			GroupID:    "io.quarkus",
			ArtifactID: "quarkus-maven-plugin",
			Version:    ctx.Build.RuntimeProvider.Quarkus.QuarkusVersion,
			Executions: []maven.Execution{
				{
					Goals: []string{
						"build",
					},
				},
			},
		},
	)

	ctx.Maven.Project = p

	return nil
}

func computeQuarkusDependencies(ctx *builder.Context) error {
	mc := maven.NewContext(path.Join(ctx.Path, "maven"), ctx.Maven.Project)
	mc.SettingsContent = ctx.Maven.SettingsData
	mc.LocalRepository = ctx.Build.Platform.Build.Maven.LocalRepository
	mc.Timeout = ctx.Build.Platform.Build.Maven.Timeout.Duration

	// Build the project, as the quarkus-bootstrap plugin build-tree goal
	// requires the artifact to be installed
	mc.AddArgument("install")
	if err := maven.Run(mc); err != nil {
		return errors.Wrap(err, "failure while building project")
	}

	// Call the Quarkus dependencies plugin
	mc.AdditionalArguments = nil
	mc.AddArguments("quarkus-bootstrap:build-tree")
	output := new(bytes.Buffer)
	// TODO: improve logging while capturing output
	mc.Stdout = output
	if err := maven.Run(mc); err != nil {
		return errors.Wrap(err, "failure while determining dependencies")
	}

	scanner := bufio.NewScanner(output)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		gav, err := maven.ParseGAV(scanner.Text())
		if err != nil {
			continue
		}

		fileName := gav.GroupID + "." + gav.ArtifactID + "-" + gav.Version + "." + gav.Type
		location := path.Join(mc.Path, "target", "lib", fileName)
		_, err = os.Stat(location)
		// We check that the dependency actually exists in the lib directory as the
		// quarkus-bootstrap Maven plugin reports deployment dependencies as well
		if os.IsNotExist(err) {
			continue
		}

		ctx.Artifacts = append(ctx.Artifacts, v1alpha1.Artifact{
			ID:       gav.GroupID + ":" + gav.ArtifactID + ":" + gav.Type + ":" + gav.Version,
			Location: location,
			Target:   path.Join("lib", fileName),
		})
	}

	runner := "camel-k-integration-" + defaults.Version + "-runner.jar"
	ctx.Artifacts = append(ctx.Artifacts, v1alpha1.Artifact{
		ID:       runner,
		Location: path.Join(mc.Path, "target", runner),
		Target:   runner,
	})

	return nil
}
