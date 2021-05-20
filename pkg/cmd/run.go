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
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"reflect"
	"regexp"
	"strings"
	"syscall"

	"github.com/magiconair/properties"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/flow"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	k8slog "github.com/apache/camel-k/pkg/util/kubernetes/log"
	"github.com/apache/camel-k/pkg/util/sync"
	"github.com/apache/camel-k/pkg/util/watch"
)

var (
	traitConfigRegexp = regexp.MustCompile(`^([a-z0-9-]+)((?:\.[a-z0-9-]+)+)=(.*)$`)
)

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
	cmd.Flags().StringArrayP("connect", "c", nil, "A ServiceBinding or Provisioned Service that the integration should bind to, specified as [[apigroup/]version:]kind:[namespace/]name")
	cmd.Flags().StringArrayP("dependency", "d", nil, "A dependency that should be included, e.g., \"-d camel-mail\" for a Camel component, or \"-d mvn:org.my:app:1.0\" for a Maven dependency")
	cmd.Flags().BoolP("wait", "w", false, "Wait for the integration to be running")
	cmd.Flags().StringP("kit", "k", "", "The kit used to run the integration")
	cmd.Flags().StringArrayP("property", "p", nil, "Add a camel property")
	cmd.Flags().StringArray("build-property", nil, "Add a build time property")
	cmd.Flags().StringArray("configmap", nil, "Add a ConfigMap")
	cmd.Flags().StringArray("secret", nil, "Add a Secret")
	cmd.Flags().StringArray("maven-repository", nil, "Add a maven repository")
	cmd.Flags().Bool("logs", false, "Print integration logs")
	cmd.Flags().Bool("sync", false, "Synchronize the local source file with the cluster, republishing at each change")
	cmd.Flags().Bool("dev", false, "Enable Dev mode (equivalent to \"-w --logs --sync\")")
	cmd.Flags().Bool("use-flows", true, "Write yaml sources as Flow objects in the integration custom resource")
	cmd.Flags().String("profile", "", "Trait profile used for deployment")
	cmd.Flags().StringArrayP("trait", "t", nil, "Configure a trait. E.g. \"-t service.enabled=false\"")
	cmd.Flags().StringArray("logging-level", nil, "Configure the logging level. e.g. \"--logging-level org.apache.camel=DEBUG\"")
	cmd.Flags().StringP("output", "o", "", "Output format. One of: json|yaml")
	cmd.Flags().Bool("compression", false, "Enable storage of sources and resources as a compressed binary blobs")
	cmd.Flags().StringArray("resource", nil, "Add a resource")
	cmd.Flags().StringArray("open-api", nil, "Add an OpenAPI v2 spec")
	cmd.Flags().StringArrayP("volume", "v", nil, "Mount a volume into the integration container. E.g \"-v pvcname:/container/path\"")
	cmd.Flags().StringArrayP("env", "e", nil, "Set an environment variable in the integration container. E.g \"-e MY_VAR=my-value\"")
	cmd.Flags().StringArray("property-file", nil, "Bind a property file to the integration. E.g. \"--property-file integration.properties\"")
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
	ConfigMaps      []string `mapstructure:"configmaps" yaml:",omitempty"`
	Secrets         []string `mapstructure:"secrets" yaml:",omitempty"`
	Repositories    []string `mapstructure:"maven-repositories" yaml:",omitempty"`
	Traits          []string `mapstructure:"traits" yaml:",omitempty"`
	LoggingLevels   []string `mapstructure:"logging-levels" yaml:",omitempty"`
	Volumes         []string `mapstructure:"volumes" yaml:",omitempty"`
	EnvVars         []string `mapstructure:"envs" yaml:",omitempty"`
	PropertyFiles   []string `mapstructure:"property-files" yaml:",omitempty"`
	Labels          []string `mapstructure:"labels" yaml:",omitempty"`
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

	err := validatePropertyFiles(o.PropertyFiles)
	if err != nil {
		return err
	}

	for _, label := range o.Labels {
		parts := strings.Split(label, "=")
		if len(parts) != 2 {
			return fmt.Errorf(`invalid label specification %s. Expected "<labelkey>=<labelvalue>"`, label)
		}
	}

	return nil
}

func (o *runCmdOptions) run(cmd *cobra.Command, args []string) error {
	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}

	catalog := trait.NewCatalog(o.Context, c)
	tp := catalog.ComputeTraitsProperties()
	for _, t := range o.Traits {
		kv := strings.SplitN(t, "=", 2)

		if !util.StringSliceExists(tp, kv[0]) {
			fmt.Printf("Error: %s is not a valid trait property\n", t)
			return nil
		}
	}

	integration, err := o.createIntegration(c, args, catalog)
	if err != nil {
		return err
	}

	if o.Dev {
		cs := make(chan os.Signal)
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
		go watch.HandleIntegrationEvents(o.Context, integration, func(event *corev1.Event) bool {
			fmt.Fprintln(cmd.OutOrStdout(), event.Message)
			return true
		})
	}
	if o.Wait || o.Dev {
		for {
			integrationPhase, err := o.waitForIntegrationReady(cmd, integration)
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

// nolint:errcheck
func (o *runCmdOptions) waitForIntegrationReady(cmd *cobra.Command, integration *v1.Integration) (*v1.IntegrationPhase, error) {
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

	return watch.HandleIntegrationStateChanges(o.Context, integration, handler)
}

func (o *runCmdOptions) syncIntegration(cmd *cobra.Command, c client.Client, sources []string, catalog *trait.Catalog) error {
	// Let's watch all relevant files when in dev mode
	var files []string
	files = append(files, sources...)
	files = append(files, o.Resources...)
	files = append(files, o.PropertyFiles...)
	files = append(files, o.OpenAPIs...)

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
							_, err := o.updateIntegrationCode(c, sources, catalog)
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

func (o *runCmdOptions) createIntegration(c client.Client, sources []string, catalog *trait.Catalog) (*v1.Integration, error) {
	return o.updateIntegrationCode(c, sources, catalog)
}

// nolint: gocyclo
func (o *runCmdOptions) updateIntegrationCode(c client.Client, sources []string, catalog *trait.Catalog) (*v1.Integration, error) {
	namespace := o.Namespace

	name := o.GetIntegrationName(sources)

	if name == "" {
		return nil, errors.New("unable to determine integration name")
	}

	var integrationKit *corev1.ObjectReference
	if o.IntegrationKit != "" {
		integrationKit = &corev1.ObjectReference{
			Namespace: namespace,
			Name:      o.IntegrationKit,
		}
	}

	integration := v1.Integration{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.IntegrationKind,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: v1.IntegrationSpec{
			Dependencies:   make([]string, 0, len(o.Dependencies)),
			IntegrationKit: integrationKit,
			Configuration:  make([]v1.ConfigurationSpec, 0),
			Repositories:   o.Repositories,
			Profile:        v1.TraitProfileByName(o.Profile),
		},
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

	srcs := make([]string, 0, len(sources)+len(o.Sources))
	srcs = append(srcs, sources...)
	srcs = append(srcs, o.Sources...)

	resolvedSources, err := ResolveSources(context.Background(), srcs, o.Compression)
	if err != nil {
		return nil, err
	}

	for _, source := range resolvedSources {
		if o.UseFlows && !o.Compression && (strings.HasSuffix(source.Name, ".yaml") || strings.HasSuffix(source.Name, ".yml")) {
			flows, err := flow.FromYamlDSLString(source.Content)
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

	for _, resource := range o.Resources {
		rawData, contentType, err := loadRawContent(resource)
		if err != nil {
			return nil, err
		}

		resourceSpec, err := binaryOrTextResource(path.Base(resource), rawData, contentType, o.Compression)
		if err != nil {
			return nil, err
		}
		integration.Spec.AddResources(resourceSpec)
	}

	for _, resource := range o.OpenAPIs {
		data, _, compressed, err := loadTextContent(resource, o.Compression)
		if err != nil {
			return nil, err
		}

		integration.Spec.AddResources(v1.ResourceSpec{
			DataSpec: v1.DataSpec{
				Name:        path.Base(resource),
				Content:     data,
				Compression: compressed,
			},
			Type: v1.ResourceTypeOpenAPI,
		})
	}

	for _, item := range o.Dependencies {
		integration.Spec.AddDependency(item)
	}
	for _, pf := range o.PropertyFiles {
		if err := addPropertyFile(pf, &integration.Spec); err != nil {
			return nil, err
		}
	}
	for _, item := range o.Properties {
		integration.Spec.AddConfiguration("property", item)
	}
	// convert each build configuration to a builder trait property
	for _, item := range o.BuildProperties {
		o.Traits = append(o.Traits, fmt.Sprintf("builder.properties=%s", item))
	}
	for _, item := range o.LoggingLevels {
		integration.Spec.AddConfiguration("property", "logging.level."+item)
	}
	for _, item := range o.ConfigMaps {
		integration.Spec.AddConfiguration("configmap", item)
	}
	for _, item := range o.Secrets {
		integration.Spec.AddConfiguration("secret", item)
	}
	for _, item := range o.Volumes {
		integration.Spec.AddConfiguration("volume", item)
	}
	for _, item := range o.EnvVars {
		integration.Spec.AddConfiguration("env", item)
	}

	if err := o.configureTraits(&integration, o.Traits, catalog); err != nil {
		return nil, err
	}

	switch o.OutputFormat {
	case "":
		// continue..
	case "yaml":
		data, err := kubernetes.ToYAML(&integration)
		if err != nil {
			return nil, err
		}
		fmt.Print(string(data))
		return nil, nil

	case "json":
		data, err := kubernetes.ToJSON(&integration)
		if err != nil {
			return nil, err
		}
		fmt.Print(string(data))
		return nil, nil

	default:
		return nil, fmt.Errorf("invalid output format option '%s', should be one of: yaml|json", o.OutputFormat)
	}

	existed := false
	err = c.Create(o.Context, &integration)
	if err != nil && k8serrors.IsAlreadyExists(err) {
		existed = true
		existing := &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: integration.Namespace,
				Name:      integration.Name,
			},
		}
		err = c.Get(o.Context, ctrl.ObjectKeyFromObject(existing), existing)
		if err != nil {
			return nil, err
		}
		// Hold the resource from the operator controller
		existing.Status.Phase = v1.IntegrationPhaseUpdating
		err = c.Status().Update(o.Context, existing)
		if err != nil {
			return nil, err
		}
		// Update the spec
		integration.ResourceVersion = existing.ResourceVersion
		err = c.Update(o.Context, &integration)
		if err != nil {
			return nil, err
		}
		// Reset the status
		integration.Status = v1.IntegrationStatus{}
		err = c.Status().Update(o.Context, &integration)
	}

	if err != nil {
		return nil, err
	}

	if !existed {
		fmt.Printf("integration \"%s\" created\n", name)
	} else {
		fmt.Printf("integration \"%s\" updated\n", name)
	}
	return &integration, nil
}

func binaryOrTextResource(fileName string, data []byte, contentType string, base64Compression bool) (v1.ResourceSpec, error) {
	resourceSpec := v1.ResourceSpec{
		DataSpec: v1.DataSpec{
			Name:        fileName,
			ContentKey:  fileName,
			ContentType: contentType,
			Compression: false,
		},
		Type: v1.ResourceTypeData,
	}

	if !base64Compression && isBinary(contentType) {
		resourceSpec.RawContent = data
		return resourceSpec, nil
	}
	// either is a text resource or base64 compression is enabled
	if base64Compression {
		content, err := compressToString(data)
		if err != nil {
			return resourceSpec, err
		}
		resourceSpec.Content = content
		resourceSpec.Compression = true
	} else {
		resourceSpec.Content = string(data)
	}
	return resourceSpec, nil
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

func (o *runCmdOptions) configureTraits(integration *v1.Integration, options []string, catalog *trait.Catalog) error {
	// configure ServiceBinding trait
	for _, sb := range o.Connects {
		bindings := fmt.Sprintf("service-binding.service-bindings=%s", sb)
		options = append(options, bindings)
	}
	traits, err := configureTraits(options, catalog)
	if err != nil {
		return err
	}

	integration.Spec.Traits = traits

	return nil
}

func isLocalAndFileExists(fileName string) (bool, error) {
	info, err := os.Stat(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		// If it is a different error (ie, permission denied) we should report it back
		return false, errors.Wrap(err, fmt.Sprintf("file system error while looking for %s", fileName))
	}
	return !info.IsDir(), nil
}

func addPropertyFile(fileName string, spec *v1.IntegrationSpec) error {
	props, err := loadPropertyFile(fileName)
	if err != nil {
		return err
	}
	for _, k := range props.Keys() {
		v, _ := props.Get(k)
		spec.AddConfiguration(
			"property",
			escapePropertyFileItem(k)+"="+escapePropertyFileItem(v),
		)
	}
	return nil
}

func loadPropertyFile(fileName string) (*properties.Properties, error) {
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	p, err := properties.Load(file, properties.UTF8)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func escapePropertyFileItem(item string) string {
	item = strings.ReplaceAll(item, `=`, `\=`)
	item = strings.ReplaceAll(item, `:`, `\:`)
	return item
}

func resolvePodTemplate(ctx context.Context, templateSrc string, spec *v1.IntegrationSpec) (err error) {
	// check if template is set
	if templateSrc == "" {
		return nil
	}
	var template v1.PodSpec

	//check if value is a path to the file
	if _, err := os.Stat(templateSrc); err == nil {
		rsc, err := ResolveSources(ctx, []string{templateSrc}, false)
		if err == nil && len(rsc) > 0 {
			templateSrc = rsc[0].Content
		}
	}
	//template is inline
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

func configureTraits(options []string, catalog *trait.Catalog) (map[string]v1.TraitSpec, error) {
	traits := make(map[string]map[string]interface{})

	for _, option := range options {
		parts := traitConfigRegexp.FindStringSubmatch(option)
		if len(parts) < 4 {
			return nil, errors.New("unrecognized config format (expected \"<trait>.<prop>=<value>\"): " + option)
		}
		id := parts[1]
		prop := parts[2][1:]
		value := parts[3]
		if _, ok := traits[id]; !ok {
			traits[id] = make(map[string]interface{})
		}
		switch v := traits[id][prop].(type) {
		case []string:
			traits[id][prop] = append(v, value)
		case string:
			// Aggregate multiple occurrences of the same option into a string array, to emulate POSIX conventions.
			// This enables executing:
			// $ kamel run -t <trait>.<property>=<value_1> ... -t <trait>.<property>=<value_N>
			// Or:
			// $ kamel run --trait <trait>.<property>=<value_1>,...,<trait>.<property>=<value_N>
			traits[id][prop] = []string{v, value}
		case nil:
			traits[id][prop] = value
		}
	}

	specs := make(map[string]v1.TraitSpec)
	for id, config := range traits {
		t := catalog.GetTrait(id)
		if t != nil {
			// let's take a clone to prevent default values set at runtime from being serialized
			zero := reflect.New(reflect.TypeOf(t)).Interface()
			err := configureTrait(config, zero)
			if err != nil {
				return nil, err
			}
			data, err := json.Marshal(zero)
			if err != nil {
				return nil, err
			}
			var spec v1.TraitSpec
			err = json.Unmarshal(data, &spec.Configuration)
			if err != nil {
				return nil, err
			}
			specs[id] = spec
		}
	}

	return specs, nil
}

func configureTrait(config map[string]interface{}, trait interface{}) error {
	md := mapstructure.Metadata{}

	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			Metadata:         &md,
			WeaklyTypedInput: true,
			TagName:          "property",
			Result:           &trait,
		},
	)

	if err != nil {
		return err
	}

	return decoder.Decode(config)
}
