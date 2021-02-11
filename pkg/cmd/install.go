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
	"os"
	"regexp"
	"strings"
	"time"

	"go.uber.org/multierr"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/apache/camel-k/pkg/apis"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/install"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/olm"
	"github.com/apache/camel-k/pkg/util/registry"
	"github.com/apache/camel-k/pkg/util/watch"
)

const installCommand = "install"

func newCmdInstall(rootCmdOptions *RootCmdOptions) (*cobra.Command, *installCmdOptions) {
	options := installCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:     installCommand,
		Short:   "Install Camel K on a Kubernetes cluster",
		Long:    `Install Camel K on a Kubernetes or OpenShift cluster.`,
		PreRunE: options.decode,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.validate(cmd, args); err != nil {
				return err
			}
			if err := options.install(cmd, args); err != nil {
				if k8serrors.IsAlreadyExists(err) {
					return errors.Wrap(err, "Camel K seems already installed (use the --force option to overwrite existing resources)")
				}
				return err
			}
			return nil
		},
		PostRunE: options.postRun,
	}

	cmd.Flags().BoolP("wait", "w", false, "Wait for the platform to be running")
	cmd.Flags().Bool("cluster-setup", false, "Execute cluster-wide operations only (may require admin rights)")
	cmd.Flags().String("cluster-type", "", "Set explicitly the cluster type to Kubernetes or OpenShift")
	cmd.Flags().Bool("skip-operator-setup", false, "Do not install the operator in the namespace (in case there's a global one)")
	cmd.Flags().Bool("skip-cluster-setup", false, "Skip the cluster-setup phase")
	cmd.Flags().Bool("example", false, "Install example integration")
	cmd.Flags().Bool("global", false, "Configure the operator to watch all namespaces. No integration platform is created. You can run integrations in a namespace by installing an integration platform: 'kamel install --skip-operator-setup -n my-namespace'")
	cmd.Flags().Bool("force", false, "Force replacement of configuration resources when already present.")
	cmd.Flags().StringP("output", "o", "", "Output format. One of: json|yaml")
	cmd.Flags().String("organization", "", "A organization on the Docker registry that can be used to publish images")
	cmd.Flags().String("registry", "", "A Docker registry that can be used to publish images")
	cmd.Flags().String("registry-secret", "", "A secret used to push/pull images to the Docker registry")
	cmd.Flags().Bool("registry-insecure", false, "Configure to configure registry access in insecure mode or not")
	cmd.Flags().String("registry-auth-file", "", "A docker registry configuration file containing authorization tokens for pushing and pulling images")
	cmd.Flags().String("registry-auth-server", "", "The docker registry authentication server")
	cmd.Flags().String("registry-auth-username", "", "The docker registry authentication username")
	cmd.Flags().String("registry-auth-password", "", "The docker registry authentication password")
	cmd.Flags().StringArrayP("property", "p", nil, "Add a camel property")
	cmd.Flags().String("runtime-version", "", "Set the camel-k runtime version")
	cmd.Flags().String("base-image", "", "Set the base Image used to run integrations")
	cmd.Flags().String("operator-image", "", "Set the operator Image used for the operator deployment")
	cmd.Flags().String("operator-image-pull-policy", "", "Set the operator ImagePullPolicy used for the operator deployment")
	cmd.Flags().String("build-strategy", "", "Set the build strategy")
	cmd.Flags().String("build-publish-strategy", "", "Set the build publish strategy")
	cmd.Flags().String("build-timeout", "", "Set how long the build process can last")
	cmd.Flags().String("trait-profile", "", "The profile to use for traits")
	cmd.Flags().Bool("kaniko-build-cache", false, "To enable or disable the Kaniko cache")
	cmd.Flags().String("http-proxy-secret", "", "Configure the source of the secret holding HTTP proxy server details "+
		"(HTTP_PROXY|HTTPS_PROXY|NO_PROXY)")

	// OLM
	cmd.Flags().Bool("olm", true, "Try to install everything via OLM (Operator Lifecycle Manager) if available")
	cmd.Flags().String("olm-operator-name", olm.DefaultOperatorName, "Name of the Camel K operator in the OLM source or marketplace")
	cmd.Flags().String("olm-package", olm.DefaultPackage, "Name of the Camel K package in the OLM source or marketplace")
	cmd.Flags().String("olm-channel", olm.DefaultChannel, "Name of the Camel K channel in the OLM source or marketplace")
	cmd.Flags().String("olm-source", olm.DefaultSource, "Name of the OLM source providing the Camel K package (defaults to the standard Operator Hub source)")
	cmd.Flags().String("olm-source-namespace", olm.DefaultSourceNamespace, "Namespace where the OLM source is available")
	cmd.Flags().String("olm-starting-csv", olm.DefaultStartingCSV, "Allow to install a specific version from the operator source instead of latest available "+
		"from the channel")
	cmd.Flags().String("olm-global-namespace", olm.DefaultGlobalNamespace, "A namespace containing an OperatorGroup that defines global scope for the "+
		"operator (used in combination with the --global flag)")

	// Maven settings
	cmd.Flags().String("local-repository", "", "Location of the local maven repository")
	cmd.Flags().String("maven-settings", "", "Configure the source of the maven settings (configmap|secret:name[/key])")
	cmd.Flags().StringArray("maven-repository", nil, "Add a maven repository")

	// health
	cmd.Flags().Int("health-port", 8081, "The port of the health endpoint")

	// monitoring
	cmd.Flags().Bool("monitoring", false, "To enable or disable the operator monitoring")
	cmd.Flags().Int("monitoring-port", 8080, "The port of the metrics endpoint")

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
	Wait                    bool     `mapstructure:"wait"`
	ClusterSetupOnly        bool     `mapstructure:"cluster-setup"`
	SkipOperatorSetup       bool     `mapstructure:"skip-operator-setup"`
	SkipClusterSetup        bool     `mapstructure:"skip-cluster-setup"`
	ExampleSetup            bool     `mapstructure:"example"`
	Global                  bool     `mapstructure:"global"`
	KanikoBuildCache        bool     `mapstructure:"kaniko-build-cache"`
	Save                    bool     `mapstructure:"save" kamel:"omitsave"`
	Force                   bool     `mapstructure:"force"`
	Olm                     bool     `mapstructure:"olm"`
	ClusterType             string   `mapstructure:"cluster-type"`
	OutputFormat            string   `mapstructure:"output"`
	RuntimeVersion          string   `mapstructure:"runtime-version"`
	BaseImage               string   `mapstructure:"base-image"`
	OperatorImage           string   `mapstructure:"operator-image"`
	OperatorImagePullPolicy string   `mapstructure:"operator-image-pull-policy"`
	LocalRepository         string   `mapstructure:"local-repository"`
	BuildStrategy           string   `mapstructure:"build-strategy"`
	BuildPublishStrategy    string   `mapstructure:"build-publish-strategy"`
	BuildTimeout            string   `mapstructure:"build-timeout"`
	MavenRepositories       []string `mapstructure:"maven-repositories"`
	MavenSettings           string   `mapstructure:"maven-settings"`
	HealthPort              int32    `mapstructure:"health-port"`
	Monitoring              bool     `mapstructure:"monitoring"`
	MonitoringPort          int32    `mapstructure:"monitoring-port"`
	Properties              []string `mapstructure:"properties"`
	TraitProfile            string   `mapstructure:"trait-profile"`
	HTTPProxySecret         string   `mapstructure:"http-proxy-secret"`

	registry         v1.IntegrationPlatformRegistrySpec
	registryAuth     registry.Auth
	RegistryAuthFile string `mapstructure:"registry-auth-file"`

	olmOptions olm.Options
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
		var olmAvailable bool
		if olmAvailable, err = olm.IsAPIAvailable(o.Context, olmClient, o.Namespace); err != nil {
			return errors.Wrap(err, "error while checking OLM availability. Run with '--olm=false' to skip this check")
		}

		if olmAvailable {
			if installViaOLM, err = olm.HasPermissionToInstall(o.Context, olmClient, o.Namespace, o.Global, o.olmOptions); err != nil {
				return errors.Wrap(err, "error while checking permissions to install operator via OLM. Run with '--olm=false' to skip this check")
			}
			if !installViaOLM {
				fmt.Fprintln(cobraCmd.OutOrStdout(), "OLM is available but current user has not enough permissions to create the operator. "+
					"You can either ask your administrator to provide permissions (preferred) or run the install command with the `--olm=false` flag.")
				os.Exit(1)
			}
		}

		if installViaOLM {
			fmt.Fprintln(cobraCmd.OutOrStdout(), "OLM is available in the cluster")
			var installed bool
			if installed, err = olm.Install(o.Context, olmClient, o.Namespace, o.Global, o.olmOptions, collection); err != nil {
				return err
			}
			if !installed {
				fmt.Fprintln(cobraCmd.OutOrStdout(), "OLM resources are already available: skipping installation")
			}

			if err = install.WaitForAllCRDInstallation(o.Context, clientProvider, 90*time.Second); err != nil {
				return err
			}
		}
	}

	if !o.SkipClusterSetup && !installViaOLM {
		err := install.SetupClusterWideResourcesOrCollect(o.Context, clientProvider, collection, o.ClusterType)
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
			fmt.Fprintln(cobraCmd.OutOrStdout(), "Camel K cluster setup completed successfully")
		}
	} else {
		c, err := o.GetCmdClient()
		if err != nil {
			return err
		}

		namespace := o.Namespace

		if !o.SkipOperatorSetup && !installViaOLM {
			cfg := install.OperatorConfiguration{
				CustomImage:           o.OperatorImage,
				CustomImagePullPolicy: o.OperatorImagePullPolicy,
				Namespace:             namespace,
				Global:                o.Global,
				ClusterType:           o.ClusterType,
				Health: install.OperatorHealthConfiguration{
					Port: o.HealthPort,
				},
				Monitoring: install.OperatorMonitoringConfiguration{
					Enabled: o.Monitoring,
					Port:    o.MonitoringPort,
				},
			}
			err = install.OperatorOrCollect(o.Context, c, cfg, collection, o.Force)
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
			generatedSecretName, err = install.RegistrySecretOrCollect(o.Context, c, namespace, regData, collection, o.Force)
			if err != nil {
				return err
			}
		} else if o.RegistryAuthFile != "" {
			generatedSecretName, err = install.RegistrySecretFromFileOrCollect(o.Context, c, namespace, o.RegistryAuthFile, collection, o.Force)
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
			platform.Spec.Build.BuildStrategy = v1.IntegrationPlatformBuildStrategy(o.BuildStrategy)
		}
		if o.BuildPublishStrategy != "" {
			platform.Spec.Build.PublishStrategy = v1.IntegrationPlatformBuildPublishStrategy(o.BuildPublishStrategy)
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

		// Do not create an integration platform in global mode as platforms are expected
		// to be created in other namespaces.
		// In OLM mode, the operator is installed in an external namespace, so it's ok to install the platform locally.
		if !o.Global || installViaOLM {
			err = install.RuntimeObjectOrCollect(o.Context, c, namespace, collection, o.Force, platform)
			if err != nil {
				return err
			}
		}

		if o.ExampleSetup {
			err = install.ExampleOrCollect(o.Context, c, namespace, collection, o.Force)
			if err != nil {
				return err
			}
		}

		if collection == nil {
			if o.Wait {
				err = o.waitForPlatformReady(cobraCmd, platform)
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

	return nil
}

func (o *installCmdOptions) postRun(cmd *cobra.Command, _ []string) error {
	if o.Save {
		cfg, err := LoadConfiguration()
		if err != nil {
			return err
		}

		cfg.Update(cmd, pathToRoot(cmd), o, true)

		return cfg.Save()
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

// nolint:errcheck
func (o *installCmdOptions) waitForPlatformReady(cmd *cobra.Command, platform *v1.IntegrationPlatform) error {
	handler := func(i *v1.IntegrationPlatform) bool {
		if i.Status.Phase == v1.IntegrationPlatformPhaseReady || i.Status.Phase == v1.IntegrationPlatformPhaseError {
			return false
		}

		return true
	}

	go watch.HandleIntegrationPlatformEvents(o.Context, platform, func(event *corev1.Event) bool {
		fmt.Fprintln(cmd.OutOrStdout(), event.Message)
		return true
	})

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

	if o.registry.Secret != "" && (o.registryAuth.IsSet() || o.RegistryAuthFile != "") {
		err := fmt.Errorf("incompatible options combinations: you cannot set both registry-secret and registry-auth-[*] settings")
		result = multierr.Append(result, err)
	}

	if o.registryAuth.IsSet() && o.RegistryAuthFile != "" {
		err := fmt.Errorf("incompatible options combinations: you cannot set registry-auth-file with other registry-auth-[*] settings")
		result = multierr.Append(result, err)
	}

	if o.RegistryAuthFile != "" {
		nfo, err := os.Stat(o.RegistryAuthFile)
		if err != nil {
			result = multierr.Append(result, err)
		} else if nfo.IsDir() {
			result = multierr.Append(result, errors.Wrapf(err, "registry file cannot be a directory: %s", o.RegistryAuthFile))
		}
	}

	if o.BuildStrategy != "" {
		found := false
		for _, s := range v1.IntegrationPlatformBuildStrategies {
			if string(s) == o.BuildStrategy {
				found = true
				break
			}
		}
		if !found {
			var strategies []string
			for _, s := range v1.IntegrationPlatformBuildStrategies {
				strategies = append(strategies, string(s))
			}
			return fmt.Errorf("unknown build strategy: %s. One of [%s] is expected", o.BuildStrategy, strings.Join(strategies, ", "))
		}
	}

	if o.BuildPublishStrategy != "" {
		found := false
		for _, s := range v1.IntegrationPlatformBuildPublishStrategies {
			if string(s) == o.BuildPublishStrategy {
				found = true
				break
			}
		}
		if !found {
			var strategies []string
			for _, s := range v1.IntegrationPlatformBuildPublishStrategies {
				strategies = append(strategies, string(s))
			}
			return fmt.Errorf("unknown build publish strategy: %s. One of [%s] is expected", o.BuildPublishStrategy, strings.Join(strategies, ", "))
		}
	}

	return result
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
