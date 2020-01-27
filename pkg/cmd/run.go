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
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/gzip"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	k8slog "github.com/apache/camel-k/pkg/util/kubernetes/log"
	"github.com/apache/camel-k/pkg/util/sync"
	"github.com/apache/camel-k/pkg/util/watch"
	"github.com/magiconair/properties"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	traitConfigRegexp = regexp.MustCompile(`^([a-z-]+)((?:\.[a-z-]+)+)=(.*)$`)
)

func newCmdRun(rootCmdOptions *RootCmdOptions) (*cobra.Command, *runCmdOptions) {
	options := runCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "run [file to run]",
		Short:   "Run a integration on Kubernetes",
		Long:    `Deploys and execute a integration pod on Kubernetes.`,
		Args:    options.validateArgs,
		PreRunE: options.decode,
		RunE:    options.run,
	}

	cmd.Flags().String("name", "", "The integration name")
	cmd.Flags().StringArrayP("dependency", "d", nil, "The integration dependency")
	cmd.Flags().BoolP("wait", "w", false, "Waits for the integration to be running")
	cmd.Flags().StringP("kit", "k", "", "The kit used to run the integration")
	cmd.Flags().StringArrayP("property", "p", nil, "Add a camel property")
	cmd.Flags().StringArray("configmap", nil, "Add a ConfigMap")
	cmd.Flags().StringArray("secret", nil, "Add a Secret")
	cmd.Flags().StringArray("maven-repository", nil, "Add a maven repository")
	cmd.Flags().Bool("logs", false, "Print integration logs")
	cmd.Flags().Bool("sync", false, "Synchronize the local source file with the cluster, republishing at each change")
	cmd.Flags().Bool("dev", false, "Enable Dev mode (equivalent to \"-w --logs --sync\")")
	cmd.Flags().String("profile", "", "Trait profile used for deployment")
	cmd.Flags().StringArrayP("trait", "t", nil, "Configure a trait. E.g. \"-t service.enabled=false\"")
	cmd.Flags().StringArray("logging-level", nil, "Configure the logging level. e.g. \"--logging-level org.apache.camel=DEBUG\"")
	cmd.Flags().StringP("output", "o", "", "Output format. One of: json|yaml")
	cmd.Flags().Bool("compression", false, "Enable store source as a compressed binary blob")
	cmd.Flags().StringArray("resource", nil, "Add a resource")
	cmd.Flags().StringArray("open-api", nil, "Add an OpenAPI v2 spec")
	cmd.Flags().StringArrayP("volume", "v", nil, "Mount a volume into the integration container. E.g \"-v pvcname:/container/path\"")
	cmd.Flags().StringArrayP("env", "e", nil, "Set an environment variable in the integration container. E.g \"-e MY_VAR=my-value\"")
	cmd.Flags().StringArrayP("property-file", "", nil, "Bind a property file to the integration. E.g. \"--property-file integration.properties\"")
	cmd.Flags().StringArrayP("label", "", nil, "Add a label to the integration. E.g. \"--label my.company=hello\"")

	cmd.Flags().Bool("save", false, "Save the run parameters into the default kamel configuration file (kamel-config.yaml)")

	// completion support
	configureKnownCompletions(&cmd)

	return &cmd, &options
}

type runCmdOptions struct {
	*RootCmdOptions
	Compression     bool     `mapstructure:"compression"`
	Wait            bool     `mapstructure:"wait"`
	Logs            bool     `mapstructure:"logs"`
	Sync            bool     `mapstructure:"sync"`
	Dev             bool     `mapstructure:"dev"`
	Save            bool     `mapstructure:"save"`
	IntegrationKit  string   `mapstructure:"kit"`
	IntegrationName string   `mapstructure:"name"`
	Profile         string   `mapstructure:"profile"`
	OutputFormat    string   `mapstructure:"output"`
	Resources       []string `mapstructure:"resources"`
	OpenAPIs        []string `mapstructure:"open-apis"`
	Dependencies    []string `mapstructure:"dependencies"`
	Properties      []string `mapstructure:"properties"`
	ConfigMaps      []string `mapstructure:"configmaps"`
	Secrets         []string `mapstructure:"secrets"`
	Repositories    []string `mapstructure:"maven-repositories"`
	Traits          []string `mapstructure:"traits"`
	LoggingLevels   []string `mapstructure:"logging-levels"`
	Volumes         []string `mapstructure:"volumes"`
	EnvVars         []string `mapstructure:"envs"`
	PropertyFiles   []string `mapstructure:"property-files"`
	Labels          []string `mapstructure:"labels"`
}

func (o *runCmdOptions) decode(cmd *cobra.Command, args []string) error {
	pathToRoot := pathToRoot(cmd)
	if err := decodeKey(o, pathToRoot); err != nil {
		return err
	}

	name := o.GetIntegrationName(args)
	if name != "" {
		secondaryPath := pathToRoot + ".integration." + name
		if err := decodeKey(o, secondaryPath); err != nil {
			return err
		}
	}

	return nil
}

func (o *runCmdOptions) validateArgs(_ *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("run expects at least 1 argument, received 0")
	}
	if len(args) > 1 && o.IntegrationName == "" {
		return errors.New("integration name is mandatory when using multiple sources")
	}

	for _, fileName := range args {
		if !isRemoteHTTPFile(fileName) {
			if _, err := os.Stat(fileName); err != nil && os.IsNotExist(err) {
				return errors.Wrap(err, "file "+fileName+" does not exist")
			} else if err != nil {
				return errors.Wrap(err, "error while accessing file "+fileName)
			}
		} else {
			// nolint: gosec
			resp, err := http.Get(fileName)
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}

			if err != nil {
				return errors.Wrap(err, "The URL provided is not reachable")
			} else if resp.StatusCode != 200 {
				return errors.New("The URL provided is not reachable " + fileName + " The error code returned is " + strconv.Itoa(resp.StatusCode))
			}
		}
	}

	for _, volume := range o.Volumes {
		volumeConfig := strings.Split(volume, ":")
		if len(volumeConfig) != 2 || len(strings.TrimSpace(volumeConfig[0])) == 0 || len(strings.TrimSpace(volumeConfig[1])) == 0 {
			return fmt.Errorf("volume '%s' is invalid, it should be in the format: pvcname:/container/path", volume)
		}
	}

	for _, fileName := range o.PropertyFiles {
		if !strings.HasSuffix(fileName, ".properties") {
			return fmt.Errorf("supported property files must have a .properties extension: %s", fileName)
		}

		if file, err := os.Stat(fileName); err != nil {
			return errors.Wrapf(err, "unable to access property file %s", fileName)
		} else if file.IsDir() {
			return fmt.Errorf("property file %s is a directory", fileName)
		}
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

	integration, err := o.createIntegration(c, args)
	if err != nil {
		return err
	}

	if o.Dev {
		cs := make(chan os.Signal)
		signal.Notify(cs, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-cs
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
		err = o.syncIntegration(c, args)
		if err != nil {
			return err
		}
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
			clone := integration.DeepCopy()
			var key k8sclient.ObjectKey
			key, err = k8sclient.ObjectKeyFromObject(clone)
			if err != nil {
				return err
			}
			err = c.Get(o.Context, key, clone)
			if err != nil {
				return err
			}

			integration.ObjectMeta.ResourceVersion = clone.ObjectMeta.ResourceVersion
		}
	}
	if o.Logs || o.Dev {
		err = k8slog.Print(o.Context, c, integration, cmd.OutOrStdout())
		if err != nil {
			return err
		}
	}

	if o.Sync && !o.Logs && !o.Dev {
		// Let's add a Wait point, otherwise the script terminates
		<-o.Context.Done()
	}

	if o.Save {
		name := o.GetIntegrationName(args)
		if name != "" {
			key := fmt.Sprintf("kamel.run.integration.%s", name)
			if err := saveDefaultConfig(cmd, "kamel.run", key); err != nil {
				return err
			}
		}
	}

	return nil
}

func (o *runCmdOptions) waitForIntegrationReady(cmd *cobra.Command, integration *v1.Integration) (*v1.IntegrationPhase, error) {
	handler := func(i *v1.Integration) bool {
		//
		// TODO when we add health checks, we should Wait until they are passed
		//
		if i.Status.Phase == v1.IntegrationPhaseRunning || i.Status.Phase == v1.IntegrationPhaseError {
			return false
		}

		return true
	}

	go watch.HandleIntegrationEvents(o.Context, integration, func(event *corev1.Event) bool {
		fmt.Fprintln(cmd.OutOrStdout(), event.Message)
		return true
	})

	return watch.HandleIntegrationStateChanges(o.Context, integration, handler)
}

func (o *runCmdOptions) syncIntegration(c client.Client, sources []string) error {
	// Let's watch all relevant files when in dev mode
	var files []string
	files = append(files, sources...)
	files = append(files, o.Resources...)
	files = append(files, o.PropertyFiles...)

	for _, s := range files {
		if !isRemoteHTTPFile(s) {
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
						_, err := o.updateIntegrationCode(c, sources)
						if err != nil {
							fmt.Println("Unable to sync integration: ", err.Error())
						}
					}
				}
			}()
		} else {
			fmt.Printf("WARNING: the following URL will not be watched for changes: %s\n", s)
		}
	}

	return nil
}

func (o *runCmdOptions) createIntegration(c client.Client, sources []string) (*v1.Integration, error) {
	return o.updateIntegrationCode(c, sources)
}

//nolint: gocyclo
func (o *runCmdOptions) updateIntegrationCode(c client.Client, sources []string) (*v1.Integration, error) {
	namespace := o.Namespace

	name := o.GetIntegrationName(sources)

	if name == "" {
		return nil, errors.New("unable to determine integration name")
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
			Dependencies:  make([]string, 0, len(o.Dependencies)),
			Kit:           o.IntegrationKit,
			Configuration: make([]v1.ConfigurationSpec, 0),
			Repositories:  o.Repositories,
			Profile:       v1.TraitProfileByName(o.Profile),
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

	for _, source := range sources {
		data, err := o.loadData(source, o.Compression)
		if err != nil {
			return nil, err
		}

		integration.Spec.AddSources(v1.SourceSpec{
			DataSpec: v1.DataSpec{
				Name:        path.Base(source),
				Content:     data,
				Compression: o.Compression,
			},
		})
	}

	for _, resource := range o.Resources {
		data, err := o.loadData(resource, o.Compression)
		if err != nil {
			return nil, err
		}

		integration.Spec.AddResources(v1.ResourceSpec{
			DataSpec: v1.DataSpec{
				Name:        path.Base(resource),
				Content:     data,
				Compression: o.Compression,
			},
			Type: v1.ResourceTypeData,
		})
	}

	for _, resource := range o.OpenAPIs {
		data, err := o.loadData(resource, o.Compression)
		if err != nil {
			return nil, err
		}

		integration.Spec.AddResources(v1.ResourceSpec{
			DataSpec: v1.DataSpec{
				Name:        path.Base(resource),
				Content:     data,
				Compression: o.Compression,
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

	for _, traitConf := range o.Traits {
		if err := o.configureTrait(&integration, traitConf); err != nil {
			return nil, err
		}
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
	err := c.Create(o.Context, &integration)
	if err != nil && k8serrors.IsAlreadyExists(err) {
		existed = true
		clone := integration.DeepCopy()
		var key k8sclient.ObjectKey
		key, err = k8sclient.ObjectKeyFromObject(clone)
		if err != nil {
			return nil, err
		}
		err = c.Get(o.Context, key, clone)
		if err != nil {
			return nil, err
		}
		integration.ResourceVersion = clone.ResourceVersion
		err = c.Update(o.Context, &integration)
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

func (*runCmdOptions) loadData(fileName string, compress bool) (string, error) {
	var content []byte
	var err error

	if !isRemoteHTTPFile(fileName) {
		content, err = ioutil.ReadFile(fileName)
		if err != nil {
			return "", err
		}
	} else {
		/* #nosec */
		resp, err := http.Get(fileName)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		content, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
	}

	if compress {
		var b bytes.Buffer

		if err := gzip.Compress(&b, content); err != nil {
			return "", err
		}

		return base64.StdEncoding.EncodeToString(b.Bytes()), nil
	}

	return string(content), nil
}

func (*runCmdOptions) configureTrait(integration *v1.Integration, config string) error {
	if integration.Spec.Traits == nil {
		integration.Spec.Traits = make(map[string]v1.TraitSpec)
	}

	parts := traitConfigRegexp.FindStringSubmatch(config)
	if len(parts) < 4 {
		return errors.New("unrecognized config format (expected \"<trait>.<prop>=<val>\"): " + config)
	}
	traitID := parts[1]
	prop := parts[2][1:]
	val := parts[3]

	spec, ok := integration.Spec.Traits[traitID]
	if !ok {
		spec = v1.TraitSpec{
			Configuration: make(map[string]string),
		}
	}

	if len(spec.Configuration[prop]) > 0 {
		// Aggregate multiple occurrences of the same option into a comma-separated string,
		// attempting to follow POSIX conventions.
		// This enables to execute:
		// $ kamel run -t <trait>.<property>=<value_1> ... -t <trait>.<property>=<value_N>
		// Or:
		// $ kamel run --trait <trait>.<property>=<value_1>,...,<trait>.<property>=<value_N>
		spec.Configuration[prop] = spec.Configuration[prop] + "," + val
	} else {
		spec.Configuration[prop] = val
	}
	integration.Spec.Traits[traitID] = spec
	return nil
}

func isRemoteHTTPFile(fileName string) bool {
	return strings.HasPrefix(fileName, "http://") || strings.HasPrefix(fileName, "https://")
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
