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
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/apache/camel-k/pkg/util/olm"
	"github.com/apache/camel-k/pkg/util/registry"
	"go.uber.org/multierr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/apache/camel-k/deploy"
	"github.com/apache/camel-k/pkg/apis"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/install"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/watch"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

func newCmdInstall(rootCmdOptions *RootCmdOptions) (*cobra.Command, *installCmdOptions) {
	options := installCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:     "install",
		Short:   "Installs Camel K on a Kubernetes cluster",
		Long:    `Installs Camel K on a Kubernetes or OpenShift cluster.`,
		PreRunE: options.decode,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.validate(cmd, args); err != nil {
				return err
			}
			if err := options.install(cmd, args); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolP("wait", "w", false, "Waits for the platform to be running")
	cmd.Flags().Bool("cluster-setup", false, "Execute cluster-wide operations only (may require admin rights)")
	cmd.Flags().String("cluster-type", "", "Set explicitly the cluster type to Kubernetes or OpenShift")
	cmd.Flags().Bool("skip-operator-setup", false, "Do not install the operator in the namespace (in case there's a global one)")
	cmd.Flags().Bool("skip-cluster-setup", false, "Skip the cluster-setup phase")
	cmd.Flags().Bool("example", false, "Install example integration")
	cmd.Flags().Bool("global", false, "Configure the operator to watch all namespaces. No integration platform is created.")

	cmd.Flags().StringP("output", "o", "", "Output format. One of: json|yaml")
	cmd.Flags().String("organization", "", "A organization on the Docker registry that can be used to publish images")
	cmd.Flags().String("registry", "", "A Docker registry that can be used to publish images")
	cmd.Flags().String("registry-secret", "", "A secret used to push/pull images to the Docker registry")
	cmd.Flags().Bool("registry-insecure", false, "Configure to configure registry access in insecure mode or not")
	cmd.Flags().String("registry-auth-server", "", "The docker registry authentication server")
	cmd.Flags().String("registry-auth-username", "", "The docker registry authentication username")
	cmd.Flags().String("registry-auth-password", "", "The docker registry authentication password")
	cmd.Flags().StringArrayP("property", "p", nil, "Add a camel property")
	cmd.Flags().String("camel-version", "", "Set the camel version")
	cmd.Flags().String("runtime-version", "", "Set the camel-k runtime version")
	cmd.Flags().String("base-image", "", "Set the base Image used to run integrations")
	cmd.Flags().String("operator-image", "", "Set the operator Image used for the operator deployment")
	cmd.Flags().StringArray("kit", nil, "Add an integration kit to build at startup")
	cmd.Flags().String("build-strategy", "", "Set the build strategy")
	cmd.Flags().String("build-timeout", "", "Set how long the build process can last")
	cmd.Flags().String("trait-profile", "", "The profile to use for traits")
	cmd.Flags().Bool("kaniko-build-cache", false, "To enable or disable the Kaniko cache")
	cmd.Flags().String("http-proxy-secret", "", "Configure the source of the secret holding HTTP proxy server details "+
		"(HTTP_PROXY|HTTPS_PROXY|NO_PROXY)")

	// olm
	cmd.Flags().Bool("olm", true, "Try to install everything via OLM (Operator Lifecycle Manager) if available")
	cmd.Flags().String("olm-operator-name", olm.DefaultOperatorName, "Name of the Camel K operator in the OLM source or marketplace")
	cmd.Flags().String("olm-package", olm.DefaultPackage, "Name of the Camel K package in the OLM source or marketplace")
	cmd.Flags().String("olm-channel", olm.DefaultChannel, "Name of the Camel K channel in the OLM source or marketplace")
	cmd.Flags().String("olm-source", olm.DefaultSource, "Name of the OLM source providing the Camel K package (defaults to the standard Operator Hub source)")
	cmd.Flags().String("olm-source-namespace", olm.DefaultSourceNamespace, "Namespace where the OLM source is available")
	cmd.Flags().String("olm-starting-csv", olm.DefaultStartingCSV, "Allow to install a specific version from the operator source instead of latest available from the channel")
	cmd.Flags().String("olm-global-namespace", olm.DefaultGlobalNamespace, "A namespace containing an OperatorGroup that defines global scope for the operator (used in combination with the --global flag)")

	// maven settings
	cmd.Flags().String("local-repository", "", "Location of the local maven repository")
	cmd.Flags().String("maven-settings", "", "Configure the source of the maven settings (configmap|secret:name[/key])")
	cmd.Flags().StringArray("maven-repository", nil, "Add a maven repository")

	// save
	cmd.Flags().Bool("save", false, "Save the install parameters into the default kamel configuration file (kamel-config.yaml)")

	// completion support
	configureBashAnnotationForFlag(
		&cmd,
		"context",
		map[string][]string{
			cobra.BashCompCustom: {"__kamel_kubectl_get_known_integrationcontexts"},
		},
	)

	return &cmd, &options
}

type installCmdOptions struct {
	*RootCmdOptions
	Wait              bool     `mapstructure:"wait"`
	ClusterSetupOnly  bool     `mapstructure:"cluster-setup"`
	SkipOperatorSetup bool     `mapstructure:"skip-operator-setup"`
	SkipClusterSetup  bool     `mapstructure:"skip-cluster-setup"`
	ExampleSetup      bool     `mapstructure:"example"`
	Global            bool     `mapstructure:"global"`
	KanikoBuildCache  bool     `mapstructure:"kaniko-build-cache"`
	Save              bool     `mapstructure:"save"`
	Olm               bool    `mapstructure:"olm"`
	ClusterType       string   `mapstructure:"cluster-type"`
	OutputFormat      string   `mapstructure:"output"`
	RuntimeVersion    string   `mapstructure:"runtime-version"`
	BaseImage         string   `mapstructure:"base-image"`
	OperatorImage     string   `mapstructure:"operator-image"`
	LocalRepository   string   `mapstructure:"local-repository"`
	BuildStrategy     string   `mapstructure:"build-strategy"`
	BuildTimeout      string   `mapstructure:"build-timeout"`
	MavenRepositories []string `mapstructure:"maven-repositories"`
	MavenSettings     string   `mapstructure:"maven-settings"`
	Properties        []string `mapstructure:"properties"`
	Kits              []string `mapstructure:"kits"`
	TraitProfile      string   `mapstructure:"trait-profile"`
	HTTPProxySecret   string   `mapstructure:"http-proxy-secret"`

	registry     v1.IntegrationPlatformRegistrySpec
	registryAuth registry.Auth
	olmOptions   olm.Options
}

// nolint: gocyclo
func (o *installCmdOptions) install(cobraCmd *cobra.Command, _ []string) error {
	var collection *kubernetes.Collection
	if o.OutputFormat != "" {
		collection = kubernetes.NewCollection()
	}

	// Let's use a client provider during cluster installation, to eliminate the problem of CRD object caching
	clientProvider := client.Provider{Get: o.NewCmdClient}

	installViaOLM := false
	if o.Olm {
		var err error
		var olmClient client.Client
		if olmClient, err = clientProvider.Get(); err != nil {
			return err
		}
		if installViaOLM, err = olm.IsAvailable(o.Context, olmClient); err != nil {
			return errors.Wrap(err, "error while checking OLM availability. Run with '--olm=false' to skip this check")
		}

		if installViaOLM {
			fmt.Fprintln(cobraCmd.OutOrStdout(), "OLM is available in the cluster");
			if err = olm.Install(o.Context, olmClient, o.Namespace, o.Global, o.olmOptions, collection); err != nil {
				return err
			}
		}

		if err = install.WaitForAllCRDInstallation(o.Context, clientProvider, 90 * time.Second); err != nil {
			return err
		}
	}

	if !o.SkipClusterSetup && !installViaOLM {
		err := install.SetupClusterWideResourcesOrCollect(o.Context, clientProvider, collection)
		if err != nil && k8serrors.IsForbidden(err) {
			fmt.Fprintln(cobraCmd.OutOrStdout(), "Current user is not authorized to create cluster-wide objects like custom resource definitions or cluster roles: ", err)

			meg := `please login as cluster-admin and execute "kamel install --cluster-setup" to install cluster-wide resources (one-time operation)`
			return errors.New(meg)
		} else if err != nil {
			return err
		}
	}

	if o.ClusterSetupOnly {
		if collection == nil {
			fmt.Fprintln(cobraCmd.OutOrStdout(),"Camel K cluster setup completed successfully")
		}
	} else {
		c, err := o.GetCmdClient()
		if err != nil {
			return err
		}

		namespace := o.Namespace

		if !o.SkipOperatorSetup && !installViaOLM {
			cfg := install.OperatorConfiguration{
				CustomImage: o.OperatorImage,
				Namespace:   namespace,
				Global:      o.Global,
				ClusterType: o.ClusterType,
			}
			err = install.OperatorOrCollect(o.Context, c, cfg, collection)
			if err != nil {
				return err
			}
		} else if o.SkipOperatorSetup {
			fmt.Fprintln(cobraCmd.OutOrStdout(), "Camel K operator installation skipped")
		}

		generatedSecretName := ""
		if o.registryAuth.IsSet() {
			regData := o.registryAuth
			regData.Registry = o.registry.Address
			generatedSecretName, err = install.RegistrySecretOrCollect(o.Context, c, namespace, regData, collection)
			if err != nil {
				return err
			}
		}

		platform, err := install.PlatformOrCollect(o.Context, c, o.ClusterType, namespace, o.registry, collection)
		if err != nil {
			return err
		}

		if generatedSecretName != "" {
			platform.Spec.Build.Registry.Secret = generatedSecretName
		}

		if len(o.Properties) > 0 {
			platform.Spec.Build.Properties = make(map[string]string)

			for _, property := range o.Properties {
				kv := strings.Split(property, "=")

				if len(kv) == 2 {
					platform.Spec.Build.Properties[kv[0]] = kv[1]
				}
			}
		}
		if o.LocalRepository != "" {
			platform.Spec.Build.Maven.LocalRepository = o.LocalRepository
		}
		if o.RuntimeVersion != "" {
			platform.Spec.Build.RuntimeVersion = o.RuntimeVersion
		}
		if o.BaseImage != "" {
			platform.Spec.Build.BaseImage = o.BaseImage
		}
		if o.BuildStrategy != "" {
			switch s := o.BuildStrategy; s {
			case v1.IntegrationPlatformBuildStrategyPod:
				platform.Spec.Build.BuildStrategy = v1.IntegrationPlatformBuildStrategyPod
			case v1.IntegrationPlatformBuildStrategyRoutine:
				platform.Spec.Build.BuildStrategy = v1.IntegrationPlatformBuildStrategyRoutine
			default:
				return fmt.Errorf("unknown build strategy: %s", s)
			}
		}
		if o.BuildTimeout != "" {
			d, err := time.ParseDuration(o.BuildTimeout)
			if err != nil {
				return err
			}

			platform.Spec.Build.Timeout = &metav1.Duration{
				Duration: d,
			}
		}
		if o.TraitProfile != "" {
			platform.Spec.Profile = v1.TraitProfileByName(o.TraitProfile)
		}

		if len(o.MavenRepositories) > 0 {
			for _, r := range o.MavenRepositories {
				platform.AddConfiguration("repository", r)
			}
		}

		if o.MavenSettings != "" {
			mavenSettings, err := decodeMavenSettings(o.MavenSettings)
			if err != nil {
				return err
			}
			platform.Spec.Build.Maven.Settings = mavenSettings
		}

		if o.HTTPProxySecret != "" {
			platform.Spec.Build.HTTPProxySecret = o.HTTPProxySecret
		}

		if o.ClusterType != "" {
			for _, c := range v1.AllIntegrationPlatformClusters {
				if strings.EqualFold(string(c), o.ClusterType) {
					platform.Spec.Cluster = c
				}
			}
		}

		kanikoBuildCacheFlag := cobraCmd.Flags().Lookup("kaniko-build-cache")
		if kanikoBuildCacheFlag.Changed {
			platform.Spec.Build.KanikoBuildCache = &o.KanikoBuildCache
		}

		platform.Spec.Resources.Kits = o.Kits

		// Do not create an integration platform in global mode as platforms are expected
		// to be created in other namespaces.
		// In OLM mode, the operator is installed in an external namespace, so it's ok to install the platform locally.
		if !o.Global || installViaOLM {
			err = install.RuntimeObjectOrCollect(o.Context, c, namespace, collection, platform)
			if err != nil {
				return err
			}
		}

		if o.ExampleSetup {
			err = install.ExampleOrCollect(o.Context, c, namespace, collection)
			if err != nil {
				return err
			}
		}

		if collection == nil {
			if o.Wait {
				err = o.waitForPlatformReady(platform)
				if err != nil {
					return err
				}
			}

			strategy := ""
			if installViaOLM {
				strategy = "via OLM subscription"
			}
			if o.Global {
				fmt.Println("Camel K installed in namespace", namespace, strategy, "(global mode)")
			} else {
				fmt.Println("Camel K installed in namespace", namespace, strategy)
			}
		}
	}

	if collection != nil {
		return o.printOutput(collection)
	}

	if o.Save {
		if err := saveDefaultConfig(cobraCmd, "kamel.install", "kamel.install"); err != nil {
			return err
		}
	}

	return nil
}

func (o *installCmdOptions) printOutput(collection *kubernetes.Collection) error {
	lst := collection.AsKubernetesList()
	switch o.OutputFormat {
	case "yaml":
		data, err := kubernetes.ToYAML(lst)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
	case "json":
		data, err := kubernetes.ToJSON(lst)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
	default:
		return errors.New("unknown output format: " + o.OutputFormat)
	}
	return nil
}

func (o *installCmdOptions) waitForPlatformReady(platform *v1.IntegrationPlatform) error {
	handler := func(i *v1.IntegrationPlatform) bool {
		if i.Status.Phase != "" {
			fmt.Println("platform \""+platform.Name+"\" in phase", i.Status.Phase)

			if i.Status.Phase == v1.IntegrationPlatformPhaseReady {
				// TODO display some error info when available in the status
				return false
			}

			if i.Status.Phase == v1.IntegrationPlatformPhaseError {
				fmt.Println("platform installation failed")
				return false
			}
		}

		return true
	}

	return watch.HandlePlatformStateChanges(o.Context, platform, handler)
}

func (o *installCmdOptions) decode(cmd *cobra.Command, _ []string) error {
	path := pathToRoot(cmd)
	if err := decodeKey(o, path); err != nil {
		return err
	}

	o.registry.Address = viper.GetString(path + ".registry")
	o.registry.Organization = viper.GetString(path + ".organization")
	o.registry.Secret = viper.GetString(path + ".registry-secret")
	o.registry.Insecure = viper.GetBool(path + ".registry-insecure")
	o.registryAuth.Username = viper.GetString(path + ".registry-auth-username")
	o.registryAuth.Password = viper.GetString(path + ".registry-auth-password")
	o.registryAuth.Server = viper.GetString(path + ".registry-auth-server")

	o.olmOptions.OperatorName = viper.GetString(path + ".olm-operator-name")
	o.olmOptions.Package = viper.GetString(path + ".olm-package")
	o.olmOptions.Channel = viper.GetString(path + ".olm-channel")
	o.olmOptions.Source = viper.GetString(path + ".olm-source")
	o.olmOptions.SourceNamespace = viper.GetString(path + ".olm-source-namespace")
	o.olmOptions.StartingCSV = viper.GetString(path + ".olm-starting-csv")
	o.olmOptions.GlobalNamespace = viper.GetString(path + ".olm-global-namespace")

	return nil
}

func (o *installCmdOptions) validate(_ *cobra.Command, _ []string) error {
	var result error

	// Let's register only our own APIs
	schema := runtime.NewScheme()
	if err := apis.AddToScheme(schema); err != nil {
		return err
	}

	for _, kit := range o.Kits {
		err := errorIfKitIsNotAvailable(schema, kit)
		result = multierr.Append(result, err)
	}

	if len(o.MavenRepositories) > 0 && o.MavenSettings != "" {
		err := fmt.Errorf("incompatible options combinations: you cannot set both mavenRepository and mavenSettings")
		result = multierr.Append(result, err)
	}

	if o.TraitProfile != "" {
		tp := v1.TraitProfileByName(o.TraitProfile)
		if tp == v1.TraitProfile("") {
			err := fmt.Errorf("unknown trait profile %s", o.TraitProfile)
			result = multierr.Append(result, err)
		}
	}

	if o.registry.Secret != "" && o.registryAuth.IsSet() {
		err := fmt.Errorf("incompatible options combinations: you cannot set both registry-secret and registry-auth-[*] settings")
		result = multierr.Append(result, err)
	}

	return result
}

func errorIfKitIsNotAvailable(schema *runtime.Scheme, kit string) error {
	for _, name := range deploy.Resources("/") {
		resourceData := deploy.ResourceAsString(name)
		resource, err := kubernetes.LoadResourceFromYaml(schema, resourceData)
		if err != nil {
			// Not one of our registered schemas
			continue
		}
		kind := resource.GetObjectKind().GroupVersionKind()
		if kind.Kind != "IntegrationKit" {
			continue
		}
		integrationKit := resource.(*v1.IntegrationKit)
		if integrationKit.Name == kit {
			return nil
		}
	}
	return errors.Errorf("Unknown kit '%s'", kit)
}

func decodeMavenSettings(mavenSettings string) (v1.ValueSource, error) {
	sub := make([]string, 0)
	rex := regexp.MustCompile(`^(configmap|secret):([a-zA-Z0-9][a-zA-Z0-9-]*)(/([a-zA-Z0-9].*))?$`)
	hits := rex.FindAllStringSubmatch(mavenSettings, -1)

	for _, hit := range hits {
		if len(hit) > 1 {
			sub = append(sub, hit[1:]...)
		}
	}

	if len(sub) >= 2 {
		key := "settings.xml"

		if len(sub) == 4 {
			key = sub[3]
		}

		if sub[0] == "configmap" {
			return v1.ValueSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: sub[1],
					},
					Key: key,
				},
			}, nil
		}
		if sub[0] == "secret" {
			return v1.ValueSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: sub[1],
					},
					Key: key,
				},
			}, nil
		}
	}

	return v1.ValueSource{}, fmt.Errorf("illegal maven setting definition, syntax: configmap|secret:resource-name[/settings path]")
}
