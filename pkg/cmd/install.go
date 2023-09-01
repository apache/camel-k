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
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	platformutil "github.com/apache/camel-k/v2/pkg/platform"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.uber.org/multierr"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/cmd/set/env"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/builder"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/install"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/maven"
	"github.com/apache/camel-k/v2/pkg/util/olm"
	"github.com/apache/camel-k/v2/pkg/util/patch"
	"github.com/apache/camel-k/v2/pkg/util/registry"
	"github.com/apache/camel-k/v2/pkg/util/watch"
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
					return fmt.Errorf("camel K seems already installed (use the --force option to overwrite existing resources): %w", err)
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
	cmd.Flags().Bool("skip-registry-setup", false, "Skip the registry-setup phase (may negatively impact building of integrations)")
	cmd.Flags().Bool("skip-default-kamelets-setup", false, "Skip installation of the default Kamelets from catalog")
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
	cmd.Flags().StringP("operator-id", "x", "camel-k", "Set the operator id that is used to select the resources this operator should manage")
	cmd.Flags().String("operator-image", "", "Set the operator Image used for the operator deployment")
	cmd.Flags().String("operator-image-pull-policy", "", "Set the operator ImagePullPolicy used for the operator deployment")
	cmd.Flags().String("build-strategy", "", "Set the build strategy")
	cmd.Flags().String("build-order-strategy", "", "Set the build order strategy")
	cmd.Flags().String("build-publish-strategy", "", "Set the build publish strategy")
	cmd.Flags().StringArray("build-publish-strategy-option", nil, "Add a build publish strategy option, as <name=value>")
	cmd.Flags().String("build-timeout", "", "Set how long the build process can last")
	cmd.Flags().String("trait-profile", "", "The profile to use for traits")

	// OLM
	cmd.Flags().Bool("olm", true, "Try to install everything via OLM (Operator Lifecycle Manager) if available")
	cmd.Flags().String("olm-operator-name", "", "Name of the Camel K operator in the OLM source or marketplace")
	cmd.Flags().String("olm-package", "", "Name of the Camel K package in the OLM source or marketplace")
	cmd.Flags().String("olm-channel", "", "Name of the Camel K channel in the OLM source or marketplace")
	cmd.Flags().String("olm-source", "", "Name of the OLM source providing the Camel K package (defaults to the standard Operator Hub source)")
	cmd.Flags().String("olm-source-namespace", "", "Namespace where the OLM source is available")
	cmd.Flags().String("olm-starting-csv", "", "Allow to install a specific version from the operator source instead of latest available "+
		"from the channel")
	cmd.Flags().String("olm-global-namespace", "", "A namespace containing an OperatorGroup that defines global scope for the "+
		"operator (used in combination with the --global flag)")

	// Maven
	cmd.Flags().String("maven-local-repository", "", "Path of the local Maven repository")
	cmd.Flags().StringArray("maven-property", nil, "Add a Maven property")
	cmd.Flags().StringArray("maven-extension", nil, "Add a Maven build extension")
	cmd.Flags().String("maven-settings", "", "Configure the source of the Maven settings (configmap|secret:name[/key])")
	cmd.Flags().StringArray("maven-repository", nil, "Add a Maven repository")
	cmd.Flags().String("maven-ca-secret", "", "Configure the secret key containing the Maven CA certificates (secret/key)")
	cmd.Flags().StringArray("maven-cli-option", nil, "Add a default Maven CLI option to the list of arguments for Maven commands")

	// health
	cmd.Flags().Int("health-port", 8081, "The port of the health endpoint")

	// monitoring
	cmd.Flags().Bool("monitoring", false, "To enable or disable the operator monitoring")
	cmd.Flags().Int("monitoring-port", 8080, "The port of the metrics endpoint")

	// debugging
	cmd.Flags().Bool("debugging", false, "To enable or disable the operator debugging")
	cmd.Flags().Int("debugging-port", 4040, "The port of the debugger")
	cmd.Flags().String("debugging-path", "/usr/local/bin/kamel", "The path to the kamel executable file")

	// Operator settings
	cmd.Flags().StringArray("toleration", nil, "Add a Toleration to the operator Pod")
	cmd.Flags().StringArray("node-selector", nil, "Add a NodeSelector to the operator Pod")
	cmd.Flags().StringArray("operator-resources", nil, "Define the resources requests and limits assigned to the operator Pod as <requestType.requestResource=value> (i.e., limits.memory=256Mi)")
	cmd.Flags().StringArray("operator-env-vars", nil, "Add an environment variable to set in the operator Pod(s), as <name=value>")
	cmd.Flags().StringP("log-level", "z", "info", "The level of operator logging (default - info): info or 0, debug or 1")
	cmd.Flags().Int("max-running-pipelines", 0, "Maximum number of parallel running pipelines")

	// save
	cmd.Flags().Bool("save", false, "Save the install parameters into the default kamel configuration file (kamel-config.yaml)")

	return &cmd, &options
}

type installCmdOptions struct {
	*RootCmdOptions
	Wait                        bool `mapstructure:"wait"`
	ClusterSetupOnly            bool `mapstructure:"cluster-setup"`
	SkipOperatorSetup           bool `mapstructure:"skip-operator-setup"`
	SkipClusterSetup            bool `mapstructure:"skip-cluster-setup"`
	SkipRegistrySetup           bool `mapstructure:"skip-registry-setup"`
	SkipDefaultKameletsSetup    bool `mapstructure:"skip-default-kamelets-setup"`
	ExampleSetup                bool `mapstructure:"example"`
	Global                      bool `mapstructure:"global"`
	Save                        bool `mapstructure:"save" kamel:"omitsave"`
	Force                       bool `mapstructure:"force"`
	Olm                         bool `mapstructure:"olm"`
	olmOptions                  olm.Options
	ClusterType                 string   `mapstructure:"cluster-type"`
	OutputFormat                string   `mapstructure:"output"`
	RuntimeVersion              string   `mapstructure:"runtime-version"`
	BaseImage                   string   `mapstructure:"base-image"`
	OperatorID                  string   `mapstructure:"operator-id"`
	OperatorImage               string   `mapstructure:"operator-image"`
	OperatorImagePullPolicy     string   `mapstructure:"operator-image-pull-policy"`
	BuildStrategy               string   `mapstructure:"build-strategy"`
	BuildOrderStrategy          string   `mapstructure:"build-order-strategy"`
	BuildPublishStrategy        string   `mapstructure:"build-publish-strategy"`
	BuildPublishStrategyOptions []string `mapstructure:"build-publish-strategy-options"`
	BuildTimeout                string   `mapstructure:"build-timeout"`
	MavenExtensions             []string `mapstructure:"maven-extensions"`
	MavenLocalRepository        string   `mapstructure:"maven-local-repository"`
	MavenProperties             []string `mapstructure:"maven-properties"`
	MavenRepositories           []string `mapstructure:"maven-repositories"`
	MavenSettings               string   `mapstructure:"maven-settings"`
	MavenCASecret               string   `mapstructure:"maven-ca-secret"`
	MavenCLIOptions             []string `mapstructure:"maven-cli-options"`
	HealthPort                  int32    `mapstructure:"health-port"`
	MaxRunningBuilds            int32    `mapstructure:"max-running-pipelines"`
	Monitoring                  bool     `mapstructure:"monitoring"`
	MonitoringPort              int32    `mapstructure:"monitoring-port"`
	Debugging                   bool     `mapstructure:"debugging"`
	DebuggingPort               int32    `mapstructure:"debugging-port"`
	DebuggingPath               string   `mapstructure:"debugging-path"`
	TraitProfile                string   `mapstructure:"trait-profile"`
	Tolerations                 []string `mapstructure:"tolerations"`
	NodeSelectors               []string `mapstructure:"node-selectors"`
	ResourcesRequirements       []string `mapstructure:"operator-resources"`
	LogLevel                    string   `mapstructure:"log-level"`
	EnvVars                     []string `mapstructure:"operator-env-vars"`
	registry                    v1.RegistrySpec
	registryAuth                registry.Auth
	RegistryAuthFile            string `mapstructure:"registry-auth-file"`
}

func (o *installCmdOptions) install(cmd *cobra.Command, _ []string) error {
	var output *kubernetes.Collection
	if o.OutputFormat != "" {
		output = kubernetes.NewCollection()
	}

	// Let's use a client provider during cluster installation, to eliminate the problem of CRD object caching
	clientProvider := client.Provider{Get: o.NewCmdClient}

	o.setupEnvVars()

	installViaOLM := false
	if o.Olm {
		installed, err := o.tryInstallViaOLM(cmd, clientProvider, output)
		if err != nil {
			return err
		}
		installViaOLM = installed
	}

	if !o.SkipClusterSetup && !installViaOLM {
		if err := install.SetupClusterWideResourcesOrCollect(o.Context, clientProvider,
			output, o.ClusterType, o.Force); err != nil {
			if k8serrors.IsForbidden(err) {
				fmt.Fprintln(cmd.OutOrStdout(),
					"Current user is not authorized to create cluster-wide objects like custom resource definitions or cluster roles:",
					err)
				msg := `please login as cluster-admin and execute "kamel install --cluster-setup" to install cluster-wide resources (one-time operation)`
				return errors.New(msg)
			}
			return err
		}
	}

	if o.ClusterSetupOnly {
		if output != nil {
			return o.printOutput(cmd, output)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Camel K cluster setup completed successfully")
		return nil
	}

	return o.installOperator(cmd, output, installViaOLM)
}

// setupEnvVars sets up additional env vars from the command options.
func (o *installCmdOptions) setupEnvVars() {
	// --operator-id={id} is a syntax sugar for '--operator-env-vars KAMEL_OPERATOR_ID={id}'
	o.EnvVars = append(o.EnvVars, fmt.Sprintf("KAMEL_OPERATOR_ID=%s", strings.TrimSpace(o.OperatorID)))

	// --skip-default-kamelets-setup is a syntax sugar for '--operator-env-vars KAMEL_INSTALL_DEFAULT_KAMELETS=false'
	if o.SkipDefaultKameletsSetup {
		o.EnvVars = append(o.EnvVars, "KAMEL_INSTALL_DEFAULT_KAMELETS=false")
	}

	// Set the log-level
	if len(o.LogLevel) > 0 {
		o.EnvVars = append(o.EnvVars, fmt.Sprintf("LOG_LEVEL=%s", strings.TrimSpace(o.LogLevel)))
	}
}

func (o *installCmdOptions) tryInstallViaOLM(
	cmd *cobra.Command, clientProvider client.Provider, output *kubernetes.Collection,
) (bool, error) {
	olmClient, err := clientProvider.Get()
	if err != nil {
		return false, err
	}
	if olmAvailable, err := olm.IsAPIAvailable(o.Context, olmClient, o.Namespace); err != nil {
		return false, fmt.Errorf("error while checking OLM availability. Run with '--olm=false' to skip this check: %w", err)

	} else if !olmAvailable {
		fmt.Fprintln(cmd.OutOrStdout(), "OLM is not available in the cluster. Fallback to regular installation.")
		return false, nil
	}

	if hasPermission, err := olm.HasPermissionToInstall(o.Context, olmClient,
		o.Namespace, o.Global, o.olmOptions); err != nil {
		return false, fmt.Errorf("error while checking permissions to install operator via OLM. Run with '--olm=false' to skip this check: %w", err)

	} else if !hasPermission {
		return false, errors.New(
			"OLM is available but current user has not enough permissions to create the operator. " +
				"You can either ask your administrator to provide permissions (preferred) " +
				"or run the install command with the '--olm=false' flag")
	}

	// Install or collect via OLM
	fmt.Fprintln(cmd.OutOrStdout(), "OLM is available in the cluster")
	if installed, err := olm.Install(o.Context, olmClient, o.Namespace, o.Global, o.olmOptions, output,
		o.Tolerations, o.NodeSelectors, o.ResourcesRequirements, o.EnvVars); err != nil {
		return false, err
	} else if !installed {
		fmt.Fprintln(cmd.OutOrStdout(), "OLM resources are already available; skipping installation")
	}

	if err := install.WaitForAllCrdInstallation(o.Context, clientProvider, 90*time.Second); err != nil {
		return false, err
	}

	return true, nil
}

func (o *installCmdOptions) installOperator(cmd *cobra.Command, output *kubernetes.Collection, olm bool) error {
	operatorID, err := getOperatorID(o.EnvVars)
	if err != nil {
		return err
	}

	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}

	namespace := o.Namespace

	var platformName string
	if operatorID != "" {
		platformName = operatorID
	} else {
		platformName = platformutil.DefaultPlatformName
	}

	// Set up operator
	if !olm {
		if !o.SkipOperatorSetup {
			if err := o.setupOperator(cmd, c, namespace, platformName, output); err != nil {
				return err
			}
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "Camel K operator installation skipped")
		}
	}

	// Set up registry secret
	registrySecretName := ""
	if !o.SkipRegistrySetup {
		registrySecretName, err = o.setupRegistrySecret(c, namespace, output)
		if err != nil {
			return err
		}
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "Camel K operator registry setup skipped")
	}

	// Set up IntegrationPlatform
	platform, err := o.setupIntegrationPlatform(c, namespace, platformName, registrySecretName, output)
	if err != nil {
		return err
	}

	if output != nil {
		return o.printOutput(cmd, output)
	}

	if o.Wait {
		if err := o.waitForPlatformReady(cmd, platform); err != nil {
			return err
		}
	}

	strategy := ""
	if olm {
		strategy = "via OLM subscription"
	}
	if o.Global {
		fmt.Fprintln(cmd.OutOrStdout(), "Camel K installed in namespace", namespace, strategy, "(global mode)")
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "Camel K installed in namespace", namespace, strategy)
	}

	return nil
}

func getOperatorID(vars []string) (string, error) {
	envs, _, _, err := env.ParseEnv(vars, nil)
	if err != nil {
		return "", err
	}
	for _, e := range envs {
		if e.Name == "KAMEL_OPERATOR_ID" {
			return e.Value, nil
		}
	}

	return "", nil
}

func (o *installCmdOptions) setupOperator(
	cmd *cobra.Command, c client.Client, namespace string, platformName string, output *kubernetes.Collection,
) error {
	if ok, err := isInstallAllowed(o.Context, c, platformName, o.Force, cmd.OutOrStdout()); err != nil {
		if k8serrors.IsForbidden(err) {
			o.PrintfVerboseOutf(cmd,
				"Unable to verify existence of operator id %q due to lack of user privileges\n",
				platformName)
		} else {
			return err
		}
	} else if !ok {
		return fmt.Errorf(
			"installation not allowed because operator with id %q already exists; use the --force option to skip this check",
			platformName)
	}

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
		Debugging: install.OperatorDebuggingConfiguration{
			Enabled: o.Debugging,
			Port:    o.DebuggingPort,
			Path:    o.DebuggingPath,
		},
		Tolerations:           o.Tolerations,
		NodeSelectors:         o.NodeSelectors,
		ResourcesRequirements: o.ResourcesRequirements,
		EnvVars:               o.EnvVars,
	}

	return install.OperatorOrCollect(o.Context, cmd, c, cfg, output, o.Force)
}

func isInstallAllowed(ctx context.Context, c client.Client, operatorID string, force bool, out io.Writer) (bool, error) {
	// find existing platform with given name in any namespace
	pl, err := platformutil.LookupForPlatformName(ctx, c, operatorID)

	if pl != nil && force {
		fmt.Fprintf(out, "Overwriting existing operator with id %q\n", operatorID)
		return true, nil
	}

	// only allow installation when platform with given name is not found
	return pl == nil, err
}

func (o *installCmdOptions) setupRegistrySecret(c client.Client, namespace string, output *kubernetes.Collection) (string, error) {
	if o.registryAuth.IsSet() {
		regData := o.registryAuth
		regData.Registry = o.registry.Address
		return install.RegistrySecretOrCollect(o.Context, c, namespace, regData, output, o.Force)
	} else if o.RegistryAuthFile != "" {
		return install.RegistrySecretFromFileOrCollect(o.Context, c, namespace, o.RegistryAuthFile, output, o.Force)
	}

	return "", nil
}

func (o *installCmdOptions) setupIntegrationPlatform(c client.Client, namespace string, platformName string, registrySecretName string,
	output *kubernetes.Collection,
) (*v1.IntegrationPlatform, error) {
	platform, err := install.NewPlatform(o.Context, c, o.ClusterType, o.SkipRegistrySetup, o.registry, platformName)
	if err != nil {
		return nil, err
	}

	if registrySecretName != "" {
		platform.Spec.Build.Registry.Secret = registrySecretName
	}

	if len(o.MavenProperties) > 0 {
		platform.Spec.Build.Maven.Properties = make(map[string]string)
		for _, property := range o.MavenProperties {
			kv := strings.Split(property, "=")
			if len(kv) == 2 {
				platform.Spec.Build.Maven.Properties[kv[0]] = kv[1]
			}
		}
	}

	if size := len(o.MavenExtensions); size > 0 {
		platform.Spec.Build.Maven.Extension = make([]v1.MavenArtifact, 0, size)
		for _, extension := range o.MavenExtensions {
			gav := strings.Split(extension, ":")
			if len(gav) != 2 && len(gav) != 3 {
				msg := fmt.Sprintf("Maven build extension GAV must match <groupId>:<artifactId>:<version>, found: %s", extension)
				return nil, errors.New(msg)
			}
			ext := v1.MavenArtifact{
				GroupID:    gav[0],
				ArtifactID: gav[1],
			}
			if len(gav) == 3 {
				ext.Version = gav[2]
			}
			platform.Spec.Build.Maven.Extension = append(platform.Spec.Build.Maven.Extension, ext)
		}
	}

	if o.MavenLocalRepository != "" {
		platform.Spec.Build.Maven.LocalRepository = o.MavenLocalRepository
	}

	if len(o.MavenCLIOptions) > 0 {
		platform.Spec.Build.Maven.CLIOptions = o.MavenCLIOptions
	}

	if o.RuntimeVersion != "" {
		platform.Spec.Build.RuntimeVersion = o.RuntimeVersion
	}
	if o.BaseImage != "" {
		platform.Spec.Build.BaseImage = o.BaseImage
	}
	if o.BuildStrategy != "" {
		platform.Spec.Build.BuildConfiguration.Strategy = v1.BuildStrategy(o.BuildStrategy)
	}
	if o.BuildOrderStrategy != "" {
		platform.Spec.Build.BuildConfiguration.OrderStrategy = v1.BuildOrderStrategy(o.BuildOrderStrategy)
	}
	if o.BuildPublishStrategy != "" {
		platform.Spec.Build.PublishStrategy = v1.IntegrationPlatformBuildPublishStrategy(o.BuildPublishStrategy)
	}
	if o.BuildTimeout != "" {
		d, err := time.ParseDuration(o.BuildTimeout)
		if err != nil {
			return nil, err
		}

		platform.Spec.Build.Timeout = &metav1.Duration{
			Duration: d,
		}
	}
	if o.MaxRunningBuilds > 0 {
		platform.Spec.Build.MaxRunningBuilds = o.MaxRunningBuilds
	}

	if o.TraitProfile != "" {
		platform.Spec.Profile = v1.TraitProfileByName(o.TraitProfile)
	}

	if len(o.MavenRepositories) > 0 {
		settings, err := maven.NewSettings(maven.Repositories(o.MavenRepositories...), maven.DefaultRepositories)
		if err != nil {
			return nil, err
		}
		err = createDefaultMavenSettingsConfigMap(o.Context, c, namespace, platform.Name, settings)
		if err != nil {
			return nil, err
		}
		platform.Spec.Build.Maven.Settings.ConfigMapKeyRef = &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: platform.Name + "-maven-settings",
			},
			Key: "settings.xml",
		}
	}

	if o.MavenSettings != "" {
		mavenSettings, err := v1.DecodeValueSource(o.MavenSettings, "settings.xml",
			"illegal maven setting definition, syntax: configmap|secret:resource-name[/settings path]")
		if err != nil {
			return nil, err
		}
		platform.Spec.Build.Maven.Settings = mavenSettings
	}

	if o.MavenCASecret != "" {
		secret, err := decodeSecretKeySelector(o.MavenCASecret)
		if err != nil {
			return nil, err
		}
		platform.Spec.Build.Maven.CASecrets = append(platform.Spec.Build.Maven.CASecrets, *secret)
	}

	if o.ClusterType != "" {
		for _, c := range v1.AllIntegrationPlatformClusters {
			if strings.EqualFold(string(c), o.ClusterType) {
				platform.Spec.Cluster = c
			}
		}
	}
	if len(o.BuildPublishStrategyOptions) > 0 {
		if err = o.addBuildPublishStrategyOptions(&platform.Spec.Build); err != nil {
			return nil, err
		}
	}
	// Always create a platform in the namespace where the operator is located
	err = install.ObjectOrCollect(o.Context, c, namespace, output, o.Force, platform)
	if err != nil {
		return nil, err
	}

	if err := install.IntegrationPlatformViewerRole(o.Context, c, namespace); err != nil && !k8serrors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("error while installing global IntegrationPlatform viewer role: %w", err)
	}

	if o.ExampleSetup {
		err = install.ExampleOrCollect(o.Context, c, namespace, output, o.Force)
		if err != nil {
			return nil, err
		}
	}

	return platform, nil
}

func (o *installCmdOptions) printOutput(cmd *cobra.Command, collection *kubernetes.Collection) error {
	lst := collection.AsKubernetesList()
	switch o.OutputFormat {
	case "yaml":
		data, err := kubernetes.ToYAML(lst)
		if err != nil {
			return err
		}
		fmt.Fprint(cmd.OutOrStdout(), string(data))
	case "json":
		data, err := kubernetes.ToJSON(lst)
		if err != nil {
			return err
		}
		fmt.Fprint(cmd.OutOrStdout(), (data))
	default:
		return errors.New("unknown output format: " + o.OutputFormat)
	}
	return nil
}

// nolint:errcheck
func (o *installCmdOptions) waitForPlatformReady(cmd *cobra.Command, platform *v1.IntegrationPlatform) error {
	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}

	handler := func(i *v1.IntegrationPlatform) bool {
		if i.Status.Phase == v1.IntegrationPlatformPhaseReady || i.Status.Phase == v1.IntegrationPlatformPhaseError {
			return false
		}
		return true
	}

	go watch.HandleIntegrationPlatformEvents(o.Context, c, platform, func(event *corev1.Event) bool {
		fmt.Fprintln(cmd.OutOrStdout(), event.Message)
		return true
	})

	return watch.HandlePlatformStateChanges(o.Context, c, platform, handler)
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

	if o.OperatorID == "" {
		result = multierr.Append(result, fmt.Errorf("cannot use empty operator id"))
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
			result = multierr.Append(result, fmt.Errorf("registry file cannot be a directory: %s: %w", o.RegistryAuthFile, err))
		}
	}

	if o.BuildStrategy != "" {
		found := false
		for _, s := range v1.BuildStrategies {
			if string(s) == o.BuildStrategy {
				found = true
				break
			}
		}
		if !found {
			var strategies []string
			for _, s := range v1.BuildStrategies {
				strategies = append(strategies, string(s))
			}
			return fmt.Errorf("unknown build strategy: %s. One of [%s] is expected", o.BuildStrategy, strings.Join(strategies, ", "))
		}
	}

	if o.BuildOrderStrategy != "" {
		found := false
		for _, s := range v1.BuildOrderStrategies {
			if string(s) == o.BuildOrderStrategy {
				found = true
				break
			}
		}
		if !found {
			var strategies []string
			for _, s := range v1.BuildOrderStrategies {
				strategies = append(strategies, string(s))
			}
			return fmt.Errorf("unknown build order strategy: %s. One of [%s] is expected", o.BuildOrderStrategy, strings.Join(strategies, ", "))
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

// addBuildPublishStrategyOptions parses and adds all the build publish strategy options to the given IntegrationPlatformBuildSpec.
func (o *installCmdOptions) addBuildPublishStrategyOptions(pipeline *v1.IntegrationPlatformBuildSpec) error {
	for _, option := range o.BuildPublishStrategyOptions {
		kv := strings.Split(option, "=")
		if len(kv) == 2 {
			key := kv[0]
			if builder.IsSupportedPublishStrategyOption(pipeline.PublishStrategy, key) {
				pipeline.AddOption(key, kv[1])
			} else {
				return fmt.Errorf("build publish strategy option '%s' not supported. %s", option, supportedOptionsAsString(pipeline.PublishStrategy))
			}
		} else {
			return fmt.Errorf("build publish strategy option '%s' not in the expected format (name=value)", option)
		}
	}
	return nil
}

// supportedOptionsAsString provides all the supported options for the given strategy as string.
func supportedOptionsAsString(strategy v1.IntegrationPlatformBuildPublishStrategy) string {
	options := builder.GetSupportedPublishStrategyOptions(strategy)
	if len(options) == 0 {
		return fmt.Sprintf("no options are supported for the strategy '%s'.", strategy)
	}
	var sb strings.Builder
	for _, supportedOption := range builder.GetSupportedPublishStrategyOptions(strategy) {
		sb.WriteString(fmt.Sprintf("* %s\n", supportedOption.ToString()))
	}
	return fmt.Sprintf("\n\nSupported options for the strategy '%s':\n\n%s", strategy, sb.String())
}

func decodeMavenSettings(mavenSettings string) (v1.ValueSource, error) {
	return v1.DecodeValueSource(mavenSettings, "settings.xml", "illegal maven setting definition, syntax: configmap|secret:resource-name[/settings path]")
}

func decodeSecretKeySelector(secretKey string) (*corev1.SecretKeySelector, error) {
	r := regexp.MustCompile(`^([a-zA-Z0-9-]*)/([a-zA-Z0-9].*)$`)

	if !r.MatchString(secretKey) {
		return nil, fmt.Errorf("illegal Maven CA certificates secret key selector, syntax: secret-name/secret-key")
	}

	match := r.FindStringSubmatch(secretKey)

	return &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: match[1],
		},
		Key: match[2],
	}, nil
}

func createDefaultMavenSettingsConfigMap(ctx context.Context, client client.Client, namespace, name string, settings maven.Settings) error {
	cm, err := settingsConfigMap(namespace, name, settings)
	if err != nil {
		return err
	}

	err = client.Create(ctx, cm)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	} else if k8serrors.IsAlreadyExists(err) {
		existing := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cm.Namespace,
				Name:      cm.Name,
			},
		}
		err = client.Get(ctx, ctrl.ObjectKeyFromObject(existing), existing)
		if err != nil {
			return err
		}

		p, err := patch.MergePatch(existing, cm)
		if err != nil {
			return err
		} else if len(p) != 0 {
			err = client.Patch(ctx, cm, ctrl.RawPatch(types.MergePatchType, p))
			if err != nil {
				return fmt.Errorf("error during patch resource: %w", err)
			}
		}
	}

	return nil
}

func settingsConfigMap(namespace string, name string, settings maven.Settings) (*corev1.ConfigMap, error) {
	data, err := util.EncodeXML(settings)
	if err != nil {
		return nil, err
	}

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-maven-settings",
			Namespace: namespace,
			Labels: map[string]string{
				"app": "camel-k",
			},
		},
		Data: map[string]string{
			"settings.xml": string(data),
		},
	}

	return cm, nil
}
