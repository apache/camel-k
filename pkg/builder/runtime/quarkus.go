package runtime

import (
	"bufio"
	"bytes"
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
	Steps.GenerateQuarkusProject,
	Steps.ComputeQuarkusDependencies,
}

func generateQuarkusProject(ctx *builder.Context) error {
	// Catalog
	if ctx.Catalog == nil {
		c, err := camel.LoadCatalog(ctx.C, ctx.Client, ctx.Namespace, ctx.Build.Platform.Build.CamelVersion, ctx.Build.Platform.Build.RuntimeVersion)
		if err != nil {
			return err
		}

		ctx.Catalog = c
	}

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
			Version:    "0.2.0",
			//Version:    ctx.Catalog.Version,
			Type:  "pom",
			Scope: "import",
		},
		maven.Dependency{
			GroupID:    "org.apache.camel.k",
			ArtifactID: "camel-k-runtime-bom",
			Version:    ctx.Catalog.RuntimeVersion,
			Type:       "pom",
			Scope:      "import",
		},
	)

	// Plugins
	p.Build.Plugins = append(p.Build.Plugins,
		maven.Plugin{
			GroupID:    "io.quarkus",
			ArtifactID: "quarkus-bootstrap-maven-plugin",
			// TODO: must be the same as the version required by camel-k-runtime
			Version: "0.21.2",
		},
		maven.Plugin{
			GroupID:    "io.quarkus",
			ArtifactID: "quarkus-maven-plugin",
			// TODO: must be the same as the version required by camel-k-runtime
			Version: "0.21.2",
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
	mc.LocalRepository = ctx.Build.Platform.Build.LocalRepository
	mc.Timeout = ctx.Build.Platform.Build.Maven.Timeout.Duration

	// Build the project, as the quarkus-bootstrap plugin build-tree goal
	// requires the artifact to be installed
	mc.AddArgument("package")
	if err := maven.Run(mc); err != nil {
		return errors.Wrap(err, "failure while building project")
	}

	// Call the Quarkus dependencies plugin
	mc.AdditionalArguments = nil
	mc.AddArguments("quarkus-bootstrap:build-tree")
	output := new(bytes.Buffer)
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
			ID:       fileName,
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
