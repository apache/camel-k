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

package trait

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/rs/xid"

	corev1 "k8s.io/api/core/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

type quarkusPackageType string

const (
	quarkusTraitId = "quarkus"

	fastJarPackageType quarkusPackageType = "fast-jar"
	nativePackageType  quarkusPackageType = "native"
)

var kitPriority = map[quarkusPackageType]string{
	fastJarPackageType: "1000",
	nativePackageType:  "2000",
}

// The Quarkus trait configures the Quarkus runtime.
//
// It's enabled by default.
//
// NOTE: Compiling to a native executable, i.e. when using `package-type=native`, is only supported
// for kamelets, as well as YAML and XML integrations.
// It also requires at least 4GiB of memory, so the Pod running the native build, that is either
// the operator Pod, or the build Pod (depending on the build strategy configured for the platform),
// must have enough memory available.
//
// +camel-k:trait=quarkus
type quarkusTrait struct {
	BaseTrait `property:",squash"`
	// The Quarkus package types, either `fast-jar` or `native` (default `fast-jar`).
	// In case both `fast-jar` and `native` are specified, two `IntegrationKit` resources are created,
	// with the `native` kit having precedence over the `fast-jar` one once ready.
	// The order influences the resolution of the current kit for the integration.
	// The kit corresponding to the first package type will be assigned to the
	// integration in case no existing kit that matches the integration exists.
	PackageTypes []quarkusPackageType `property:"package-type" json:"packageTypes,omitempty"`
}

func newQuarkusTrait() Trait {
	return &quarkusTrait{
		BaseTrait: NewBaseTrait(quarkusTraitId, 1700),
	}
}

// IsPlatformTrait overrides base class method
func (t *quarkusTrait) IsPlatformTrait() bool {
	return true
}

// InfluencesKit overrides base class method
func (t *quarkusTrait) InfluencesKit() bool {
	return true
}

var _ ComparableTrait = &quarkusTrait{}

func (t *quarkusTrait) Matches(trait Trait) bool {
	qt, ok := trait.(*quarkusTrait)
	if !ok {
		return false
	}

	if IsNilOrTrue(t.Enabled) && IsFalse(qt.Enabled) {
		return false
	}

	if len(t.PackageTypes) == 0 && len(qt.PackageTypes) != 0 && !containsPackageType(qt.PackageTypes, fastJarPackageType) {
		return false
	}

types:
	for _, pt := range t.PackageTypes {
		if pt == fastJarPackageType && len(qt.PackageTypes) == 0 {
			continue
		}
		if containsPackageType(qt.PackageTypes, pt) {
			continue types
		}
		return false
	}

	return true
}

func (t *quarkusTrait) Configure(e *Environment) (bool, error) {
	if IsFalse(t.Enabled) {
		return false, nil
	}

	return e.IntegrationInPhase(v1.IntegrationPhaseBuildingKit) ||
			e.IntegrationKitInPhase(v1.IntegrationKitPhaseBuildSubmitted) ||
			e.IntegrationKitInPhase(v1.IntegrationKitPhaseReady) && e.IntegrationInRunningPhases(),
		nil
}

func (t *quarkusTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseBuildingKit) {
		if containsPackageType(t.PackageTypes, nativePackageType) {
			// Native compilation is only supported for a subset of languages,
			// so let's check for compatibility, and fail-fast the Integration,
			// to save compute resources and user time.
			for _, source := range e.Integration.Sources() {
				if language := source.InferLanguage(); language != v1.LanguageKamelet &&
					language != v1.LanguageYaml &&
					language != v1.LanguageXML {
					t.L.ForIntegration(e.Integration).Infof("Integration %s contains a %s source that cannot be compiled to native executable", e.Integration.Namespace+"/"+e.Integration.Name, language)
					e.Integration.Status.Phase = v1.IntegrationPhaseError
					e.Integration.Status.SetCondition(
						v1.IntegrationConditionKitAvailable,
						corev1.ConditionFalse,
						v1.IntegrationConditionUnsupportedLanguageReason,
						fmt.Sprintf("native compilation for language %q is not supported", language))
					// Let the calling controller handle the Integration update
					return nil
				}
			}
		}

		switch len(t.PackageTypes) {
		case 0:
			kit := t.newIntegrationKit(e, fastJarPackageType)
			e.IntegrationKits = append(e.IntegrationKits, *kit)

		case 1:
			kit := t.newIntegrationKit(e, t.PackageTypes[0])
			e.IntegrationKits = append(e.IntegrationKits, *kit)

		default:
			for _, packageType := range t.PackageTypes {
				kit := t.newIntegrationKit(e, packageType)
				data, err := json.Marshal(kit.Spec.Traits[quarkusTraitId].Configuration)
				if err != nil {
					return err
				}
				trait := quarkusTrait{}
				err = json.Unmarshal(data, &trait)
				if err != nil {
					return err
				}
				trait.PackageTypes = []quarkusPackageType{packageType}
				data, err = json.Marshal(trait)
				if err != nil {
					return err
				}
				kit.Spec.Traits[quarkusTraitId] = v1.TraitSpec{
					Configuration: v1.TraitConfiguration{
						RawMessage: data,
					},
				}
				e.IntegrationKits = append(e.IntegrationKits, *kit)
			}
		}

		return nil
	}

	switch e.IntegrationKit.Status.Phase {

	case v1.IntegrationKitPhaseBuildSubmitted:
		build := getBuilderTask(e.BuildTasks)
		if build == nil {
			return fmt.Errorf("unable to find builder task: %s", e.Integration.Name)
		}

		if build.Maven.Properties == nil {
			build.Maven.Properties = make(map[string]string)
		}

		steps, err := builder.StepsFrom(build.Steps...)
		if err != nil {
			return err
		}

		steps = append(steps, builder.Quarkus.CommonSteps...)

		if native, err := t.isNativeKit(e); err != nil {
			return err
		} else if native {
			build.Maven.Properties["quarkus.package.type"] = string(nativePackageType)
			steps = append(steps, builder.Image.NativeImageContext)
			// Spectrum does not rely on Dockerfile to assemble the image
			if e.Platform.Status.Build.PublishStrategy != v1.IntegrationPlatformBuildPublishStrategySpectrum {
				steps = append(steps, builder.Image.ExecutableDockerfile)
			}
		} else {
			build.Maven.Properties["quarkus.package.type"] = string(fastJarPackageType)
			steps = append(steps, builder.Quarkus.ComputeQuarkusDependencies, builder.Image.IncrementalImageContext)
			// Spectrum does not rely on Dockerfile to assemble the image
			if e.Platform.Status.Build.PublishStrategy != v1.IntegrationPlatformBuildPublishStrategySpectrum {
				steps = append(steps, builder.Image.JvmDockerfile)
			}
		}

		// Sort steps by phase
		sort.SliceStable(steps, func(i, j int) bool {
			return steps[i].Phase() < steps[j].Phase()
		})

		build.Steps = builder.StepIDsFor(steps...)

	case v1.IntegrationKitPhaseReady:
		if e.IntegrationInRunningPhases() && t.isNativeIntegration(e) {
			container := e.getIntegrationContainer()
			if container == nil {
				return fmt.Errorf("unable to find integration container: %s", e.Integration.Name)
			}

			container.Command = []string{"./camel-k-integration-" + defaults.Version + "-runner"}
			container.WorkingDir = builder.DeploymentDir
		}
	}

	return nil
}

func (t *quarkusTrait) newIntegrationKit(e *Environment, packageType quarkusPackageType) *v1.IntegrationKit {
	integration := e.Integration
	kit := v1.NewIntegrationKit(integration.GetIntegrationKitNamespace(e.Platform), fmt.Sprintf("kit-%s", xid.New()))

	kit.Labels = map[string]string{
		v1.IntegrationKitTypeLabel:            v1.IntegrationKitTypePlatform,
		"camel.apache.org/runtime.version":    integration.Status.RuntimeVersion,
		"camel.apache.org/runtime.provider":   string(integration.Status.RuntimeProvider),
		v1.IntegrationKitLayoutLabel:          string(packageType),
		v1.IntegrationKitPriorityLabel:        kitPriority[packageType],
		kubernetes.CamelCreatorLabelKind:      v1.IntegrationKind,
		kubernetes.CamelCreatorLabelName:      integration.Name,
		kubernetes.CamelCreatorLabelNamespace: integration.Namespace,
		kubernetes.CamelCreatorLabelVersion:   integration.ResourceVersion,
	}

	traits := t.getKitTraits(e)

	kit.Spec = v1.IntegrationKitSpec{
		Dependencies: e.Integration.Status.Dependencies,
		Repositories: e.Integration.Spec.Repositories,
		Traits:       traits,
	}

	return kit
}

func (t *quarkusTrait) getKitTraits(e *Environment) map[string]v1.TraitSpec {
	traits := make(map[string]v1.TraitSpec)
	for name, spec := range e.Integration.Spec.Traits {
		t := e.Catalog.GetTrait(name)
		if t != nil && !t.InfluencesKit() {
			continue
		}
		traits[name] = spec
	}
	return traits
}

func (t *quarkusTrait) isNativeKit(e *Environment) (bool, error) {
	switch types := t.PackageTypes; len(types) {
	case 0:
		return false, nil
	case 1:
		return types[0] == nativePackageType, nil
	default:
		return false, fmt.Errorf("kit %q has more than one package type", e.IntegrationKit.Name)
	}
}

func (t *quarkusTrait) isNativeIntegration(e *Environment) bool {
	// The current IntegrationKit determines the Integration runtime type
	return e.IntegrationKit.Labels[v1.IntegrationKitLayoutLabel] == v1.IntegrationKitLayoutNative
}

func getBuilderTask(tasks []v1.Task) *v1.BuilderTask {
	for i, task := range tasks {
		if task.Builder != nil {
			return tasks[i].Builder
		}
	}
	return nil
}

func containsPackageType(types []quarkusPackageType, t quarkusPackageType) bool {
	for _, ti := range types {
		if t == ti {
			return true
		}
	}
	return false
}
