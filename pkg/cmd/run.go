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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"strings"
	"syscall"

	spectrum "github.com/container-tools/spectrum/pkg/builder"
	"github.com/magiconair/properties"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/printers"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/dsl"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	k8slog "github.com/apache/camel-k/pkg/util/kubernetes/log"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/property"
	"github.com/apache/camel-k/pkg/util/resource"
	"github.com/apache/camel-k/pkg/util/sync"
	"github.com/apache/camel-k/pkg/util/watch"
)

var traitConfigRegexp = regexp.MustCompile(`^([a-z0-9-]+)((?:\.[a-z0-9-]+)(?:\[[0-9]+\]|\.[A-Za-z0-9-_]+)*)=(.*)$`)

func newCmdRun(rootCmdOptions *RootCmdOptions) (*cobra.Command, *runCmdOptions) {
	options := runCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:      "run [file to run]",
		Short:    "Run a integration on Kubernetes",
		Long:     `Deploys and execute a integration pod on Kubernetes.`,
		Args:     options.validateArgs,
		PreRunE:  options.decode,
		RunE:     options.run,
		PostRunE: options.postRun,
	}

	cmd.Flags().String("name", "", "The integration name")
	cmd.Flags().StringArrayP("connect", "c", nil, "A Service that the integration should bind to, specified as [[apigroup/]version:]kind:[namespace/]name")
	cmd.Flags().StringArrayP("dependency", "d", nil, "A dependency that should be included, e.g., \"-d camel-mail\" for a Camel component, or \"-d mvn:org.my:app:1.0\" for a Maven dependency")
	cmd.Flags().BoolP("wait", "w", false, "Wait for the integration to be running")
	cmd.Flags().StringP("kit", "k", "", "The kit used to run the integration")
	cmd.Flags().StringArrayP("property", "p", nil, "Add a runtime property or properties file (syntax: [my-key=my-value|file:/path/to/my-conf.properties])")
	cmd.Flags().StringArray("build-property", nil, "Add a build time property or properties file (syntax: [my-key=my-value|file:/path/to/my-conf.properties])")
	cmd.Flags().StringArray("config", nil, "Add a runtime configuration from a Configmap, a Secret or a file (syntax: [configmap|secret|file]:name[/key], where name represents the local file path or the configmap/secret name and key optionally represents the configmap/secret key to be filtered)")
	cmd.Flags().StringArray("resource", nil, "Add a runtime resource from a Configmap, a Secret or a file (syntax: [configmap|secret|file]:name[/key][@path], where name represents the local file path or the configmap/secret name, key optionally represents the configmap/secret key to be filtered and path represents the destination path)")
	cmd.Flags().StringArray("maven-repository", nil, "Add a maven repository")
	cmd.Flags().Bool("logs", false, "Print integration logs")
	cmd.Flags().Bool("sync", false, "Synchronize the local source file with the cluster, republishing at each change")
	cmd.Flags().Bool("dev", false, "Enable Dev mode (equivalent to \"-w --logs --sync\")")
	cmd.Flags().Bool("use-flows", true, "Write yaml sources as Flow objects in the integration custom resource")
	cmd.Flags().String("profile", "", "Trait profile used for deployment")
	cmd.Flags().StringArrayP("trait", "t", nil, "Configure a trait. E.g. \"-t service.enabled=false\"")
	cmd.Flags().StringP("output", "o", "", "Output format. One of: json|yaml")
	cmd.Flags().Bool("compression", false, "Enable storage of sources and resources as a compressed binary blobs")
	cmd.Flags().StringArray("open-api", nil, "Add an OpenAPI spec (syntax: [configmap|file]:name)")
	cmd.Flags().StringArrayP("volume", "v", nil, "Mount a volume into the integration container. E.g \"-v pvcname:/container/path\"")
	cmd.Flags().StringArrayP("env", "e", nil, "Set an environment variable in the integration container. E.g \"-e MY_VAR=my-value\"")
	cmd.Flags().StringArray("annotation", nil, "Add an annotation to the integration. E.g. \"--annotation my.company=hello\"")
	cmd.Flags().StringArray("label", nil, "Add a label to the integration. E.g. \"--label my.company=hello\"")
	cmd.Flags().StringArray("source", nil, "Add source file to your integration, this is added to the list of files listed as arguments of the command")
	cmd.Flags().String("pod-template", "", "The path of the YAML file containing a PodSpec template to be used for the Integration pods")

	cmd.Flags().Bool("save", false, "Save the run parameters into the default kamel configuration file (kamel-config.yaml)")

	// completion support
	configureKnownCompletions(&cmd)

	return &cmd, &options
}

type runCmdOptions struct {
	*RootCmdOptions `json:"-"`
	Compression     bool     `mapstructure:"compression" yaml:",omitempty"`
	Wait            bool     `mapstructure:"wait" yaml:",omitempty"`
	Logs            bool     `mapstructure:"logs" yaml:",omitempty"`
	Sync            bool     `mapstructure:"sync" yaml:",omitempty"`
	Dev             bool     `mapstructure:"dev" yaml:",omitempty"`
	UseFlows        bool     `mapstructure:"use-flows" yaml:",omitempty"`
	Save            bool     `mapstructure:"save" yaml:",omitempty" kamel:"omitsave"`
	IntegrationKit  string   `mapstructure:"kit" yaml:",omitempty"`
	IntegrationName string   `mapstructure:"name" yaml:",omitempty"`
	Profile         string   `mapstructure:"profile" yaml:",omitempty"`
	OutputFormat    string   `mapstructure:"output" yaml:",omitempty"`
	PodTemplate     string   `mapstructure:"pod-template" yaml:",omitempty"`
	Connects        []string `mapstructure:"connects" yaml:",omitempty"`
	Resources       []string `mapstructure:"resources" yaml:",omitempty"`
	OpenAPIs        []string `mapstructure:"open-apis" yaml:",omitempty"`
	Dependencies    []string `mapstructure:"dependencies" yaml:",omitempty"`
	Properties      []string `mapstructure:"properties" yaml:",omitempty"`
	BuildProperties []string `mapstructure:"build-properties" yaml:",omitempty"`
	Configs         []string `mapstructure:"configs" yaml:",omitempty"`
	Repositories    []string `mapstructure:"maven-repositories" yaml:",omitempty"`
	Traits          []string `mapstructure:"traits" yaml:",omitempty"`
	Volumes         []string `mapstructure:"volumes" yaml:",omitempty"`
	EnvVars         []string `mapstructure:"envs" yaml:",omitempty"`
	Labels          []string `mapstructure:"labels" yaml:",omitempty"`
	Annotations     []string `mapstructure:"annotations" yaml:",omitempty"`
	Sources         []string `mapstructure:"sources" yaml:",omitempty"`
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
	// struct field takes tha value of the the persisted configuration node.
	//
	// *************************************************************************

	// load from kamel.run (1)
	pathToRoot := pathToRoot(cmd)
	if err := decodeKey(o, pathToRoot); err != nil {
		return err
	}

	if err := o.validate(); err != nil {
		return err
	}

	// backup the values from values belonging to kamel.run by coping the
	// structure by values, which in practice is done by a marshal/unmarshal
	// to/from json.
	bkp := runCmdOptions{}
	if err := clone(&bkp, o); err != nil {
		return err
	}

	name := o.GetIntegrationName(args)
	if name != "" {
		// load from kamel.run.integration.$name (2)
		pathToRoot += ".integration." + name
		if err := decodeKey(o, pathToRoot); err != nil {
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

	return o.validate()
}

func (o *runCmdOptions) validateArgs(_ *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("run expects at least 1 argument, received 0")
	}

	if _, err := ResolveSources(context.Background(), args, false); err != nil {
		return errors.Wrap(err, "One of the provided sources is not reachable")
	}

	return nil
}

func (o *runCmdOptions) validate() error {
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

	for _, openapi := range o.OpenAPIs {
		// We support only local file and cluster configmaps
		if !(strings.HasPrefix(openapi, "file:") || strings.HasPrefix(openapi, "configmap:")) {
			return fmt.Errorf(`invalid openapi specification "%s". It supports only file or configmap`, openapi)
		}
	}

	client, err := o.GetCmdClient()
	if err != nil {
		return err
	}
	catalog := trait.NewCatalog(client)

	return validateTraits(catalog, o.Traits)
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
	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}

	catalog := trait.NewCatalog(c)
	integration, err := o.createOrUpdateIntegration(cmd, c, args, catalog)
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
			fmt.Printf("Run integration terminating\n")
			err := DeleteIntegration(o.Context, c, integration.Name, integration.Namespace)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			os.Exit(0)
		}()
	}

	if o.Sync || o.Dev {
		err = o.syncIntegration(cmd, c, args, catalog)
		if err != nil {
			return err
		}
	}
	if o.Logs || o.Dev || o.Wait {
		// nolint: errcheck
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

			integration.ObjectMeta.ResourceVersion = existing.ObjectMeta.ResourceVersion
		}
	}
	if o.Logs || o.Dev {
		err = k8slog.Print(o.Context, c, integration, cmd.OutOrStdout())
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
		name := o.GetIntegrationName(args)
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

func (o *runCmdOptions) syncIntegration(cmd *cobra.Command, c client.Client, sources []string, catalog trait.Finder) error {
	// Let's watch all relevant files when in dev mode
	var files []string
	files = append(files, sources...)
	files = append(files, filterFileLocation(o.Resources)...)
	files = append(files, filterFileLocation(o.Configs)...)
	files = append(files, filterFileLocation(o.Properties)...)
	files = append(files, filterFileLocation(o.BuildProperties)...)
	files = append(files, filterFileLocation(o.OpenAPIs)...)

	for _, s := range files {
		ok, err := isLocalAndFileExists(s)
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
							fmt.Println("Unable to sync integration: ", err.Error())

							continue
						}
						newCmd.Args = o.validateArgs
						newCmd.PreRunE = o.decode
						newCmd.RunE = func(cmd *cobra.Command, args []string) error {
							_, err := o.createOrUpdateIntegration(cmd, c, sources, catalog)
							return err
						}
						newCmd.PostRunE = nil

						// cancel the existing command to release watchers
						o.ContextCancel()
						// run the new one
						err = newCmd.Execute()
						if err != nil {
							fmt.Println("Unable to sync integration: ", err.Error())
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

// nolint: gocyclo
func (o *runCmdOptions) createOrUpdateIntegration(cmd *cobra.Command, c client.Client, sources []string, catalog trait.Finder) (*v1.Integration, error) {
	namespace := o.Namespace
	name := o.GetIntegrationName(sources)

	if name == "" {
		return nil, errors.New("unable to determine integration name")
	}

	integration := &v1.Integration{
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
	err := c.Get(o.Context, ctrl.ObjectKeyFromObject(integration), existing)
	switch {
	case err == nil:
		integration = existing.DeepCopy()
	case k8serrors.IsNotFound(err):
		existing = nil
	default:
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

	for _, label := range o.Labels {
		parts := strings.Split(label, "=")
		if len(parts) == 2 {
			if integration.Labels == nil {
				integration.Labels = make(map[string]string)
			}
			integration.Labels[parts[0]] = parts[1]
		}
	}

	for _, annotation := range o.Annotations {
		parts := strings.SplitN(annotation, "=", 2)
		if len(parts) == 2 {
			if integration.Annotations == nil {
				integration.Annotations = make(map[string]string)
			}
			integration.Annotations[parts[0]] = parts[1]
		}
	}

	srcs := make([]string, 0, len(sources)+len(o.Sources))
	srcs = append(srcs, sources...)
	srcs = append(srcs, o.Sources...)

	resolvedSources, err := ResolveSources(context.Background(), srcs, o.Compression)
	if err != nil {
		return nil, err
	}

	for _, source := range resolvedSources {
		if o.UseFlows && !o.Compression && (strings.HasSuffix(source.Name, ".yaml") || strings.HasSuffix(source.Name, ".yml")) {
			flows, err := dsl.FromYamlDSLString(source.Content)
			if err != nil {
				return nil, err
			}
			integration.Spec.AddFlows(flows...)
		} else {
			integration.Spec.AddSources(v1.SourceSpec{
				DataSpec: v1.DataSpec{
					Name:        source.Name,
					Content:     source.Content,
					Compression: source.Compress,
				},
			})
		}
	}

	err = resolvePodTemplate(context.Background(), o.PodTemplate, &integration.Spec)
	if err != nil {
		return nil, err
	}

	err = o.parseAndConvertToTrait(c, integration, o.Resources, resource.ParseResource, func(c *resource.Config) string { return c.String() }, "mount.resources")
	if err != nil {
		return nil, err
	}
	err = o.parseAndConvertToTrait(c, integration, o.Configs, resource.ParseConfig, func(c *resource.Config) string { return c.String() }, "mount.configs")
	if err != nil {
		return nil, err
	}
	err = o.parseAndConvertToTrait(c, integration, o.OpenAPIs, resource.ParseConfig, func(c *resource.Config) string { return c.Name() }, "openapi.configmaps")
	if err != nil {
		return nil, err
	}

	var platform *v1.IntegrationPlatform

	for _, item := range o.Dependencies {
		// TODO: accept URLs
		// TODO: accept other resources through Maven types (i.e not just JARs)
		if strings.HasPrefix(item, "file://") && strings.HasSuffix(item, ".jar") {
			if platform == nil {
				// let's also enable the registry trait
				o.Traits = append(o.Traits, "registry.enabled=true")
				platform, err = getPlatform(c, o.Context, integration.Namespace)
				if err != nil {
					return nil, err
				}
			}
			if err := uploadDependency(platform, item, name, cmd, integration); err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("Error trying to upload %s to the Image Registry.", item))
			}
		} else {
			integration.Spec.AddDependency(item)
		}
	}

	props, err := mergePropertiesWithPrecedence(o.Properties)
	if err != nil {
		return nil, err
	}
	for _, key := range props.Keys() {
		kv := fmt.Sprintf("%s=%s", key, props.GetString(key, ""))
		propsTraits, err := convertToTraitParameter(kv, "camel.properties")
		if err != nil {
			return nil, err
		}
		o.Traits = append(o.Traits, propsTraits...)
	}

	// convert each build configuration to a builder trait property
	buildProps, err := mergePropertiesWithPrecedence(o.BuildProperties)
	if err != nil {
		return nil, err
	}
	for _, key := range buildProps.Keys() {
		kv := fmt.Sprintf("%s=%s", key, buildProps.GetString(key, ""))
		buildPropsTraits, err := convertToTraitParameter(kv, "builder.properties")
		if err != nil {
			return nil, err
		}
		o.Traits = append(o.Traits, buildPropsTraits...)
	}

	for _, item := range o.Volumes {
		o.Traits = append(o.Traits, fmt.Sprintf("mount.volumes=%s", item))
	}
	for _, item := range o.EnvVars {
		o.Traits = append(o.Traits, fmt.Sprintf("environment.vars=%s", item))
	}
	for _, item := range o.Connects {
		o.Traits = append(o.Traits, fmt.Sprintf("service-binding.services=%s", item))
	}
	if len(o.Traits) > 0 {
		traits, err := configureTraits(o.Traits, catalog)
		if err != nil {
			return nil, err
		}
		integration.Spec.Traits = traits
	}

	if o.OutputFormat != "" {
		return nil, showIntegrationOutput(cmd, integration, o.OutputFormat, c.GetScheme())
	}

	if existing == nil {
		err = c.Create(o.Context, integration)
		fmt.Printf("Integration \"%s\" created\n", name)
	} else {
		err = c.Patch(o.Context, integration, ctrl.MergeFromWithOptions(existing, ctrl.MergeFromWithOptimisticLock{}))
		fmt.Printf("Integration \"%s\" updated\n", name)
	}

	if err != nil {
		return nil, err
	}

	return integration, nil
}

func showIntegrationOutput(cmd *cobra.Command, integration *v1.Integration, outputFormat string, scheme runtime.ObjectTyper) error {
	printer := printers.NewTypeSetter(scheme)
	printer.Delegate = &kubernetes.CLIPrinter{
		Format: outputFormat,
	}
	return printer.PrintObj(integration, cmd.OutOrStdout())
}

func (o *runCmdOptions) parseAndConvertToTrait(
	c client.Client, integration *v1.Integration, params []string,
	parse func(string) (*resource.Config, error),
	convert func(*resource.Config) string,
	traitParam string) error {
	for _, param := range params {
		config, err := parse(param)
		if err != nil {
			return err
		}
		// We try to autogenerate a configmap
		_, err = parseConfigAndGenCm(o.Context, c, config, integration, o.Compression)
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

func convertToTraitParameter(value, traitParameter string) ([]string, error) {
	traits := make([]string, 0)
	props, err := extractProperties(value)
	if err != nil {
		return nil, err
	}
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

func (o *runCmdOptions) GetIntegrationName(sources []string) string {
	name := ""
	if o.IntegrationName != "" {
		name = o.IntegrationName
		name = kubernetes.SanitizeName(name)
	} else if len(sources) == 1 {
		name = kubernetes.SanitizeName(sources[0])
	}
	return name
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

func resolvePodTemplate(ctx context.Context, templateSrc string, spec *v1.IntegrationSpec) (err error) {
	// check if template is set
	if templateSrc == "" {
		return nil
	}
	var template v1.PodSpec

	// check if value is a path to the file
	if _, err := os.Stat(templateSrc); err == nil {
		rsc, err := ResolveSources(ctx, []string{templateSrc}, false)
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

func getPlatform(c client.Client, context context.Context, ns string) (*v1.IntegrationPlatform, error) {
	list := v1.NewIntegrationPlatformList()
	if err := c.List(context, &list, ctrl.InNamespace(ns)); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("could not retrieve integration platform from namespace %s", ns))
	}
	if len(list.Items) > 1 {
		return nil, fmt.Errorf("expected 1 integration platform in the namespace, found: %d", len(list.Items))
	} else if len(list.Items) == 0 {
		return nil, errors.New("no integration platforms found in the namespace: run \"kamel install\" to install the platform")
	}
	return &list.Items[0], nil
}

func uploadDependency(platform *v1.IntegrationPlatform, item string, name string, cmd *cobra.Command, integration *v1.Integration) error {
	registry := platform.Spec.Build.Registry.Address
	insecure := platform.Spec.Build.Registry.Insecure
	newStdR, newStdW, _ := os.Pipe()
	defer newStdW.Close()

	fileInfo, err := os.Stat(item[6:])
	if err != nil {
		return err
	}
	mapping := item[6:] + ":" + "."

	version := defaults.Version
	artifactId := name + "-" + strings.TrimSuffix(fileInfo.Name(), ".jar")
	artifactPath := "/org/apache/camel/k/external/" + artifactId + "/" + version + "/" + artifactId + "-" + version + ".jar"
	artifactPath = strings.ToLower(artifactPath)
	target := registry + artifactPath + ":" + version
	options := spectrum.Options{
		PullInsecure:  true,
		PushInsecure:  insecure,
		PullConfigDir: "",
		PushConfigDir: "",
		Base:          "",
		Target:        target,
		Stdout:        cmd.OutOrStdout(),
		Stderr:        cmd.OutOrStderr(),
		Recursive:     false,
	}

	go readSpectrumLogs(newStdR)
	_, err = spectrum.Build(options, mapping)
	if err != nil {
		return err
	}
	integration.Spec.AddDependency("mvn:org.apache.camel.k.external:" + artifactId + ":" + version)
	return nil
}

func readSpectrumLogs(newStdOut *os.File) {
	scanner := bufio.NewScanner(newStdOut)

	for scanner.Scan() {
		line := scanner.Text()
		log.Infof(line)
	}
}
