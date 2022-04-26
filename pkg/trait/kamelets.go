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
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	kameletutils "github.com/apache/camel-k/pkg/kamelet"
	"github.com/apache/camel-k/pkg/kamelet/repository"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/apache/camel-k/pkg/util/dsl"
	"github.com/apache/camel-k/pkg/util/kamelets"
)

type kameletsTrait struct {
	BaseTrait
	traitv1.KameletsTrait `property:",squash"`
}

type configurationKey struct {
	kamelet         string
	configurationID string
}

func newConfigurationKey(kamelet, configurationID string) configurationKey {
	return configurationKey{
		kamelet:         kamelet,
		configurationID: configurationID,
	}
}

const (
	contentKey = "content"

	kameletLabel              = "camel.apache.org/kamelet"
	kameletConfigurationLabel = "camel.apache.org/kamelet.configuration"
)

func newKameletsTrait() Trait {
	return &kameletsTrait{
		BaseTrait: NewBaseTrait("kamelets", 450),
	}
}

// IsPlatformTrait overrides base class method.
func (t *kameletsTrait) IsPlatformTrait() bool {
	return true
}

func (t *kameletsTrait) Configure(e *Environment) (bool, error) {
	if !pointer.BoolDeref(t.Enabled, true) {
		return false, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !e.IntegrationInRunningPhases() {
		return false, nil
	}

	if pointer.BoolDeref(t.Auto, true) {
		kamelets, err := kamelets.ExtractKameletFromSources(e.Ctx, e.Client, e.CamelCatalog, e.Resources, e.Integration)
		if err != nil {
			return false, err
		}

		if len(kamelets) > 0 {
			sort.Strings(kamelets)
			t.List = strings.Join(kamelets, ",")
		}
	}

	return len(t.getKameletKeys()) > 0, nil
}

func (t *kameletsTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) || e.IntegrationInRunningPhases() {
		if err := t.addKamelets(e); err != nil {
			return err
		}
	}
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		return t.addConfigurationSecrets(e)
	} else if e.IntegrationInRunningPhases() {
		return t.configureApplicationProperties(e)
	}

	return nil
}

func (t *kameletsTrait) collectKamelets(e *Environment) (map[string]*v1alpha1.Kamelet, error) {
	repo, err := repository.NewForPlatform(e.Ctx, e.Client, e.Platform, e.Integration.Namespace, platform.GetOperatorNamespace())
	if err != nil {
		return nil, err
	}

	kamelets := make(map[string]*v1alpha1.Kamelet)
	missingKamelets := make([]string, 0)
	availableKamelets := make([]string, 0)

	for _, key := range t.getKameletKeys() {
		kamelet, err := repo.Get(e.Ctx, key)
		if err != nil {
			return nil, err
		}

		if kamelet == nil {
			missingKamelets = append(missingKamelets, key)
		} else {
			availableKamelets = append(availableKamelets, key)

			// Initialize remote kamelets
			kamelets[key], err = kameletutils.Initialize(kamelet)
			if err != nil {
				return nil, err
			}
		}
	}

	sort.Strings(availableKamelets)
	sort.Strings(missingKamelets)

	if len(missingKamelets) > 0 {
		message := fmt.Sprintf("kamelets %s found, %s not found in repositories: %s",
			strings.Join(availableKamelets, ","),
			strings.Join(missingKamelets, ","),
			repo.String())

		e.Integration.Status.SetCondition(
			v1.IntegrationConditionKameletsAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionKameletsAvailableReason,
			message,
		)

		return nil, errors.New(message)
	}

	e.Integration.Status.SetCondition(
		v1.IntegrationConditionKameletsAvailable,
		corev1.ConditionTrue,
		v1.IntegrationConditionKameletsAvailableReason,
		fmt.Sprintf("kamelets %s found in repositories: %s", strings.Join(availableKamelets, ","), repo.String()),
	)

	return kamelets, nil
}

func (t *kameletsTrait) addKamelets(e *Environment) error {
	if len(t.getKameletKeys()) > 0 {
		kamelets, err := t.collectKamelets(e)
		if err != nil {
			return err
		}

		for _, key := range t.getKameletKeys() {
			kamelet := kamelets[key]

			if kamelet.Status.Phase != v1alpha1.KameletPhaseReady {
				return fmt.Errorf("kamelet %q is not %s: %s", key, v1alpha1.KameletPhaseReady, kamelet.Status.Phase)
			}

			if err := t.addKameletAsSource(e, kamelet); err != nil {
				return err
			}

			// Adding dependencies from Kamelets
			util.StringSliceUniqueConcat(&e.Integration.Status.Dependencies, kamelet.Spec.Dependencies)
		}
		// resort dependencies
		sort.Strings(e.Integration.Status.Dependencies)
	}
	return nil
}

func (t *kameletsTrait) configureApplicationProperties(e *Environment) error {
	if len(t.getKameletKeys()) > 0 {
		kamelets, err := t.collectKamelets(e)
		if err != nil {
			return err
		}

		for _, key := range t.getKameletKeys() {
			kamelet := kamelets[key]
			// Configuring defaults from Kamelet
			for _, prop := range kamelet.Status.Properties {
				if prop.Default != "" {
					// Check whether user specified a value
					userDefined := false
					propName := fmt.Sprintf("camel.kamelet.%s.%s", kamelet.Name, prop.Name)
					propPrefix := propName + "="
					for _, userProp := range e.Integration.Spec.Configuration {
						if strings.HasPrefix(userProp.Value, propPrefix) {
							userDefined = true
							break
						}
					}
					if !userDefined {
						e.ApplicationProperties[propName] = prop.Default
					}
				}
			}
		}
	}
	return nil
}

func (t *kameletsTrait) addKameletAsSource(e *Environment, kamelet *v1alpha1.Kamelet) error {
	sources := make([]v1.SourceSpec, 0)

	// nolint: staticcheck
	if kamelet.Spec.Template != nil || kamelet.Spec.Flow != nil {
		template := kamelet.Spec.Template
		if template == nil {
			// Backward compatibility with Kamelets using flow
			var bytes []byte = kamelet.Spec.Flow.RawMessage
			template = &v1alpha1.Template{
				RawMessage: make(v1alpha1.RawMessage, len(bytes)),
			}
			copy(template.RawMessage, bytes)
		}
		flowData, err := dsl.TemplateToYamlDSL(*template, kamelet.Name)
		if err != nil {
			return err
		}

		flowSource := v1.SourceSpec{
			DataSpec: v1.DataSpec{
				Name:    fmt.Sprintf("%s.yaml", kamelet.Name),
				Content: string(flowData),
			},
			Language: v1.LanguageYaml,
		}
		flowSource, err = integrationSourceFromKameletSource(e, kamelet, flowSource, fmt.Sprintf("%s-kamelet-%s-template", e.Integration.Name, kamelet.Name))
		if err != nil {
			return err
		}
		sources = append(sources, flowSource)
	}

	for idx, s := range kamelet.Spec.Sources {
		intSource, err := integrationSourceFromKameletSource(e, kamelet, s, fmt.Sprintf("%s-kamelet-%s-%03d", e.Integration.Name, kamelet.Name, idx))
		if err != nil {
			return err
		}
		sources = append(sources, intSource)
	}

	for _, source := range sources {
		replaced := false
		for idx, existing := range e.Integration.Status.GeneratedSources {
			if existing.Name == source.Name {
				replaced = true
				e.Integration.Status.GeneratedSources[idx] = source
			}
		}
		if !replaced {
			e.Integration.Status.GeneratedSources = append(e.Integration.Status.GeneratedSources, source)
		}
	}

	return nil
}

func (t *kameletsTrait) addConfigurationSecrets(e *Environment) error {
	for _, k := range t.getConfigurationKeys() {
		options := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", kameletLabel, k.kamelet),
		}
		if k.configurationID != "" {
			options.LabelSelector = fmt.Sprintf("%s=%s,%s=%s", kameletLabel, k.kamelet, kameletConfigurationLabel, k.configurationID)
		}
		secrets, err := t.Client.CoreV1().Secrets(e.Integration.Namespace).List(e.Ctx, options)
		if err != nil {
			return err
		}

		for _, item := range secrets.Items {
			if item.Labels != nil && item.Labels[kameletConfigurationLabel] != k.configurationID {
				continue
			}

			e.Integration.Status.AddConfigurationsIfMissing(v1.ConfigurationSpec{
				Type:  "secret",
				Value: item.Name,
			})
		}
	}
	return nil
}

func (t *kameletsTrait) getKameletKeys() []string {
	answer := make([]string, 0)
	for _, item := range strings.Split(t.List, ",") {
		i := strings.Trim(item, " \t\"")
		if strings.Contains(i, "/") {
			i = strings.SplitN(i, "/", 2)[0]
		}
		if i != "" && v1alpha1.ValidKameletName(i) {
			util.StringSliceUniqueAdd(&answer, i)
		}
	}
	sort.Strings(answer)
	return answer
}

func (t *kameletsTrait) getConfigurationKeys() []configurationKey {
	answer := make([]configurationKey, 0)
	for _, item := range t.getKameletKeys() {
		answer = append(answer, newConfigurationKey(item, ""))
	}
	for _, item := range strings.Split(t.List, ",") {
		i := strings.Trim(item, " \t\"")
		if strings.Contains(i, "/") {
			parts := strings.SplitN(i, "/", 2)
			newKey := newConfigurationKey(parts[0], parts[1])
			alreadyPresent := false
			for _, existing := range answer {
				if existing == newKey {
					alreadyPresent = true
					break
				}
			}
			if !alreadyPresent {
				answer = append(answer, newKey)
			}
		}
	}
	sort.Slice(answer, func(i, j int) bool {
		o1 := answer[i]
		o2 := answer[j]
		return o1.kamelet < o2.kamelet || (o1.kamelet == o2.kamelet && o1.configurationID < o2.configurationID)
	})
	return answer
}

func integrationSourceFromKameletSource(e *Environment, kamelet *v1alpha1.Kamelet, source v1.SourceSpec, name string) (v1.SourceSpec, error) {
	if source.Type == v1.SourceTypeTemplate {
		// Kamelets must be named "<kamelet-name>.extension"
		language := source.InferLanguage()
		source.Name = fmt.Sprintf("%s.%s", kamelet.Name, string(language))
	}

	if source.DataSpec.ContentRef != "" {
		return source, nil
	}

	// Create configmaps to avoid storing kamelet definitions in the integration CR
	// Compute the input digest and store it along with the configmap
	hash, err := digest.ComputeForSource(source)
	if err != nil {
		return v1.SourceSpec{}, err
	}

	cm := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e.Integration.Namespace,
			Labels: map[string]string{
				"camel.apache.org/integration": e.Integration.Name,
				"camel.apache.org/kamelet":     kamelet.Name,
			},
			Annotations: map[string]string{
				"camel.apache.org/source.language":    string(source.Language),
				"camel.apache.org/source.name":        name,
				"camel.apache.org/source.compression": strconv.FormatBool(source.Compression),
				"camel.apache.org/source.generated":   "true",
				"camel.apache.org/source.type":        string(source.Type),
				"camel.apache.org/source.digest":      hash,
			},
		},
		Data: map[string]string{
			contentKey: source.Content,
		},
	}

	e.Resources.Add(&cm)

	target := source.DeepCopy()
	target.Content = ""
	target.ContentRef = name
	target.ContentKey = contentKey
	return *target, nil
}
