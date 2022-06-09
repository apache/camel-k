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
	"archive/zip"
	"context"

	// this is needed to generate an SHA1 sum for Jars
	// #nosec G501
	"crypto/md5"
	// #nosec G505
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"io/ioutil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"regexp"
	runtimeos "runtime"
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
	platformutil "github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/dsl"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	k8slog "github.com/apache/camel-k/pkg/util/kubernetes/log"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/apache/camel-k/pkg/util/property"
	"github.com/apache/camel-k/pkg/util/resource"
	"github.com/apache/camel-k/pkg/util/sync"
	"github.com/apache/camel-k/pkg/util/watch"
)

func newCmdRun(rootCmdOptions *RootCmdOptions) (*cobra.Command, *runCmdOptions) {
	options := runCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:               "run [file to run]",
		Short:             "Run a integration on Kubernetes",
		Long:              `Deploys and execute a integration pod on Kubernetes.`,
		Args:              options.validateArgs,
		PersistentPreRunE: options.decode,
		PreRunE:           options.preRunE,
		RunE:              options.run,
		PostRunE:          options.postRun,
		Annotations:       make(map[string]string),
	}

	cmd.Flags().String("name", "", "The integration name")
	cmd.Flags().StringArrayP("connect", "c", nil, "A Service that the integration should bind to, specified as [[apigroup/]version:]kind:[namespace/]name")
	cmd.Flags().StringArrayP("dependency", "d", nil, "A dependency that should be included, e.g., \"-d camel-mail\" for a Camel component, \"-d mvn:org.my:app:1.0\" for a Maven dependency or \"file://localPath[?targetPath=<path>&registry=<registry URL>&skipChecksums=<true>&skipPOM=<true>]\" for local files (experimental)")
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
	cmd.Flags().String("operator-id", "camel-k", "Operator id selected to manage this integration.")
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
	cmd.Flags().Bool("force", false, "Force creation of integration regardless of potential misconfiguration.")

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
	OperatorID      string   `mapstructure:"operator-id" yaml:",omitempty"`
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
	RegistryOptions url.Values
	Force           bool `mapstructure:"force" yaml:",omitempty"`
}

func (o *runCmdOptions) preRunE(cmd *cobra.Command, args []string) error {
	if o.OutputFormat != "" {
		// let the command work in offline mode
		cmd.Annotations[offlineCommandLabel] = "true"
	}
	return o.RootCmdOptions.preRun(cmd, args)
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

func (o *runCmdOptions) validateArgs(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("run expects at least 1 argument, received 0")
	}

	if _, err := ResolveSources(context.Background(), args, false, cmd); err != nil {
		return errors.Wrap(err, "One of the provided sources is not reachable")
	}

	return nil
}

func (o *runCmdOptions) validate() error {
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
		err = k8slog.Print(o.Context, cmd, c, integration, cmd.OutOrStdout())
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

func (o *runCmdOptions) syncIntegration(cmd *cobra.Command, c client.Client, sources []string) error {
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

// nolint: gocyclo
func (o *runCmdOptions) createOrUpdateIntegration(cmd *cobra.Command, c client.Client, sources []string) (*v1.Integration, error) {
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
	if !isOfflineCommand(cmd) {
		err := c.Get(o.Context, ctrl.ObjectKeyFromObject(integration), existing)
		switch {
		case err == nil:
			integration = existing.DeepCopy()
		case k8serrors.IsNotFound(err):
			existing = nil
		default:
			return nil, err
		}
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

	if integration.Annotations == nil {
		integration.Annotations = make(map[string]string)
	}

	if o.OperatorID != "" {
		if pl, err := platformutil.LookupForPlatformName(o.Context, c, o.OperatorID); err != nil {
			if k8serrors.IsForbidden(err) {
				o.PrintfVerboseOutf(cmd, "Unable to verify existence of operator id [%s] due to lack of user privileges\n", o.OperatorID)
			} else {
				return nil, err
			}
		} else if pl == nil {
			if o.Force {
				o.PrintfVerboseOutf(cmd, "Unable to find operator with given id [%s] - integration may not be reconciled and get stuck in waiting state\n", o.OperatorID)
			} else {
				return nil, fmt.Errorf("unable to find integration platform for given operator id '%s', use --force option or make sure to use a proper operator id", o.OperatorID)
			}
		}
	}

	// --operator-id={id} is a syntax sugar for '--annotation camel.apache.org/operator.id={id}'
	integration.SetOperatorID(strings.TrimSpace(o.OperatorID))

	for _, annotation := range o.Annotations {
		parts := strings.SplitN(annotation, "=", 2)
		if len(parts) == 2 {
			integration.Annotations[parts[0]] = parts[1]
		}
	}

	srcs := make([]string, 0, len(sources)+len(o.Sources))
	srcs = append(srcs, sources...)
	srcs = append(srcs, o.Sources...)

	resolvedSources, err := ResolveSources(context.Background(), srcs, o.Compression, cmd)
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

	err = resolvePodTemplate(context.Background(), cmd, o.PodTemplate, &integration.Spec)
	if err != nil {
		return nil, err
	}

	err = o.parseAndConvertToTrait(cmd, c, integration, o.Resources, resource.ParseResource, func(c *resource.Config) string { return c.String() }, "mount.resources")
	if err != nil {
		return nil, err
	}
	err = o.parseAndConvertToTrait(cmd, c, integration, o.Configs, resource.ParseConfig, func(c *resource.Config) string { return c.String() }, "mount.configs")
	if err != nil {
		return nil, err
	}
	err = o.parseAndConvertToTrait(cmd, c, integration, o.OpenAPIs, resource.ParseConfig, func(c *resource.Config) string { return c.Name() }, "openapi.configmaps")
	if err != nil {
		return nil, err
	}

	var platform *v1.IntegrationPlatform
	for _, item := range o.Dependencies {
		// TODO: accept URLs
		if strings.HasPrefix(item, "file://") {
			if platform == nil {
				// let's also enable the registry trait if not explicitly disabled
				if !contains(o.Traits, "registry.enabled=false") {
					o.Traits = append(o.Traits, "registry.enabled=true")
				}
				platform, err = platformutil.GetOrFindForResource(o.Context, c, integration, true)
				if err != nil {
					return nil, err
				}
				ca := platform.Status.Build.Registry.CA
				if ca != "" {
					o.PrintfVerboseOutf(cmd, "We've noticed the image registry is configured with a custom certificate [%s] \n", ca)
					o.PrintVerboseOut(cmd, "Please make sure Kamel CLI is configured to use it or the operation will fail.")
					o.PrintVerboseOut(cmd, "More information can be found here https://nodejs.org/api/cli.html#cli_node_extra_ca_certs_file")
				}
				secret := platform.Status.Build.Registry.Secret
				if secret != "" {
					o.PrintfVerboseOutf(cmd, "We've noticed the image registry is configured with a Secret [%s] \n", secret)
					o.PrintVerboseOut(cmd, "Please configure Docker authentication correctly or the operation will fail (by default it's $HOME/.docker/config.json).")
					o.PrintVerboseOut(cmd, "More information can be found here https://docs.docker.com/engine/reference/commandline/login/")
				}
			}
			if err := o.uploadFileOrDirectory(platform, item, name, cmd, integration); err != nil {
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
		catalog := trait.NewCatalog(c)
		if err := configureTraits(o.Traits, &integration.Spec.Traits, catalog); err != nil {
			return nil, err
		}
	}

	if o.OutputFormat != "" {
		return nil, showIntegrationOutput(cmd, integration, o.OutputFormat, c.GetScheme())
	}

	if existing == nil {
		err = c.Create(o.Context, integration)
		fmt.Fprintln(cmd.OutOrStdout(), `Integration "`+name+`" created`)
	} else {
		err = c.Patch(o.Context, integration, ctrl.MergeFromWithOptions(existing, ctrl.MergeFromWithOptimisticLock{}))
		fmt.Fprintln(cmd.OutOrStdout(), `Integration "`+name+`" updated`)
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

func (o *runCmdOptions) parseAndConvertToTrait(cmd *cobra.Command,
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
		_, err = parseConfigAndGenCm(o.Context, cmd, c, config, integration, o.Compression)
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

func resolvePodTemplate(ctx context.Context, cmd *cobra.Command, templateSrc string, spec *v1.IntegrationSpec) (err error) {
	// check if template is set
	if templateSrc == "" {
		return nil
	}
	var template v1.PodSpec

	// check if value is a path to the file
	if _, err := os.Stat(templateSrc); err == nil {
		rsc, err := ResolveSources(ctx, []string{templateSrc}, false, cmd)
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

func parseFileURI(uri string) *url.URL {
	file := new(url.URL)
	file.Scheme = "file"
	path := strings.TrimPrefix(uri, "file://")
	i := strings.IndexByte(path, '?')
	if i > 0 {
		file.Path = path[:i]
		file.RawQuery = path[i+1:]
	} else {
		file.Path = path
	}
	return file
}

func (o *runCmdOptions) getRegistry(platform *v1.IntegrationPlatform) string {
	registry := o.RegistryOptions.Get("registry")
	if registry != "" {
		return registry
	}
	return platform.Status.Build.Registry.Address
}

func (o *runCmdOptions) skipChecksums() bool {
	return o.RegistryOptions.Get("skipChecksums") == "true"
}

func (o *runCmdOptions) skipPom() bool {
	return o.RegistryOptions.Get("skipPOM") == "true"
}

func (o *runCmdOptions) getTargetPath() string {
	return o.RegistryOptions.Get("targetPath")
}

func (o *runCmdOptions) uploadFileOrDirectory(platform *v1.IntegrationPlatform, item string, integrationName string, cmd *cobra.Command, integration *v1.Integration) error {
	uri := parseFileURI(item)
	o.RegistryOptions = uri.Query()
	localPath, targetPath := uri.Path, o.getTargetPath()
	options := o.getSpectrumOptions(platform, cmd)
	dirName, err := getDirName(localPath)
	if err != nil {
		return err
	}

	return filepath.WalkDir(localPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Let's try to build a default Maven GAV from the path
		gav, err := createDefaultGav(path, dirName, integrationName)
		if err != nil {
			return err
		}
		// When uploading, there are three cases: POM files, JAR files and the rest which will be mounted on the filesystem
		switch {
		case isPom(path):
			gav := extractGavFromPom(path, gav)
			return o.uploadAsMavenArtifact(gav, path, platform, integration.Namespace, options, cmd)
		case isJar(path):
			// Try to upload pom in JAR and extract it's GAV
			gav = o.uploadPomFromJar(gav, path, platform, integration.Namespace, options, cmd)
			// add JAR to dependency list
			dependency := fmt.Sprintf("mvn:%s:%s:%s:%s", gav.GroupID, gav.ArtifactID, gav.Type, gav.Version)
			o.PrintfVerboseOutf(cmd, "Added %s to the Integration's dependency list \n", dependency)
			integration.Spec.AddDependency(dependency)
			// Upload JAR
			return o.uploadAsMavenArtifact(gav, path, platform, integration.Namespace, options, cmd)
		default:
			mountPath, err := getMountPath(targetPath, dirName, path)
			if err != nil {
				return err
			}
			dependency := fmt.Sprintf("registry-mvn:%s:%s:%s:%s@%s", gav.GroupID, gav.ArtifactID, gav.Type, gav.Version, mountPath)
			o.PrintfVerboseOutf(cmd, "Added %s to the Integration's dependency list \n", dependency)
			integration.Spec.AddDependency(dependency)
			return o.uploadAsMavenArtifact(gav, path, platform, integration.Namespace, options, cmd)
		}
	})
}

func getMountPath(targetPath string, dirName string, path string) (string, error) {
	// if the target path is a file then use that as the exact mount path
	if filepath.Ext(targetPath) != "" {
		return targetPath, nil
	}
	// else build a mount path based on the filename relative to the base directory
	// (in case we are uploading multiple files with the same name)
	localRelativePath, err := filepath.Rel(dirName, path)
	if err != nil {
		return "", err
	}
	return filepath.Join(targetPath, localRelativePath), nil
}

// nolint:errcheck
func (o *runCmdOptions) uploadPomFromJar(gav maven.Dependency, path string, platform *v1.IntegrationPlatform, ns string, options spectrum.Options, cmd *cobra.Command) maven.Dependency {
	util.WithTempDir("camel-k", func(tmpDir string) error {
		pomPath := filepath.Join(tmpDir, "pom.xml")
		jar, err := zip.OpenReader(path)
		if err != nil {
			return err
		}
		defer jar.Close()
		regPom := regexp.MustCompile(`META-INF/maven/.*/.*/pom\.xml`)
		regPomProperties := regexp.MustCompile(`META-INF/maven/.*/.*/pom\.properties`)
		foundPom := false
		foundProperties := false
		pomExtracted := false
		for _, f := range jar.File {
			if regPom.MatchString(f.Name) {
				foundPom = true
				pomExtracted = extractFromZip(pomPath, f)
			} else if regPomProperties.MatchString(f.Name) {
				foundProperties = true
				if dep, ok := o.extractGav(f, path, cmd); ok {
					gav = dep
				}
			}
			if foundPom && foundProperties {
				break
			}
		}
		if pomExtracted {
			if o.skipPom() {
				o.PrintfVerboseOutf(cmd, "Skipping uploading extracted POM from %s \n", path)
			} else {
				gav.Type = "pom"
				// Swallow error as this is not a mandatory step
				o.uploadAsMavenArtifact(gav, pomPath, platform, ns, options, cmd)
			}
		}
		return nil
	})
	gav.Type = "jar"
	return gav
}

func extractFromZip(dst string, src *zip.File) bool {
	file, err := os.Create(dst)
	if err != nil {
		return false
	}
	defer file.Close()
	rc, err := src.Open()
	if err != nil {
		return false
	}
	defer rc.Close()
	// no DoS on client side
	// #nosec G110
	_, err = io.Copy(file, rc)
	return err == nil
}

func (o *runCmdOptions) extractGav(src *zip.File, localPath string, cmd *cobra.Command) (maven.Dependency, bool) {
	rc, err := src.Open()
	if err != nil {
		return maven.Dependency{}, false
	}
	defer rc.Close()
	data, err := ioutil.ReadAll(rc)
	if err != nil {
		o.PrintfVerboseErrf(cmd, "Error while reading pom.properties from [%s], switching to default: \n %s err \n", localPath, err)
		return maven.Dependency{}, false
	}
	prop, err := properties.Load(data, properties.UTF8)
	if err != nil {
		o.PrintfVerboseErrf(cmd, "Error while reading pom.properties from [%s], switching to default: \n %s err \n", localPath, err)
		return maven.Dependency{}, false
	}

	groupID, ok := prop.Get("groupId")
	if !ok {
		o.PrintfVerboseErrf(cmd, "Couldn't find groupId property while reading pom.properties from [%s], switching to default \n", localPath)
		return maven.Dependency{}, false
	}
	artifactID, ok := prop.Get("artifactId")
	if !ok {
		o.PrintfVerboseErrf(cmd, "Couldn't find artifactId property while reading pom.properties from [%s], switching to default \n", localPath)
		return maven.Dependency{}, false
	}
	version, ok := prop.Get("version")
	if !ok {
		o.PrintfVerboseErrf(cmd, "Couldn't find version property while reading pom.properties from [%s], switching to default \n", localPath)
		return maven.Dependency{}, false
	}
	return maven.Dependency{
		GroupID:    groupID,
		ArtifactID: artifactID,
		Type:       "jar",
		Version:    version,
	}, true
}

func (o *runCmdOptions) uploadAsMavenArtifact(dependency maven.Dependency, path string, platform *v1.IntegrationPlatform, ns string, options spectrum.Options, cmd *cobra.Command) error {
	artifactHTTPPath := getArtifactHTTPPath(dependency, platform, ns)
	options.Target = fmt.Sprintf("%s/%s:%s", o.getRegistry(platform), artifactHTTPPath, dependency.Version)
	if runtimeos.GOOS == "windows" {
		// workaround for https://github.com/container-tools/spectrum/issues/8
		// work with relative paths instead
		rel, err := getRelativeToWorkingDirectory(path)
		if err != nil {
			return err
		}
		path = rel
	}
	_, err := spectrum.Build(options, fmt.Sprintf("%s:.", path))
	if err != nil {
		return err
	}
	o.PrintfVerboseOutf(cmd, "Uploaded: %s to %s \n", path, options.Target)
	if o.skipChecksums() {
		o.PrintfVerboseOutf(cmd, "Skipping generating and uploading checksum files for %s \n", path)
		return nil
	}
	return o.uploadChecksumFiles(path, options, platform, artifactHTTPPath, dependency)
}

// Deprecated: workaround for https://github.com/container-tools/spectrum/issues/8
func getRelativeToWorkingDirectory(path string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	path, err = filepath.Rel(wd, abs)
	if err != nil {
		return "", err
	}
	return path, nil
}

// Currently swallows errors because our Project model is incomplete.
// Most of the time it is irrelevant for our use case (GAV).
// nolint:errcheck
func extractGavFromPom(path string, gav maven.Dependency) maven.Dependency {
	var project maven.Project
	file, err := os.Open(path)
	if err != nil {
		return gav
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return gav
	}
	xml.Unmarshal(content, &project)
	if project.GroupID != "" {
		gav.GroupID = project.GroupID
	}
	if project.ArtifactID != "" {
		gav.ArtifactID = project.ArtifactID
	}
	if project.Version != "" {
		gav.Version = project.Version
	}
	gav.Type = "pom"
	return gav
}

func (o *runCmdOptions) uploadChecksumFiles(path string, options spectrum.Options, platform *v1.IntegrationPlatform, artifactHTTPPath string, dependency maven.Dependency) error {
	return util.WithTempDir("camel-k", func(tmpDir string) error {
		// #nosec G401
		if err := o.uploadChecksumFile(md5.New(), tmpDir, "_md5", path, options, platform, artifactHTTPPath, dependency); err != nil {
			return err
		}
		// #nosec G401
		return o.uploadChecksumFile(sha1.New(), tmpDir, "_sha1", path, options, platform, artifactHTTPPath, dependency)
	})
}

func (o *runCmdOptions) uploadChecksumFile(hash hash.Hash, tmpDir string, ext string, path string, options spectrum.Options, platform *v1.IntegrationPlatform, artifactHTTPPath string, dependency maven.Dependency) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(hash, file)
	if err != nil {
		return err
	}

	filename := "maven_" + filepath.Base(path) + ext
	filepath := filepath.Join(tmpDir, filename)
	if runtimeos.GOOS == "windows" {
		// workaround for https://github.com/container-tools/spectrum/issues/8
		// work with relative paths instead
		rel, err := getRelativeToWorkingDirectory(path)
		if err != nil {
			return err
		}
		filepath = rel
	}

	if err = writeChecksumToFile(filepath, hash); err != nil {
		return err
	}
	options.Target = fmt.Sprintf("%s/%s%s:%s", o.getRegistry(platform), artifactHTTPPath, ext, dependency.Version)
	_, err = spectrum.Build(options, fmt.Sprintf("%s:.", filepath))
	return err
}

func writeChecksumToFile(filepath string, hash hash.Hash) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(hex.EncodeToString(hash.Sum(nil)))
	return err
}

func (o *runCmdOptions) getSpectrumOptions(platform *v1.IntegrationPlatform, cmd *cobra.Command) spectrum.Options {
	insecure := platform.Status.Build.Registry.Insecure
	var stdout io.Writer
	if o.Verbose {
		stdout = cmd.OutOrStdout()
	}
	options := spectrum.Options{
		PullInsecure:  true,
		PushInsecure:  insecure,
		PullConfigDir: "",
		PushConfigDir: "",
		Base:          "",
		Stdout:        stdout,
		Stderr:        cmd.OutOrStderr(),
		Recursive:     false,
	}
	return options
}

func getArtifactHTTPPath(dependency maven.Dependency, platform *v1.IntegrationPlatform, ns string) string {
	artifactHTTPPath := fmt.Sprintf("maven_%s_%s_%s_%s-%s_%s", dependency.GroupID, dependency.ArtifactID, dependency.Version, dependency.ArtifactID, dependency.Version, dependency.Type)
	// Image repository names must be lower cased
	artifactHTTPPath = strings.ToLower(artifactHTTPPath)
	// Some vendors don't allow '/' or '.' in repository name so let's replace them with '_'
	artifactHTTPPath = strings.ReplaceAll(artifactHTTPPath, "/", "_")
	artifactHTTPPath = strings.ReplaceAll(artifactHTTPPath, ".", "_")
	organization := platform.Status.Build.Registry.Organization
	if organization == "" {
		organization = ns
	}
	return fmt.Sprintf("%s/%s", organization, artifactHTTPPath)
}

func createDefaultGav(path string, dirName string, integrationName string) (maven.Dependency, error) {
	// let's set the default ArtifactId using the integration name and the file's relative path
	// we use the relative path in case of nested files that might have the same name
	// we replace the file seperators with dots to comply with Maven GAV naming conventions.
	fileRelPath, ext, err := getFileRelativePathAndExtension(path, dirName)
	if err != nil {
		return maven.Dependency{}, err
	}

	defaultArtifactID := integrationName + "-" + strings.ReplaceAll(fileRelPath, string(os.PathSeparator), ".")
	defaultGroupID := "org.apache.camel.k.external"
	defaultVersion := defaults.Version

	return maven.Dependency{
		GroupID:    defaultGroupID,
		ArtifactID: defaultArtifactID,
		Type:       ext,
		Version:    defaultVersion,
	}, nil
}

func isPom(path string) bool {
	return strings.HasSuffix(path, ".pom") || strings.HasSuffix(path, "pom.xml")
}
func isJar(path string) bool {
	return strings.HasSuffix(path, ".jar")
}

func getFileRelativePathAndExtension(path string, dirName string) (string, string, error) {
	extension := filepath.Ext(path)
	name, err := filepath.Rel(dirName, path)
	if err != nil {
		return "", "", err
	}
	return name[0 : len(name)-len(extension)], extension[1:], nil
}

func getDirName(path string) (string, error) {
	parentDir := path
	fileInfo, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !fileInfo.IsDir() {
		parentDir = filepath.Dir(parentDir)
	}
	return parentDir, nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
