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

package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/kubernetes/scheme"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/cmd/source"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/resource"
)

func newCmdRun(rootCmdOptions *RootCmdOptions) (*cobra.Command, *runCmdOptions) {
	options := runCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:               "run [file to run]",
		Short:             "Build and run the Integration on Kubernetes.",
		Long:              `Build and run the Integration on Kubernetes.`,
		Args:              options.validateArgs,
		PersistentPreRunE: options.decode,
		PreRunE:           options.preRun,
		RunE:              options.run,
		PostRunE:          options.postRun,
		Annotations:       make(map[string]string),
	}

	cmd.Flags().String("name", "", "The integration name")
	cmd.Flags().String("image", "", "An image built externally (ie, via CICD). Enabling it will skip the Integration build phase.")
	cmd.Flags().StringArrayP("dependency", "d", nil, "A dependency that should be included, e.g., \"camel:mail\" for a Camel component, "+
		"\"mvn:org.my:app:1.0\" for a Maven dependency")
	cmd.Flags().StringP("kit", "k", "", "The kit used to run the integration")
	cmd.Flags().StringArrayP("property", "p", nil, "Add a runtime property (syntax: [my-key=my-value])")
	cmd.Flags().StringArray("build-property", nil, "Add a build time property (syntax: [my-key=my-value])")
	cmd.Flags().StringArray("config", nil, "Add a runtime configuration from a Configmap or a Secret "+
		"(syntax: [configmap|secret]:name[/key], where name represents the configmap/secret name and key optionally "+
		"represents the configmap/secret key to be filtered)")
	cmd.Flags().StringArray("resource", nil, "Add a runtime resource from a Configmap or a Secret "+
		"(syntax: [configmap|secret]:name[/key][@path], where name represents the configmap/secret name, "+
		"key optionally represents the configmap/secret key to be filtered and path represents the destination path)")
	cmd.Flags().StringArray("maven-repository", nil, "Add a maven repository")
	cmd.Flags().String("integration-profile", "", "Integration profile used for deployment")
	cmd.Flags().StringArrayP("trait", "t", nil, "Configure a trait. E.g. \"-t service.enabled=false\"")
	cmd.Flags().StringP("output", "o", "", "Output format. One of: json|yaml")
	cmd.Flags().StringArrayP("volume", "v", nil, "Mount a volume into the integration container. E.g \"-v pvcname:/container/path\"")
	cmd.Flags().StringArray("annotation", nil, "Add an annotation to the integration. E.g. \"--annotation my.company=hello\"")
	cmd.Flags().StringArray("label", nil, "Add a label to the integration. E.g. \"--label my.company=hello\"")
	cmd.Flags().StringArray("source", nil, "Add source file to your integration, "+
		"this is added to the list of files listed as arguments of the command")
	cmd.Flags().String("service-account", "", "The SA to use to run this Integration")
	cmd.Flags().String("git", "", "A Git repository containing the project to build.")
	cmd.Flags().String("git-branch", "", "Git branch to checkout when using --git option")
	cmd.Flags().String("git-tag", "", "Git tag to checkout when using --git option")
	cmd.Flags().String("git-commit", "", "Git commit (full SHA) to checkout when using --git option")
	cmd.Flags().String("git-path", "", "Git path (likely a project module) to checkout when using --git option")
	cmd.Flags().Bool("save", false, "Save the run parameters into the default kamel configuration file (kamel-config.yaml)")
	cmd.Flags().Bool("dont-run-after-build", false, "Only build, don't run the application. "+
		"You can run \"kamel deploy\" to run a built Integration.")

	return &cmd, &options
}

type runCmdOptions struct {
	*RootCmdOptions `json:"-"`

	Save               bool     `kamel:"omitsave"                    mapstructure:"save" yaml:",omitempty"`
	IntegrationName    string   `mapstructure:"name"                 yaml:",omitempty"`
	ContainerImage     string   `mapstructure:"image"                yaml:",omitempty"`
	GitRepo            string   `mapstructure:"git"                  yaml:",omitempty"`
	GitBranch          string   `mapstructure:"git-branch"           yaml:",omitempty"`
	GitTag             string   `mapstructure:"git-tag"              yaml:",omitempty"`
	GitCommit          string   `mapstructure:"git-commit"           yaml:",omitempty"`
	GitPath            string   `mapstructure:"git-path"             yaml:",omitempty"`
	IntegrationProfile string   `mapstructure:"integration-profile"  yaml:",omitempty"`
	OutputFormat       string   `mapstructure:"output"               yaml:",omitempty"`
	ServiceAccount     string   `mapstructure:"service-account"      yaml:",omitempty"`
	Resources          []string `mapstructure:"resources"            yaml:",omitempty"`
	Dependencies       []string `mapstructure:"dependencies"         yaml:",omitempty"`
	Properties         []string `mapstructure:"properties"           yaml:",omitempty"`
	BuildProperties    []string `mapstructure:"build-properties"     yaml:",omitempty"`
	Configs            []string `mapstructure:"configs"              yaml:",omitempty"`
	Repositories       []string `mapstructure:"maven-repositories"   yaml:",omitempty"`
	Traits             []string `mapstructure:"traits"               yaml:",omitempty"`
	Volumes            []string `mapstructure:"volumes"              yaml:",omitempty"`
	Labels             []string `mapstructure:"labels"               yaml:",omitempty"`
	Annotations        []string `mapstructure:"annotations"          yaml:",omitempty"`
	Sources            []string `mapstructure:"sources"              yaml:",omitempty"`
	DontRunAfterBuild  bool     `mapstructure:"dont-run-after-build" yaml:",omitempty"`
}

func (o *runCmdOptions) decode(cmd *cobra.Command, args []string) error {
	// *************************************************************************
	//
	// WARNING: this is an hack, well a huge one
	//
	// When the run command runs, it performs two steps:
	//
	// 1. load from kamel.run
	// 2. load from kamel.run.integration.$name
	//
	// the values loaded from the second steps belong to a node for which there
	// are no flags as it is a dynamic node not known when the command hierarchy
	// is initialized and configured so any flag value is simple ignored and the
	// struct field takes the value of the persisted configuration node.
	//
	// *************************************************************************

	// load from kamel.run (1)
	pathToRoot := pathToRoot(cmd)

	if err := decodeKey(o, pathToRoot, o.Flags.AllSettings()); err != nil {
		return err
	}

	if o.OutputFormat != "" {
		// let the command work in offline mode
		cmd.Annotations[offlineCommandLabel] = strconv.FormatBool(true)
	}

	// backup the values from values belonging to kamel.run by coping the
	// structure by values, which in practice is done by a marshal/unmarshal
	// to/from json.
	bkp := runCmdOptions{}
	if err := clone(&bkp, o); err != nil {
		return err
	}

	name, err := o.GetIntegrationName(args)
	if err != nil {
		return err
	}
	if name != "" {
		// load from kamel.run.integration.$name (2)
		pathToRoot += ".integration." + name
		if err := decodeKey(o, pathToRoot, o.Flags.AllSettings()); err != nil {
			return err
		}

		rdata := reflect.ValueOf(&bkp).Elem()
		idata := reflect.ValueOf(o).Elem()

		// iterate over all the flags that have been set and if so, copy the
		// value from the backed-up structure over the new one that has been
		// decoded from the kamel.run.integration.$name node
		cmd.Flags().Visit(func(flag *pflag.Flag) {
			if f, ok := fieldByMapstructureTagName(rdata, flag.Name); ok {
				rfield := rdata.FieldByName(f.Name)
				ifield := idata.FieldByName(f.Name)

				ifield.Set(rfield)
			}
		})
	}

	return o.validate(cmd)
}

func (o *runCmdOptions) validateArgs(cmd *cobra.Command, args []string) error {
	if _, err := source.Resolve(context.Background(), args, cmd); err != nil {
		return fmt.Errorf("one of the provided sources is not reachable: %w", err)
	}

	return nil
}

func (o *runCmdOptions) validate(cmd *cobra.Command) error {
	for _, volume := range o.Volumes {
		volumeConfig := strings.Split(volume, ":")
		if len(volumeConfig) != 2 || len(strings.TrimSpace(volumeConfig[0])) == 0 || len(strings.TrimSpace(volumeConfig[1])) == 0 {
			return fmt.Errorf("volume '%s' is invalid, it should be in the format: pvcname:/container/path", volume)
		}
	}

	for _, label := range o.Labels {
		parts := strings.Split(label, "=")
		if len(parts) != 2 {
			return fmt.Errorf(`invalid label specification %s. Expected "<labelkey>=<labelvalue>"`, label)
		}
	}

	for _, annotation := range o.Annotations {
		parts := strings.SplitN(annotation, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf(`invalid annotation specification %s. Expected "<annotationkey>=<annotationvalue>"`, annotation)
		}
	}

	var client client.Client
	var err error

	if !isOfflineCommand(cmd) {
		client, err = o.GetCmdClient()
		if err != nil {
			return err
		}
	}
	catalog := trait.NewCatalog(client)

	return trait.ValidateTraits(catalog, extractTraitNames(o.Traits))
}

func (o *runCmdOptions) run(cmd *cobra.Command, args []string) error {
	var c client.Client
	var err error
	if !isOfflineCommand(cmd) {
		c, err = o.GetCmdClient()
		if err != nil {
			return err
		}
	}

	// We need to make this check at this point, in order to have sources filled during decoding
	if (len(args) < 1 && len(o.Sources) < 1) && o.isManaged() {
		return errors.New("run command expects either an Integration source, a container image " +
			"(via --image argument) or a git repository (via --git argument)")
	}

	_, err = o.createOrUpdateIntegration(cmd, c, args)

	return err
}

func (o *runCmdOptions) postRun(cmd *cobra.Command, args []string) error {
	if o.Save {
		rootKey := pathToRoot(cmd)
		name, err := o.GetIntegrationName(args)
		if err != nil {
			return err
		}
		if name != "" {
			key := fmt.Sprintf("%s.integration.%s", rootKey, name)

			cfg, err := LoadConfiguration()
			if err != nil {
				return err
			}

			cfg.Update(cmd, key, o, false)

			return cfg.Save()
		}
	}

	return nil
}

func (o *runCmdOptions) createOrUpdateIntegration(cmd *cobra.Command, c client.Client, sources []string) (*v1.Integration, error) {
	namespace := o.Namespace
	name, err := o.GetIntegrationName(sources)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, errors.New("unable to determine integration name")
	}

	integration, existing, err := o.getIntegration(cmd, c, namespace, name)
	if err != nil {
		return nil, err
	}

	integration.Spec = v1.IntegrationSpec{
		Dependencies:  make([]string, 0, len(o.Dependencies)),
		Configuration: make([]v1.ConfigurationSpec, 0),
		Repositories:  o.Repositories,
	}

	o.applyLabels(integration)
	o.applyAnnotations(integration)

	//nolint:gocritic
	if o.isManaged() {
		// Resolve resources
		if err := o.resolveSources(cmd, sources, integration); err != nil {
			return nil, err
		}
	} else if o.ContainerImage != "" {
		// Self Managed Integration as the user provided a container image built externally
		o.Traits = append(o.Traits, "container.image="+o.ContainerImage)
	} else if o.GitRepo != "" {
		if o.GitBranch != "" && o.GitTag != "" {
			err := errors.New("illegal arguments: cannot specify both git branch and tag")

			return nil, err
		}
		if o.GitBranch != "" && o.GitCommit != "" {
			err := errors.New("illegal arguments: cannot specify both git branch and commit")

			return nil, err
		}
		if o.GitTag != "" && o.GitCommit != "" {
			err := errors.New("illegal arguments: cannot specify both git tag and commit")

			return nil, err
		}
		integration.Spec.Git = &v1.GitConfigSpec{
			URL:    o.GitRepo,
			Tag:    o.GitTag,
			Branch: o.GitBranch,
			Commit: o.GitCommit,
			Path:   o.GitPath,
		}
	} else {
		return nil, errors.New("you must provide a source, an image or a git repository parameters")
	}

	if err := o.convertOptionsToTraits(); err != nil {
		return nil, err
	}

	if err := o.applyDependencies(cmd, integration); err != nil {
		return nil, err
	}

	if len(o.Traits) > 0 {
		catalog := trait.NewCatalog(c)
		if err := trait.ConfigureTraits(o.Traits, &integration.Spec.Traits, catalog); err != nil {
			return nil, err
		}
	}

	if o.ServiceAccount != "" {
		integration.Spec.ServiceAccountName = o.ServiceAccount
	}

	if o.OutputFormat != "" {
		return nil, showIntegrationOutput(cmd, integration, o.OutputFormat)
	}

	if existing == nil {
		err = c.Create(o.Context, integration)
		if err != nil {
			return nil, err
		}
		fmt.Fprintln(cmd.OutOrStdout(), `Integration "`+name+`" created`)
	} else {
		patch := ctrl.MergeFrom(existing)
		d, err := patch.Data(integration)
		if err != nil {
			return nil, err
		}

		if string(d) == "{}" {
			fmt.Fprintln(cmd.OutOrStdout(), `Integration "`+name+`" unchanged`)

			return integration, nil
		}
		err = c.Patch(o.Context, integration, patch)
		if err != nil {
			return nil, err
		}
		fmt.Fprintln(cmd.OutOrStdout(), `Integration "`+name+`" updated`)
	}

	return integration, nil
}

func (o *runCmdOptions) isManaged() bool {
	return o.ContainerImage == "" && o.GitRepo == ""
}

func showIntegrationOutput(cmd *cobra.Command, integration *v1.Integration, outputFormat string) error {
	printer := printers.NewTypeSetter(scheme.Scheme)
	printer.Delegate = &kubernetes.CLIPrinter{
		Format: outputFormat,
	}

	return printer.PrintObj(integration, cmd.OutOrStdout())
}

func (o *runCmdOptions) getIntegration(cmd *cobra.Command, c client.Client, namespace, name string) (*v1.Integration, *v1.Integration, error) {
	it := &v1.Integration{
		Kind:       v1.IntegrationKind,
		APIVersion: v1.SchemeGroupVersion.String(),
		Namespace:  namespace,
		Name:       name,
	}

	existing := &v1.Integration{}
	if !isOfflineCommand(cmd) {
		err := c.Get(o.Context, ctrl.ObjectKeyFromObject(it), existing)
		switch {
		case err == nil:
			it = existing.DeepCopy()
		case k8serrors.IsNotFound(err):
			existing = nil
		default:
			return nil, nil, err
		}
	}

	return it, existing, nil
}

func (o *runCmdOptions) applyLabels(it *v1.Integration) {
	for _, label := range o.Labels {
		parts := strings.Split(label, "=")
		if len(parts) == 2 {
			if it.Labels == nil {
				it.Labels = make(map[string]string)
			}
			it.Labels[parts[0]] = parts[1]
		}
	}
}

func (o *runCmdOptions) applyAnnotations(it *v1.Integration) {
	if it.Annotations == nil {
		it.Annotations = make(map[string]string)
	}

	// --integration-profile={id} is a syntax sugar for '--annotation camel.apache.org/integration-profile.id={id}'
	if o.IntegrationProfile != "" {
		v1.SetAnnotation(&it.ObjectMeta, v1.IntegrationProfileAnnotation, o.IntegrationProfile)
	}

	for _, annotation := range o.Annotations {
		parts := strings.SplitN(annotation, "=", 2)
		if len(parts) == 2 {
			it.Annotations[parts[0]] = parts[1]
		}
	}
	if o.DontRunAfterBuild {
		it.Annotations[v1.IntegrationDontRunAfterBuildAnnotation] = strconv.FormatBool(true)
	}
}

func (o *runCmdOptions) resolveSources(cmd *cobra.Command, sources []string, it *v1.Integration) error {
	srcs := make([]string, 0, len(sources)+len(o.Sources))
	srcs = append(srcs, sources...)
	srcs = append(srcs, o.Sources...)

	resolvedSources, err := source.Resolve(context.Background(), srcs, cmd)
	if err != nil {
		return err
	}

	for _, source := range resolvedSources {
		if source.IsYaml() {
			flows, err := v1.FromYamlDSLString(source.Content)
			if err != nil {
				return err
			}
			it.Spec.AddFlows(flows...)
		} else {
			it.Spec.AddSources(v1.SourceSpec{
				Name:    source.Name,
				Content: source.Content,
			})
		}
	}

	return nil
}

func (o *runCmdOptions) convertOptionsToTraits() error {
	if err := o.parseAndConvertToTrait(o.Resources, resource.ParseResource,
		func(c *resource.Config) string { return c.String() },
		"mount.resources"); err != nil {
		return err
	}
	if err := o.parseAndConvertToTrait(o.Configs, resource.ParseConfig,
		func(c *resource.Config) string { return c.String() },
		"mount.configs"); err != nil {
		return err
	}

	o.applyProperties(o.Properties, "camel.properties")
	o.applyProperties(o.BuildProperties, "builder.properties")

	for _, item := range o.Volumes {
		o.Traits = append(o.Traits, "mount.volumes="+item)
	}

	return nil
}

func (o *runCmdOptions) parseAndConvertToTrait(
	params []string,
	parse func(string) (*resource.Config, error),
	convert func(*resource.Config,
	) string,
	traitParam string,
) error {
	for _, param := range params {
		config, err := parse(param)
		if err != nil {
			return err
		}
		o.Traits = append(o.Traits, convertToTrait(convert(config), traitParam))
	}

	return nil
}

func convertToTrait(value, traitParameter string) string {
	return fmt.Sprintf("%s=%s", traitParameter, value)
}

func (o *runCmdOptions) applyProperties(props []string, traitName string) {
	if len(props) == 0 {
		return
	}

	for _, val := range props {
		kv := sanitizeKeyValue(val)
		propsTrait := o.convertToTraitParameter(kv, traitName)
		o.Traits = append(o.Traits, propsTrait)
	}
}

func sanitizeKeyValue(val string) string {
	parts := strings.SplitN(val, "=", 2)
	if len(parts) != 2 {
		return ""
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	return key + "=" + value
}

func (o *runCmdOptions) convertToTraitParameter(keyValue, traitParameter string) string {
	return fmt.Sprintf("%s=%s", traitParameter, keyValue)
}

func (o *runCmdOptions) applyDependencies(cmd *cobra.Command, it *v1.Integration) error {
	var catalog *camel.RuntimeCatalog
	for _, item := range o.Dependencies {
		if catalog == nil {
			// The catalog used for lightweight validation of Camel components.
			// The exact runtime version is not used here since resolving the runtime version may be
			// a costly operation and most of the use cases should be covered by the default catalog.
			// And the validation only warns potential misusage of Camel components at the CLI level,
			// so strictness of catalog version is not necessary here.
			var err error
			catalog, err = createCamelCatalog()
			if err != nil {
				return err
			}
			if catalog == nil {
				return errors.New("error trying to load the default Camel catalog")
			}
		}
		addDependency(cmd, it, item, catalog)
	}

	return nil
}

func (o *runCmdOptions) GetIntegrationName(sources []string) (string, error) {
	name := ""
	switch {
	case o.IntegrationName != "":
		name = o.IntegrationName
		name = kubernetes.SanitizeName(name)
	case len(sources) == 1:
		name = kubernetes.SanitizeName(sources[0])
	case o.ContainerImage != "":
		// Self managed build execution
		name = kubernetes.SanitizeName(strings.ReplaceAll(o.ContainerImage, ":", "-v"))
	case o.GitRepo != "":
		gitRepoName, err := getRepoName(o.GitRepo)
		if err != nil {
			return "", err
		}
		name = kubernetes.SanitizeName(gitRepoName)
	}

	return name, nil
}

// getRepoName extracts the repository name from the given Git URL.
func getRepoName(repoURL string) (string, error) {
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	repoPath := parsedURL.Path
	repoName := path.Base(repoPath)
	repoName = strings.TrimSuffix(repoName, ".git")

	return repoName, nil
}
