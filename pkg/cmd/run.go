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
	"github.com/apache/camel-k/pkg/client"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/apache/camel-k/pkg/gzip"

	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/sync"
	"github.com/apache/camel-k/pkg/util/watch"
	"github.com/spf13/cobra"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	traitConfigRegexp = regexp.MustCompile(`^([a-z-]+)((?:\.[a-z-]+)+)=(.*)$`)
)

func newCmdRun(rootCmdOptions *RootCmdOptions) *cobra.Command {
	options := runCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:   "run [file to run]",
		Short: "Run a integration on Kubernetes",
		Long:  `Deploys and execute a integration pod on Kubernetes.`,
		Args:  options.validateArgs,
		RunE:  options.run,
	}

	cmd.Flags().StringVarP(&options.Runtime, "runtime", "r", "", "Runtime used by the integration")
	cmd.Flags().StringVar(&options.IntegrationName, "name", "", "The integration name")
	cmd.Flags().StringSliceVarP(&options.Dependencies, "dependency", "d", nil, "The integration dependency")
	cmd.Flags().BoolVarP(&options.Wait, "wait", "w", false, "Waits for the integration to be running")
	cmd.Flags().StringVarP(&options.IntegrationContext, "context", "x", "", "The contex used to run the integration")
	cmd.Flags().StringSliceVarP(&options.Properties, "property", "p", nil, "Add a camel property")
	cmd.Flags().StringSliceVar(&options.ConfigMaps, "configmap", nil, "Add a ConfigMap")
	cmd.Flags().StringSliceVar(&options.Secrets, "secret", nil, "Add a Secret")
	cmd.Flags().StringSliceVar(&options.Repositories, "repository", nil, "Add a maven repository")
	cmd.Flags().BoolVar(&options.Logs, "logs", false, "Print integration logs")
	cmd.Flags().BoolVar(&options.Sync, "sync", false, "Synchronize the local source file with the cluster, republishing at each change")
	cmd.Flags().BoolVar(&options.Dev, "dev", false, "Enable Dev mode (equivalent to \"-w --logs --sync\")")
	cmd.Flags().StringVar(&options.Profile, "profile", "", "Trait profile used for deployment")
	cmd.Flags().StringSliceVarP(&options.Traits, "trait", "t", nil, "Configure a trait. E.g. \"-t service.enabled=false\"")
	cmd.Flags().StringSliceVar(&options.LoggingLevels, "logging-level", nil, "Configure the logging level. "+
		"E.g. \"--logging-level org.apache.camel=DEBUG\"")
	cmd.Flags().StringVarP(&options.OutputFormat, "output", "o", "", "Output format. One of: json|yaml")
	cmd.Flags().BoolVar(&options.Compression, "compression", false, "Enable store source as a compressed binary blob")
	cmd.Flags().StringSliceVar(&options.Resources, "resource", nil, "Add a resource")

	// completion support
	configureKnownCompletions(&cmd)

	return &cmd
}

type runCmdOptions struct {
	*RootCmdOptions
	Compression        bool
	Wait               bool
	Logs               bool
	Sync               bool
	Dev                bool
	IntegrationContext string
	Runtime            string
	IntegrationName    string
	Profile            string
	OutputFormat       string
	Resources          []string
	Dependencies       []string
	Properties         []string
	ConfigMaps         []string
	Secrets            []string
	Repositories       []string
	Traits             []string
	LoggingLevels      []string
}

func (o *runCmdOptions) validateArgs(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("accepts at least 1 arg, received 0")
	}
	if len(args) > 1 && o.IntegrationName == "" {
		return errors.New("integration name is mandatory when using multiple sources")
	}

	for _, fileName := range args {
		if !strings.HasPrefix(fileName, "http://") && !strings.HasPrefix(fileName, "https://") {
			if _, err := os.Stat(fileName); err != nil && os.IsNotExist(err) {
				return errors.Wrap(err, "file "+fileName+" does not exist")
			} else if err != nil {
				return errors.Wrap(err, "error while accessing file "+fileName)
			}
		} else {
			resp, err := http.Get(fileName)
			if err != nil {
				return errors.Wrap(err, "The URL provided is not reachable")
			} else if resp.StatusCode != 200 {
				return errors.New("The URL provided is not reachable " + fileName + " The error code returned is " + strconv.Itoa(resp.StatusCode))
			}
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
			}
			os.Exit(1)
		}()
	}

	if o.Sync || o.Dev {
		err = o.syncIntegration(c, args)
		if err != nil {
			return err
		}
	}
	if o.Wait || o.Dev {
		err = o.waitForIntegrationReady(integration)
		if err != nil {
			return err
		}
	}
	if o.Logs || o.Dev {
		err = log.Print(o.Context, c, integration)
		if err != nil {
			return err
		}
	}

	if o.Sync && !o.Logs && !o.Dev {
		// Let's add a wait point, otherwise the script terminates
		<-o.Context.Done()
	}
	return nil
}

func (o *runCmdOptions) waitForIntegrationReady(integration *v1alpha1.Integration) error {
	handler := func(i *v1alpha1.Integration) bool {
		//
		// TODO when we add health checks, we should wait until they are passed
		//
		if i.Status.Phase != "" {
			fmt.Println("integration \""+integration.Name+"\" in phase", i.Status.Phase)

			if i.Status.Phase == v1alpha1.IntegrationPhaseRunning {
				// TODO display some error info when available in the status
				return false
			}

			if i.Status.Phase == v1alpha1.IntegrationPhaseError {
				fmt.Println("integration deployment failed")
				return false
			}
		}

		return true
	}

	return watch.HandleStateChanges(o.Context, integration, handler)
}

func (o *runCmdOptions) syncIntegration(c client.Client, sources []string) error {
	for _, s := range sources {
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
						logrus.Error("Unable to sync integration: ", err)
					}
				}
			}
		}()
	}

	return nil
}

func (o *runCmdOptions) createIntegration(c client.Client, sources []string) (*v1alpha1.Integration, error) {
	return o.updateIntegrationCode(c, sources)
}

func (o *runCmdOptions) updateIntegrationCode(c client.Client, sources []string) (*v1alpha1.Integration, error) {
	namespace := o.Namespace

	name := ""
	if o.IntegrationName != "" {
		name = o.IntegrationName
		name = kubernetes.SanitizeName(name)
	} else if len(sources) == 1 {
		name = kubernetes.SanitizeName(sources[0])
	}

	if name == "" {
		return nil, errors.New("unable to determine integration name")
	}

	integration := v1alpha1.Integration{
		TypeMeta: v1.TypeMeta{
			Kind:       v1alpha1.IntegrationKind,
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: v1alpha1.IntegrationSpec{
			Dependencies:  make([]string, 0, len(o.Dependencies)),
			Context:       o.IntegrationContext,
			Configuration: make([]v1alpha1.ConfigurationSpec, 0),
			Repositories:  o.Repositories,
			Profile:       v1alpha1.TraitProfileByName(o.Profile),
		},
	}

	for _, source := range sources {
		data, err := o.loadData(source, o.Compression)
		if err != nil {
			return nil, err
		}

		integration.Spec.AddSources(v1alpha1.SourceSpec{
			DataSpec: v1alpha1.DataSpec{
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

		integration.Spec.AddResources(v1alpha1.ResourceSpec{
			DataSpec: v1alpha1.DataSpec{
				Name:        path.Base(resource),
				Content:     data,
				Compression: o.Compression,
			},
		})
	}

	if o.Runtime != "" {
		integration.Spec.AddDependency("runtime:" + o.Runtime)
	}

	for _, item := range o.Dependencies {
		integration.Spec.AddDependency(item)
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
		key, err := k8sclient.ObjectKeyFromObject(clone)
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

func (*runCmdOptions) loadData(fileName string, compress bool) (string, error) {
	var content []byte
	var err error

	if !strings.HasPrefix(fileName, "http://") && !strings.HasPrefix(fileName, "https://") {
		content, err = ioutil.ReadFile(fileName)
		if err != nil {
			return "", err
		}
	} else {
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

func (*runCmdOptions) configureTrait(integration *v1alpha1.Integration, config string) error {
	if integration.Spec.Traits == nil {
		integration.Spec.Traits = make(map[string]v1alpha1.IntegrationTraitSpec)
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
		spec = v1alpha1.IntegrationTraitSpec{
			Configuration: make(map[string]string),
		}
	}

	spec.Configuration[prop] = val
	integration.Spec.Traits[traitID] = spec
	return nil
}
