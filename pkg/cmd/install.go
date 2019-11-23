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

	"github.com/apache/camel-k/pkg/util/registry"
	"go.uber.org/multierr"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/apache/camel-k/deploy"
	"github.com/apache/camel-k/pkg/apis"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/install"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/watch"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

func newCmdInstall(rootCmdOptions *RootCmdOptions) *cobra.Command {
	impl := installCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:     "install",
		Short:   "Install Camel K on a Kubernetes cluster",
		Long:    `Installs Camel K on a Kubernetes or OpenShift cluster.`,
		PreRunE: impl.validate,
		RunE:    impl.install,
	}

	cmd.Flags().BoolVarP(&impl.wait, "wait", "w", false, "Waits for the platform to be running")
	cmd.Flags().BoolVar(&impl.clusterSetupOnly, "cluster-setup", false, "Execute cluster-wide operations only (may require admin rights)")
	cmd.Flags().BoolVar(&impl.skipOperatorSetup, "skip-operator-setup", false, "Do not install the operator in the namespace (in case there's a global one)")
	cmd.Flags().BoolVar(&impl.skipClusterSetup, "skip-cluster-setup", false, "Skip the cluster-setup phase")
	cmd.Flags().BoolVar(&impl.exampleSetup, "example", false, "Install example integration")
	cmd.Flags().BoolVar(&impl.global, "global", false, "Configure the operator to watch all namespaces. No integration platform is created.")

	cmd.Flags().StringVarP(&impl.outputFormat, "output", "o", "", "Output format. One of: json|yaml")
	cmd.Flags().StringVar(&impl.registry.Organization, "organization", "", "A organization on the Docker registry that can be used to publish images")
	cmd.Flags().StringVar(&impl.registry.Address, "registry", "", "A Docker registry that can be used to publish images")
	cmd.Flags().StringVar(&impl.registry.Secret, "registry-secret", "", "A secret used to push/pull images to the Docker registry")
	cmd.Flags().BoolVar(&impl.registry.Insecure, "registry-insecure", false, "Configure to configure registry access in insecure mode or not")
	cmd.Flags().StringVar(&impl.registryAuth.Provider, "registry-auth-provider", "", "The docker registry authentication provider")
	cmd.Flags().StringVar(&impl.registryAuth.Username, "registry-auth-username", "", "The docker registry authentication username")
	cmd.Flags().StringVar(&impl.registryAuth.Password, "registry-auth-password", "", "The docker registry authentication password")
	cmd.Flags().StringSliceVarP(&impl.properties, "property", "p", nil, "Add a camel property")
	cmd.Flags().StringVar(&impl.camelVersion, "camel-version", "", "Set the camel version")
	cmd.Flags().StringVar(&impl.runtimeVersion, "runtime-version", "", "Set the camel-k runtime version")
	cmd.Flags().StringVar(&impl.baseImage, "base-image", "", "Set the base image used to run integrations")
	cmd.Flags().StringVar(&impl.operatorImage, "operator-image", "", "Set the operator image used for the operator deployment")
	cmd.Flags().StringSliceVar(&impl.kits, "kit", nil, "Add an integration kit to build at startup")
	cmd.Flags().StringVar(&impl.buildStrategy, "build-strategy", "", "Set the build strategy")
	cmd.Flags().StringVar(&impl.buildTimeout, "build-timeout", "", "Set how long the build process can last")
	cmd.Flags().StringVar(&impl.traitProfile, "trait-profile", "", "The profile to use for traits")
	cmd.Flags().BoolVar(&impl.kanikoBuildCache, "kaniko-build-cache", true, "To enable or disable the Kaniko Cache in building phase")
	cmd.Flags().StringVar(&impl.httpProxySecret, "http-proxy-secret", "", "Configure the source of the secret holding HTTP proxy server details "+
		"(HTTP_PROXY|HTTPS_PROXY|NO_PROXY)")

	// maven settings
	cmd.Flags().StringVar(&impl.localRepository, "local-repository", "", "Location of the local maven repository")
	cmd.Flags().StringVar(&impl.mavenSettings, "maven-settings", "", "Configure the source of the maven settings (configmap|secret:name[/key])")
	cmd.Flags().StringSliceVar(&impl.mavenRepositories, "maven-repository", nil, "Add a maven repository")

	// completion support
	configureBashAnnotationForFlag(
		&cmd,
		"context",
		map[string][]string{
			cobra.BashCompCustom: {"__kamel_kubectl_get_known_integrationcontexts"},
		},
	)

	return &cmd
}

type installCmdOptions struct {
	*RootCmdOptions
	wait              bool
	clusterSetupOnly  bool
	skipOperatorSetup bool
	skipClusterSetup  bool
	exampleSetup      bool
	global            bool
	kanikoBuildCache  bool
	outputFormat      string
	camelVersion      string
	runtimeVersion    string
	baseImage         string
	operatorImage     string
	localRepository   string
	buildStrategy     string
	buildTimeout      string
	mavenRepositories []string
	mavenSettings     string
	properties        []string
	kits              []string
	registry          v1alpha1.IntegrationPlatformRegistrySpec
	registryAuth      registry.Auth
	traitProfile      string
	httpProxySecret   string
}

// nolint: gocyclo
func (o *installCmdOptions) install(cobraCmd *cobra.Command, _ []string) error {
	var collection *kubernetes.Collection
	if o.outputFormat != "" {
		collection = kubernetes.NewCollection()
	}

	if !o.skipClusterSetup {
		// Let's use a client provider during cluster installation, to eliminate the problem of CRD object caching
		clientProvider := client.Provider{Get: o.NewCmdClient}

		err := install.SetupClusterWideResourcesOrCollect(o.Context, clientProvider, collection)
		if err != nil && k8serrors.IsForbidden(err) {
			fmt.Println("Current user is not authorized to create cluster-wide objects like custom resource definitions or cluster roles: ", err)

			meg := `please login as cluster-admin and execute "kamel install --cluster-setup" to install cluster-wide resources (one-time operation)`
			return errors.New(meg)
		} else if err != nil {
			return err
		}
	}

	if o.clusterSetupOnly {
		if collection == nil {
			fmt.Println("Camel K cluster setup completed successfully")
		}
	} else {
		c, err := o.GetCmdClient()
		if err != nil {
			return err
		}

		namespace := o.Namespace

		if !o.skipOperatorSetup {
			cfg := install.OperatorConfiguration{
				CustomImage: o.operatorImage,
				Namespace:   namespace,
				Global:      o.global,
			}
			err = install.OperatorOrCollect(o.Context, c, cfg, collection)
			if err != nil {
				return err
			}
		} else {
			fmt.Println("Camel K operator installation skipped")
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

		platform, err := install.PlatformOrCollect(o.Context, c, namespace, o.registry, collection)
		if err != nil {
			return err
		}

		if generatedSecretName != "" {
			platform.Spec.Build.Registry.Secret = generatedSecretName
		}

		if len(o.properties) > 0 {
			platform.Spec.Build.Properties = make(map[string]string)

			for _, property := range o.properties {
				kv := strings.Split(property, "=")

				if len(kv) == 2 {
					platform.Spec.Build.Properties[kv[0]] = kv[1]
				}
			}
		}
		if o.localRepository != "" {
			platform.Spec.Build.Maven.LocalRepository = o.localRepository
		}
		if o.camelVersion != "" {
			platform.Spec.Build.CamelVersion = o.camelVersion
		}
		if o.runtimeVersion != "" {
			platform.Spec.Build.RuntimeVersion = o.runtimeVersion
		}
		if o.baseImage != "" {
			platform.Spec.Build.BaseImage = o.baseImage
		}
		if o.buildStrategy != "" {
			switch s := o.buildStrategy; s {
			case v1alpha1.IntegrationPlatformBuildStrategyPod:
				platform.Spec.Build.BuildStrategy = v1alpha1.IntegrationPlatformBuildStrategyPod
			case v1alpha1.IntegrationPlatformBuildStrategyRoutine:
				platform.Spec.Build.BuildStrategy = v1alpha1.IntegrationPlatformBuildStrategyRoutine
			default:
				return fmt.Errorf("unknown build strategy: %s", s)
			}
		}
		if o.buildTimeout != "" {
			d, err := time.ParseDuration(o.buildTimeout)
			if err != nil {
				return err
			}

			platform.Spec.Build.Timeout.Duration = d
		}
		if o.traitProfile != "" {
			platform.Spec.Profile = v1alpha1.TraitProfileByName(o.traitProfile)
		}

		if len(o.mavenRepositories) > 0 {
			for _, r := range o.mavenRepositories {
				platform.AddConfiguration("repository", r)
			}
		}

		if o.mavenSettings != "" {
			mavenSettings, err := decodeMavenSettings(o.mavenSettings)
			if err != nil {
				return err
			}
			platform.Spec.Build.Maven.Settings = mavenSettings
		}

		if o.httpProxySecret != "" {
			platform.Spec.Build.HTTPProxySecret = o.httpProxySecret
		}

		kanikoBuildCacheFlag := cobraCmd.Flags().Lookup("kaniko-build-cache")

		defaultKanikoBuildCache := true

		if kanikoBuildCacheFlag.Changed {
			platform.Spec.Build.KanikoBuildCache = &o.kanikoBuildCache
		} else {
			platform.Spec.Build.KanikoBuildCache = &defaultKanikoBuildCache
		}

		platform.Spec.Resources.Kits = o.kits

		// Do not create an integration platform in global mode as platforms are expected
		// to be created in other namespaces
		if !o.global {
			err = install.RuntimeObjectOrCollect(o.Context, c, namespace, collection, platform)
			if err != nil {
				return err
			}
		}

		if o.exampleSetup {
			err = install.ExampleOrCollect(o.Context, c, namespace, collection)
			if err != nil {
				return err
			}
		}

		if collection == nil {
			if o.wait {
				err = o.waitForPlatformReady(platform)
				if err != nil {
					return err
				}
			}

			if o.global {
				fmt.Println("Camel K installed in namespace", namespace, "(global mode)")
			} else {
				fmt.Println("Camel K installed in namespace", namespace)
			}
		}
	}

	if collection != nil {
		return o.printOutput(collection)
	}

	return nil
}

func (o *installCmdOptions) printOutput(collection *kubernetes.Collection) error {
	lst := collection.AsKubernetesList()
	switch o.outputFormat {
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
		return errors.New("unknown output format: " + o.outputFormat)
	}
	return nil
}

func (o *installCmdOptions) waitForPlatformReady(platform *v1alpha1.IntegrationPlatform) error {
	handler := func(i *v1alpha1.IntegrationPlatform) bool {
		if i.Status.Phase != "" {
			fmt.Println("platform \""+platform.Name+"\" in phase", i.Status.Phase)

			if i.Status.Phase == v1alpha1.IntegrationPlatformPhaseReady {
				// TODO display some error info when available in the status
				return false
			}

			if i.Status.Phase == v1alpha1.IntegrationPlatformPhaseError {
				fmt.Println("platform installation failed")
				return false
			}
		}

		return true
	}

	return watch.HandlePlatformStateChanges(o.Context, platform, handler)
}

func (o *installCmdOptions) validate(_ *cobra.Command, _ []string) error {
	var result error

	// Let's register only our own APIs
	schema := runtime.NewScheme()
	if err := apis.AddToScheme(schema); err != nil {
		return err
	}

	for _, kit := range o.kits {
		err := errorIfKitIsNotAvailable(schema, kit)
		result = multierr.Append(result, err)
	}

	if len(o.mavenRepositories) > 0 && o.mavenSettings != "" {
		err := fmt.Errorf("incompatible options combinations: you cannot set both mavenRepository and mavenSettings")
		result = multierr.Append(result, err)
	}

	if o.traitProfile != "" {
		tp := v1alpha1.TraitProfileByName(o.traitProfile)
		if tp == v1alpha1.TraitProfile("") {
			err := fmt.Errorf("unknown trait profile %s", o.traitProfile)
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
	for _, resource := range deploy.Resources {
		resource, err := kubernetes.LoadResourceFromYaml(schema, resource)
		if err != nil {
			// Not one of our registered schemas
			continue
		}
		kind := resource.GetObjectKind().GroupVersionKind()
		if kind.Kind != "IntegrationKit" {
			continue
		}
		integrationKit := resource.(*v1alpha1.IntegrationKit)
		if integrationKit.Name == kit {
			return nil
		}
	}
	return errors.Errorf("Unknown kit '%s'", kit)
}

func decodeMavenSettings(mavenSettings string) (v1alpha1.ValueSource, error) {
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
			return v1alpha1.ValueSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: sub[1],
					},
					Key: key,
				},
			}, nil
		}
		if sub[0] == "secret" {
			return v1alpha1.ValueSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: sub[1],
					},
					Key: key,
				},
			}, nil
		}
	}

	return v1alpha1.ValueSource{}, fmt.Errorf("illegal maven setting definition, syntax: configmap|secret:resource-name[/settings path]")
}
