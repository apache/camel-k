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
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path"
	"reflect"
	"strings"
	"syscall"

	"github.com/magiconair/properties"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/kubernetes/scheme"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/cmd/source"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	k8slog "github.com/apache/camel-k/v2/pkg/util/kubernetes/log"
	"github.com/apache/camel-k/v2/pkg/util/property"
	"github.com/apache/camel-k/v2/pkg/util/resource"
	"github.com/apache/camel-k/v2/pkg/util/sync"
	"github.com/apache/camel-k/v2/pkg/util/watch"
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
	cmd.Flags().BoolP("wait", "w", false, "Wait for the integration to be running")
	cmd.Flags().StringP("kit", "k", "", "The kit used to run the integration")
	cmd.Flags().StringArrayP("property", "p", nil, "Add a runtime property or a local properties file from a path "+
		"(syntax: [my-key=my-value|file:/path/to/my-conf.properties])")
	cmd.Flags().StringArray("build-property", nil, "Add a build time property or properties file from a path "+
		"(syntax: [my-key=my-value|file:/path/to/my-conf.properties])")
	cmd.Flags().StringArray("config", nil, "Add a runtime configuration from a Configmap or a Secret "+
		"(syntax: [configmap|secret]:name[/key], where name represents the configmap/secret name and key optionally "+
		"represents the configmap/secret key to be filtered)")
	cmd.Flags().StringArray("resource", nil, "Add a runtime resource from a Configmap or a Secret "+
		"(syntax: [configmap|secret]:name[/key][@path], where name represents the configmap/secret name, "+
		"key optionally represents the configmap/secret key to be filtered and path represents the destination path)")
	cmd.Flags().StringArray("maven-repository", nil, "Add a maven repository")
	cmd.Flags().Bool("logs", false, "Print integration logs")
	cmd.Flags().Bool("sync", false, "Synchronize the local source file with the cluster, republishing at each change")
	cmd.Flags().Bool("dev", false, "Enable Dev mode (equivalent to \"-w --logs --sync\")")
	cmd.Flags().Bool("use-flows", true, "Write yaml sources as Flow objects in the integration custom resource")
	cmd.Flags().StringP("operator-id", "x", "camel-k", "Operator id selected to manage this integration.")
	cmd.Flags().String("profile", "", "Trait profile used for deployment")
	cmd.Flags().String("integration-profile", "", "Integration profile used for deployment")
	cmd.Flags().StringArrayP("trait", "t", nil, "Configure a trait. E.g. \"-t service.enabled=false\"")
	cmd.Flags().StringP("output", "o", "", "Output format. One of: json|yaml")
	cmd.Flags().Bool("compression", false, "Enable storage of sources and resources as a compressed binary blobs")
	cmd.Flags().StringArrayP("volume", "v", nil, "Mount a volume into the integration container. E.g \"-v pvcname:/container/path\"")
	cmd.Flags().StringArrayP("env", "e", nil, "Set an environment variable in the integration container. E.g \"-e MY_VAR=my-value\"")
	cmd.Flags().StringArray("annotation", nil, "Add an annotation to the integration. E.g. \"--annotation my.company=hello\"")
	cmd.Flags().StringArray("label", nil, "Add a label to the integration. E.g. \"--label my.company=hello\"")
	cmd.Flags().StringArray("source", nil, "Add source file to your integration, "+
		"this is added to the list of files listed as arguments of the command")
	cmd.Flags().String("pod-template", "", "The path of the YAML file containing a PodSpec template to be used for the Integration pods")
	cmd.Flags().String("service-account", "", "The SA to use to run this Integration")
	cmd.Flags().String("git", "", "A Git repository containing the project to build.")
	cmd.Flags().String("git-branch", "", "Git branch to checkout when using --git option")
	cmd.Flags().String("git-tag", "", "Git tag to checkout when using --git option")
	cmd.Flags().String("git-commit", "", "Git commit (full SHA) to checkout when using --git option")
	cmd.Flags().Bool("save", false, "Save the run parameters into the default kamel configuration file (kamel-config.yaml)")
	cmd.Flags().Bool("dont-run-after-build", false, "Only build, don't run the application. "+
		"You can run \"kamel deploy\" to run a built Integration.")

	return &cmd, &options
}

type runCmdOptions struct {
	*RootCmdOptions `json:"-"`

	// Deprecated: won't be supported in the future
	Compression bool `mapstructure:"compression" yaml:",omitempty"`
	Wait        bool `mapstructure:"wait" yaml:",omitempty"`
	Logs        bool `mapstructure:"logs" yaml:",omitempty"`
	Sync        bool `mapstructure:"sync" yaml:",omitempty"`
	Dev         bool `mapstructure:"dev" yaml:",omitempty"`
	UseFlows    bool `mapstructure:"use-flows" yaml:",omitempty"`
	Save        bool `mapstructure:"save" yaml:",omitempty" kamel:"omitsave"`
	// Deprecated: won't be supported in the future
	IntegrationKit     string `mapstructure:"kit" yaml:",omitempty"`
	IntegrationName    string `mapstructure:"name" yaml:",omitempty"`
	ContainerImage     string `mapstructure:"image" yaml:",omitempty"`
	GitRepo            string `mapstructure:"git" yaml:",omitempty"`
	GitBranch          string `mapstructure:"git-branch" yaml:",omitempty"`
	GitTag             string `mapstructure:"git-tag" yaml:",omitempty"`
	GitCommit          string `mapstructure:"git-commit" yaml:",omitempty"`
	Profile            string `mapstructure:"profile" yaml:",omitempty"`
	IntegrationProfile string `mapstructure:"integration-profile" yaml:",omitempty"`
	OperatorID         string `mapstructure:"operator-id" yaml:",omitempty"`
	OutputFormat       string `mapstructure:"output" yaml:",omitempty"`
	// Deprecated: won't be supported in the future
	PodTemplate    string   `mapstructure:"pod-template" yaml:",omitempty"`
	ServiceAccount string   `mapstructure:"service-account" yaml:",omitempty"`
	Resources      []string `mapstructure:"resources" yaml:",omitempty"`
	Dependencies   []string `mapstructure:"dependencies" yaml:",omitempty"`
	// Deprecated: won't be supported in the future
	Properties []string `mapstructure:"properties" yaml:",omitempty"`
	// Deprecated: won't be supported in the future
	BuildProperties []string `mapstructure:"build-properties" yaml:",omitempty"`
	Configs         []string `mapstructure:"configs" yaml:",omitempty"`
	Repositories    []string `mapstructure:"maven-repositories" yaml:",omitempty"`
	Traits          []string `mapstructure:"traits" yaml:",omitempty"`
	Volumes         []string `mapstructure:"volumes" yaml:",omitempty"`
	// Deprecated: won't be supported in the future
	EnvVars           []string `mapstructure:"envs" yaml:",omitempty"`
	Labels            []string `mapstructure:"labels" yaml:",omitempty"`
	Annotations       []string `mapstructure:"annotations" yaml:",omitempty"`
	Sources           []string `mapstructure:"sources" yaml:",omitempty"`
	DontRunAfterBuild bool     `mapstructure:"dont-run-after-build" yaml:",omitempty"`
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

	//nolint:goconst
	if o.OutputFormat != "" {
		// let the command work in offline mode
		cmd.Annotations[offlineCommandLabel] = "true"
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
	if _, err := source.Resolve(context.Background(), args, false, cmd); err != nil {
		return fmt.Errorf("one of the provided sources is not reachable: %w", err)
	}

	return nil
}

func (o *runCmdOptions) validate(cmd *cobra.Command) error {
	if o.OperatorID == "" {
		return fmt.Errorf("cannot use empty operator id")
	}

	for _, volume := range o.Volumes {
		volumeConfig := strings.Split(volume, ":")
		if len(volumeConfig) != 2 || len(strings.TrimSpace(volumeConfig[0])) == 0 || len(strings.TrimSpace(volumeConfig[1])) == 0 {
			return fmt.Errorf("volume '%s' is invalid, it should be in the format: pvcname:/container/path", volume)
		}
	}

	propertyFiles := filterBuildPropertyFiles(o.Properties)
	propertyFiles = append(propertyFiles, filterBuildPropertyFiles(o.BuildProperties)...)
	err := validatePropertyFiles(propertyFiles)
	if err != nil {
		return err
	}

	if o.OutputFormat != "" && o.Dev {
		return fmt.Errorf("cannot use --dev with -o/--output option")
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

	for i, property := range o.Properties {
		// We support only --config
		if strings.HasPrefix(property, "configmap:") || strings.HasPrefix(property, "secret:") {
			o.Configs = append(o.Configs, property)
			// clean it to avoid further processing
			o.Properties[i] = ""
			fmt.Fprintf(cmd.OutOrStdout(), "Property %s is deprecated: use --config %s instead\n", property, property)
		}
	}

	for _, bp := range o.BuildProperties {
		// Deprecated: to be removed
		if strings.HasPrefix(bp, "configmap:") || strings.HasPrefix(bp, "secret:") {
			fmt.Fprintf(cmd.OutOrStdout(), "Build property %s is deprecated. It will be removed from future releases.\n", bp)
		}
	}

	// Deprecated: to be removed
	if o.Compression {
		fmt.Fprintf(cmd.OutOrStdout(), "Compression property is deprecated. It will be removed from future releases.\n")
	}

	var client client.Client
	if !isOfflineCommand(cmd) {
		client, err = o.GetCmdClient()
		if err != nil {
			return err
		}
	}
	catalog := trait.NewCatalog(client)

	return trait.ValidateTraits(catalog, extractTraitNames(o.Traits))
}

func filterBuildPropertyFiles(maybePropertyFiles []string) []string {
	var propertyFiles []string
	for _, maybePropertyFile := range maybePropertyFiles {
		if strings.HasPrefix(maybePropertyFile, "file:") {
			propertyFiles = append(propertyFiles, strings.Replace(maybePropertyFile, "file:", "", 1))
		}
	}

	return propertyFiles
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

	integration, err := o.createOrUpdateIntegration(cmd, c, args)
	if err != nil {
		return err
	}

	if o.Dev {
		cs := make(chan os.Signal, 1)
		signal.Notify(cs, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-cs
			if o.Context.Err() != nil {
				// Context canceled
				return
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Run integration terminating")
			err := DeleteIntegration(o.Context, c, integration.Name, integration.Namespace)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				os.Exit(1)
			}
			os.Exit(0)
		}()
	}

	if o.Sync || o.Dev {
		err = o.syncIntegration(cmd, c, args)
		if err != nil {
			return err
		}
	}
	if o.Logs || o.Dev || o.Wait {
		//nolint:errcheck
		go watch.HandleIntegrationEvents(o.Context, c, integration, func(event *corev1.Event) bool {
			fmt.Fprintln(cmd.OutOrStdout(), event.Message)
			return true
		})
	}
	if o.Wait || o.Dev {
		for {
			integrationPhase, err := o.waitForIntegrationReady(cmd, c, integration)
			if err != nil {
				return err
			}

			if integrationPhase == nil || *integrationPhase == v1.IntegrationPhaseError {
				return fmt.Errorf("integration \"%s\" deployment failed", integration.Name)
			} else if *integrationPhase == v1.IntegrationPhaseRunning {
				break
			}

			// The integration watch timed out so recreate it using the latest integration resource version
			existing := v1.NewIntegration(integration.Namespace, integration.Name)
			err = c.Get(o.Context, ctrl.ObjectKeyFromObject(&existing), &existing)
			if err != nil {
				return err
			}

			integration.ResourceVersion = existing.ResourceVersion
		}
	}
	if o.Logs || o.Dev {
		err = k8slog.Print(o.Context, cmd, c, integration, nil, cmd.OutOrStdout())
		if err != nil {
			return err
		}
	}

	if o.Sync || o.Logs || o.Dev {
		// Let's add a Wait point, otherwise the script terminates
		<-o.RootContext.Done()
	}

	return nil
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

func (o *runCmdOptions) waitForIntegrationReady(cmd *cobra.Command, c client.Client, integration *v1.Integration) (*v1.IntegrationPhase, error) {
	handler := func(i *v1.Integration) bool {
		//
		// TODO when we add health checks, we should Wait until they are passed
		//
		if i.Status.Phase != "" {
			// TODO remove this log when we make sure that events are always created
			fmt.Fprintf(cmd.OutOrStdout(), "Progress: integration %q in phase %s\n", integration.Name, string(i.Status.Phase))
		}
		if i.Status.Phase == v1.IntegrationPhaseRunning || i.Status.Phase == v1.IntegrationPhaseError {
			return false
		}

		return true
	}

	return watch.HandleIntegrationStateChanges(o.Context, c, integration, handler)
}

func (o *runCmdOptions) syncIntegration(cmd *cobra.Command, c client.Client, sources []string) error {
	// Let's watch all relevant files when in dev mode
	var files []string
	files = append(files, sources...)
	files = append(files, filterFileLocation(o.Resources)...)
	files = append(files, filterFileLocation(o.Configs)...)
	files = append(files, filterFileLocation(o.Properties)...)
	files = append(files, filterFileLocation(o.BuildProperties)...)

	for _, s := range files {
		ok, err := source.IsLocalAndFileExists(s)
		if err != nil {
			return err
		}
		if ok {
			changes, err := sync.File(o.Context, s)
			if err != nil {
				return err
			}
			go func() {
				for {
					select {
					case <-o.Context.Done():
						return
					case <-changes:
						// let's create a new command to parse modeline changes and update our integration
						newCmd, _, err := createKamelWithModelineCommand(o.RootContext, os.Args[1:])
						newCmd.SetOut(cmd.OutOrStdout())
						newCmd.SetErr(cmd.ErrOrStderr())
						if err != nil {
							fmt.Fprintln(newCmd.ErrOrStderr(), "Unable to sync integration: ", err.Error())

							continue
						}
						newCmd.Args = o.validateArgs
						newCmd.PreRunE = o.decode
						newCmd.RunE = func(cmd *cobra.Command, args []string) error {
							_, err := o.createOrUpdateIntegration(cmd, c, sources)
							return err
						}
						newCmd.PostRunE = nil

						// cancel the existing command to release watchers
						o.ContextCancel()
						// run the new one
						err = newCmd.Execute()
						if err != nil {
							fmt.Fprintln(newCmd.ErrOrStderr(), "Unable to sync integration: ", err.Error())
						}
					}
				}
			}()
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: the following URL will not be watched for changes: %s\n", s)
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

	var integrationKit *corev1.ObjectReference
	if o.IntegrationKit != "" {
		integrationKit = &corev1.ObjectReference{
			Namespace: namespace,
			Name:      o.IntegrationKit,
		}
	}

	integration.Spec = v1.IntegrationSpec{
		Dependencies:   make([]string, 0, len(o.Dependencies)),
		IntegrationKit: integrationKit,
		Configuration:  make([]v1.ConfigurationSpec, 0),
		Repositories:   o.Repositories,
		Profile:        v1.TraitProfileByName(o.Profile),
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
		o.Traits = append(o.Traits, fmt.Sprintf("container.image=%s", o.ContainerImage))
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
		}
	} else {
		return nil, errors.New("you must provide a source, an image or a git repository parameters")
	}

	if err := resolvePodTemplate(context.Background(), cmd, o.PodTemplate, &integration.Spec); err != nil {
		return nil, err
	}

	if err := o.convertOptionsToTraits(cmd, c, integration); err != nil {
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
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.IntegrationKind,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
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

	// --operator-id={id} is a syntax sugar for '--annotation camel.apache.org/operator.id={id}'
	it.SetOperatorID(strings.TrimSpace(o.OperatorID))

	// --integration-profile={id} is a syntax sugar for '--annotation camel.apache.org/integration-profile.id={id}'
	if o.IntegrationProfile != "" {
		if strings.Contains(o.IntegrationProfile, "/") {
			namespacedName := strings.SplitN(o.IntegrationProfile, "/", 2)
			v1.SetAnnotation(&it.ObjectMeta, v1.IntegrationProfileNamespaceAnnotation, namespacedName[0])
			v1.SetAnnotation(&it.ObjectMeta, v1.IntegrationProfileAnnotation, namespacedName[1])
		} else {
			v1.SetAnnotation(&it.ObjectMeta, v1.IntegrationProfileAnnotation, o.IntegrationProfile)
		}
	}

	for _, annotation := range o.Annotations {
		parts := strings.SplitN(annotation, "=", 2)
		if len(parts) == 2 {
			it.Annotations[parts[0]] = parts[1]
		}
	}
	if o.DontRunAfterBuild {
		it.Annotations[v1.IntegrationDontRunAfterBuildAnnotation] = "true"
	}
}

func (o *runCmdOptions) resolveSources(cmd *cobra.Command, sources []string, it *v1.Integration) error {
	srcs := make([]string, 0, len(sources)+len(o.Sources))
	srcs = append(srcs, sources...)
	srcs = append(srcs, o.Sources...)

	resolvedSources, err := source.Resolve(context.Background(), srcs, o.Compression, cmd)
	if err != nil {
		return err
	}

	for _, source := range resolvedSources {
		if o.UseFlows && !o.Compression && source.IsYaml() {
			flows, err := v1.FromYamlDSLString(source.Content)
			if err != nil {
				return err
			}
			it.Spec.AddFlows(flows...)
		} else {
			it.Spec.AddSources(v1.SourceSpec{
				DataSpec: v1.DataSpec{
					Name:        source.Name,
					Content:     source.Content,
					Compression: source.Compress,
				},
			})
		}
	}

	return nil
}

func (o *runCmdOptions) convertOptionsToTraits(cmd *cobra.Command, c client.Client, it *v1.Integration) error {
	if err := o.parseAndConvertToTrait(cmd, c, it, o.Resources, resource.ParseResource,
		func(c *resource.Config) string { return c.String() },
		"mount.resources"); err != nil {
		return err
	}
	if err := o.parseAndConvertToTrait(cmd, c, it, o.Configs, resource.ParseConfig,
		func(c *resource.Config) string { return c.String() },
		"mount.configs"); err != nil {
		return err
	}

	if err := o.applyProperties(c, o.Properties, "camel.properties"); err != nil {
		return err
	}

	if err := o.applyProperties(c, o.BuildProperties, "builder.properties"); err != nil {
		return err
	}

	for _, item := range o.Volumes {
		o.Traits = append(o.Traits, fmt.Sprintf("mount.volumes=%s", item))
	}
	for _, item := range o.EnvVars {
		o.Traits = append(o.Traits, fmt.Sprintf("environment.vars=%s", item))
	}

	return nil
}

func (o *runCmdOptions) parseAndConvertToTrait(cmd *cobra.Command,
	c client.Client, integration *v1.Integration, params []string,
	parse func(string) (*resource.Config, error),
	convert func(*resource.Config) string,
	traitParam string,
) error {
	for _, param := range params {
		config, err := parse(param)
		if err != nil {
			return err
		}
		if o.OutputFormat == "" {
			if err := parseConfig(o.Context, cmd, c, config, integration); err != nil {
				return err
			}
		}
		o.Traits = append(o.Traits, convertToTrait(convert(config), traitParam))
	}
	return nil
}

func convertToTrait(value, traitParameter string) string {
	return fmt.Sprintf("%s=%s", traitParameter, value)
}

func (o *runCmdOptions) applyProperties(c client.Client, items []string, traitName string) error {
	if len(items) == 0 {
		return nil
	}
	props, err := o.mergePropertiesWithPrecedence(c, items)
	if err != nil {
		return err
	}
	for _, key := range props.Keys() {
		val, _ := props.Get(key)
		kv := fmt.Sprintf("%s=%s", key, val)
		propsTraits, err := o.convertToTraitParameter(c, kv, traitName)
		if err != nil {
			return err
		}
		o.Traits = append(o.Traits, propsTraits...)
	}

	return nil
}

func (o *runCmdOptions) convertToTraitParameter(c client.Client, value, traitParameter string) ([]string, error) {
	traits := make([]string, 0)
	props, err := o.extractProperties(c, value)
	if err != nil {
		return nil, err
	}
	props.DisableExpansion = true
	for _, k := range props.Keys() {
		v, ok := props.Get(k)
		if ok {
			entry, err := property.EncodePropertyFileEntry(k, v)
			if err != nil {
				return nil, err
			}
			traits = append(traits, fmt.Sprintf("%s=%s", traitParameter, entry))
		} else {
			return nil, err
		}
	}

	return traits, nil
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
				return fmt.Errorf("error trying to load the default Camel catalog")
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

func (o *runCmdOptions) mergePropertiesWithPrecedence(c client.Client, items []string) (*properties.Properties, error) {
	loPrecedenceProps := properties.NewProperties()
	loPrecedenceProps.DisableExpansion = true
	hiPrecedenceProps := properties.NewProperties()
	hiPrecedenceProps.DisableExpansion = true
	for _, item := range items {
		prop, err := o.extractProperties(c, item)
		if err != nil {
			return nil, err
		}
		prop.DisableExpansion = true
		// We consider file, secret and config map props to have a lower priority versus single properties
		if strings.HasPrefix(item, "file:") || strings.HasPrefix(item, "secret:") || strings.HasPrefix(item, "configmap:") {
			loPrecedenceProps.Merge(prop)
		} else {
			hiPrecedenceProps.Merge(prop)
		}
	}
	// Any property contained in both collections will be merged
	// giving precedence to the ones in hiPrecedenceProps
	loPrecedenceProps.Merge(hiPrecedenceProps)
	return loPrecedenceProps, nil
}

// The function parse the value and if it is a file (file:/path/), it will parse as property file
// otherwise return a single property built from the item passed as `key=value`.
func (o *runCmdOptions) extractProperties(c client.Client, value string) (*properties.Properties, error) {
	switch {
	case strings.HasPrefix(value, "file:"):
		// we already validated the existence of files during validate()
		return loadPropertyFile(strings.Replace(value, "file:", "", 1))
	case strings.HasPrefix(value, "secret:"):
		return loadPropertiesFromSecret(o.Context, c, o.Namespace, strings.Replace(value, "secret:", "", 1))
	case strings.HasPrefix(value, "configmap:"):
		return loadPropertiesFromConfigMap(o.Context, c, o.Namespace, strings.Replace(value, "configmap:", "", 1))
	default:
		return keyValueProps(value)
	}
}

func loadPropertyFile(fileName string) (*properties.Properties, error) {
	file, err := util.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	p, err := properties.Load(file, properties.UTF8)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Deprecated: to be removed in future releases.
func resolvePodTemplate(ctx context.Context, cmd *cobra.Command, templateSrc string, spec *v1.IntegrationSpec) error {
	// check if template is set
	if templateSrc == "" {
		return nil
	}
	var template v1.PodSpec

	// check if value is a path to the file
	if _, err := os.Stat(templateSrc); err == nil {
		rsc, err := source.Resolve(ctx, []string{templateSrc}, false, cmd)
		if err == nil && len(rsc) > 0 {
			templateSrc = rsc[0].Content
		}
	}
	// template is inline
	templateBytes := []byte(templateSrc)

	jsonTemplate, err := yaml.ToJSON(templateBytes)
	if err != nil {
		jsonTemplate = templateBytes
	}
	err = json.Unmarshal(jsonTemplate, &template)

	if err == nil {
		spec.PodTemplate = &v1.PodSpecTemplate{
			Spec: template,
		}
	}
	return err
}
