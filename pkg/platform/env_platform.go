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

package platform

import (
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/maven"
	corev1 "k8s.io/api/core/v1"
)

// Used to check runtime architecture.
var operatorArch = runtime.GOARCH

// SingletonPlatform is initialized once for performance reasons when the application starts.
var SingletonPlatform = getEnvPlatform()

// Platform contains a series of configuration required during build and packaging.
type Platform struct {
	CatalogNamespace    string
	BuildRuntimeVersion string
	BuildTimeout        time.Duration
	BuildConfiguration  v1.BuildConfiguration
	BuildBaseImage      string
	PublishStrategy     v1.IntegrationPlatformBuildPublishStrategy
	Registry            v1.RegistrySpec
	Maven               v1.MavenBuildSpec
	MaxRunningBuilds    int32
}

// getEnvPlatform is in charge to parse the environment variables of the operator and return the Platform object.
func getEnvPlatform() Platform {
	registry := registry()
	if registry.Address == "" {
		// TODO: fail fast exiting the program when we don't support IntegrationPlatform.
		log.Info("failed to initialize singleton platform from environment variables: missing mandatory env var REGISTRY_ADDRESS. " +
			"Mind that this will be required when we stop supporting IntegrationPlatform in future releases.")
	}

	return Platform{
		CatalogNamespace:    GetOperatorNamespace(),
		BuildRuntimeVersion: getEnvOrDefault("BUILD_RUNTIME_VERSION", defaults.DefaultRuntimeVersion),
		BuildTimeout:        buildTimeout(),
		BuildConfiguration: v1.BuildConfiguration{
			Strategy:       buildStrategy(),
			OrderStrategy:  orderStrategy(),
			ImagePlatforms: imagePlatforms(),
		},
		BuildBaseImage:  getEnvOrDefault("BUILD_BASE_IMAGE", defaults.BaseImage()),
		PublishStrategy: publishStrategy(),
		Registry:        registry,
		Maven: v1.MavenBuildSpec{
			MavenSpec:    mavenSpec(),
			Repositories: repositories(),
		},
		MaxRunningBuilds: maxRunningBuilds(),
	}
}

func getEnvOrDefault(key string, def string) string {
	env, exists := os.LookupEnv(key)
	if exists {
		return env
	}

	return def
}

func getEnvOrDefaultSlice(key string, def string) []string {
	env, exists := os.LookupEnv(key)
	if !exists {
		env = def
	}

	return strings.Split(env, ",")
}

func buildTimeout() time.Duration {
	buildTimeoutSeconds := getEnvOrDefault("BUILD_TIMEOUT_SECONDS", "")
	if buildTimeoutSeconds != "" {
		seconds, err := strconv.Atoi(buildTimeoutSeconds)
		if err == nil {
			return time.Duration(seconds) * time.Second
		}

		log.Error(err, "could not parse BUILD_TIMEOUT_SECONDS environment variable, fallback to default value")
	}

	return DefaultBuildTimeout
}

func buildStrategy() v1.BuildStrategy {
	buildStrategy := getEnvOrDefault("BUILD_STRATEGY", "")
	if buildStrategy != "" {
		bs := v1.BuildStrategy(buildStrategy)
		if err := bs.Validate(); err == nil {
			return bs
		}
		log.Info("BUILD_STRATEGY env var value is unsupported: " + buildStrategy + ", fallback to default")
	}

	return DefaultBuildStrategy
}

func maxRunningBuilds() int32 {
	maxRunningBuildsString := getEnvOrDefault("MAX_RUNNING_BUILDS", "")
	if maxRunningBuildsString != "" {
		val, err := strconv.ParseInt(maxRunningBuildsString, 10, 32)
		if err == nil {
			return int32(val)
		}

		log.Error(err, "could not parse MAX_RUNNING_BUILDS environment variable, fallback to default value")
	}

	if buildStrategy() == v1.BuildStrategyRoutine {
		return DefaultMaxRunningBuildsRoutineStrategy
	}

	return DefaultMaxRunningBuildsPodStrategy
}

func orderStrategy() v1.BuildOrderStrategy {
	buildOrderStrategy := getEnvOrDefault("BUILD_ORDER_STRATEGY", "")
	if buildOrderStrategy != "" {
		bs := v1.BuildOrderStrategy(buildOrderStrategy)
		if err := bs.Validate(); err == nil {
			return bs
		}
		log.Info("BUILD_ORDER_STRATEGY env var value is unsupported: " + buildOrderStrategy + ", fallback to default")
	}

	return DefaultBuildOrderStrategy
}

func publishStrategy() v1.IntegrationPlatformBuildPublishStrategy {
	publishStrategy := getEnvOrDefault("PUBLISH_STRATEGY", "")
	if publishStrategy != "" {
		ps := v1.IntegrationPlatformBuildPublishStrategy(publishStrategy)
		if err := ps.Validate(); err == nil {
			return ps
		}
		log.Info("PUBLISH_STRATEGY env var value is unsupported: " + publishStrategy + ", fallback to default")
	}

	return DefaultPublishStrategy
}

func imagePlatforms() []string {
	buildImagePlatforms := getEnvOrDefault("BUILD_IMAGE_PLATFORMS", "")
	if buildImagePlatforms != "" {
		return strings.Split(buildImagePlatforms, ",")
	}
	// Special case if we detect the operator is running in an arm64 architecture
	if operatorArch == "arm64" {
		log.Info("the operator is running on an ARM64 architecture, Integrations are going to be built as \"linux/arm64\" containers. " +
			"Use BUILD_IMAGE_PLATFORMS env var if you want to change this behavior.")

		return []string{"linux/arm64"}
	}

	return nil
}

func registry() v1.RegistrySpec {
	insecure, err := strconv.ParseBool(getEnvOrDefault("REGISTRY_INSECURE", "false"))
	if err != nil {
		insecure = false
		log.Error(err, "could not parse REGISTRY_INSECURE environment variable, fallback to true")
	}
	registry := v1.RegistrySpec{
		Insecure:     insecure,
		Address:      getEnvOrDefault("REGISTRY_ADDRESS", ""),
		Secret:       getEnvOrDefault("REGISTRY_SECRET", ""),
		CA:           getEnvOrDefault("REGISTRY_CA_CONFIGMAP", ""),
		Organization: getEnvOrDefault("REGISTRY_ORGANIZATION", ""),
	}

	return registry
}

func repositories() []v1.Repository {
	csvRepos := getEnvOrDefault("MAVEN_REPOSITORIES", "")
	if csvRepos != "" {
		parts := strings.Split(csvRepos, ",")

		repositories := make([]v1.Repository, 0, len(parts))
		for _, repo := range parts {
			repo = strings.TrimSpace(repo)
			if repo == "" {
				continue
			}
			repositories = append(repositories, maven.NewRepository(repo))
		}

		return repositories
	}

	return nil
}

func mavenSpec() v1.MavenSpec {
	settings, err := valueSource("MAVEN_SETTINGS")
	if err != nil {
		log.Error(err, "could not parse MAVEN_SETTINGS env variable, this setting will be skipped")
	}
	settingsSecurity, err := valueSource("MAVEN_SETTINGS_SECURITY")
	if err != nil {
		log.Error(err, "could not parse MAVEN_SETTINGS_SECURITY env variable, this setting will be skipped")
	}

	return v1.MavenSpec{
		LocalRepository:  "",
		Settings:         settings,
		SettingsSecurity: settingsSecurity,
		CASecrets:        caSecrets(),
		CLIOptions:       getEnvOrDefaultSlice("MAVEN_CLI_OPTIONS", DefaultMavenCLIOptions),
	}
}

// valueSource expects any var to contain <configmap|secret>:<my-name>@<my-key>.
func valueSource(envName string) (v1.ValueSource, error) {
	valueSource := v1.ValueSource{}
	vs := getEnvOrDefault(envName, "")
	if vs != "" {
		// Split kind from the rest
		parts := strings.SplitN(vs, ":", 2)
		if len(parts) != 2 {
			return valueSource, errors.New("invalid value source format: missing ':'")
		}
		kind := parts[0]

		// Split name and key
		subparts := strings.SplitN(parts[1], "@", 2)
		if len(subparts) != 2 {
			return valueSource, errors.New("invalid value source format for: missing '@'")
		}
		name := subparts[0]
		key := subparts[1]

		switch kind {
		case "configmap":
			valueSource.ConfigMapKeyRef = &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
				Key: key,
			}
		case "secret":
			valueSource.SecretKeyRef = &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
				Key: key,
			}
		default:
			return valueSource, errors.New("invalid value source format: unsupported " + kind)
		}
	}

	return valueSource, nil
}

// caSecrets expects a csv like my-secret-1@key-a,mysecret2@key2.
func caSecrets() []corev1.SecretKeySelector {
	var caSecrets []corev1.SecretKeySelector
	val := getEnvOrDefault("MAVEN_CA_SECRETS", "")
	if val != "" {
		secrets := strings.SplitSeq(val, ",")
		for secret := range secrets {
			subparts := strings.SplitN(secret, "@", 2)
			if len(subparts) != 2 {
				log.Info("could not parse MAVEN_CA_SECRETS env variable, \"" + secret + "\" will be skipped")

				continue
			}
			caSecrets = append(caSecrets, corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: subparts[0],
				},
				Key: subparts[1],
			})
		}
	}

	return caSecrets
}

func FromIntegrationPlatform(itp *v1.IntegrationPlatform) Platform {
	return Platform{
		CatalogNamespace:    itp.GetNamespace(),
		BuildRuntimeVersion: itp.Status.Build.RuntimeVersion,
		BuildTimeout:        itp.Status.Build.GetTimeout().Duration,
		BuildConfiguration:  itp.Status.Build.BuildConfiguration,
		BuildBaseImage:      itp.Status.Build.BaseImage,
		PublishStrategy:     itp.Status.Build.PublishStrategy,
		Registry:            itp.Status.Build.Registry,
		Maven: v1.MavenBuildSpec{
			MavenSpec: itp.Status.Build.Maven,
		},
		MaxRunningBuilds: itp.Status.Build.MaxRunningBuilds,
	}
}
