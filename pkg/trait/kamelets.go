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
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	kameletutils "github.com/apache/camel-k/pkg/kamelet"
	"github.com/apache/camel-k/pkg/kamelet/repository"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/apache/camel-k/pkg/util/flow"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The kamelets trait is a platform trait used to inject Kamelets into the integration runtime.
//
// +camel-k:trait=kamelets
type kameletsTrait struct {
	BaseTrait `property:",squash"`
	// Automatically inject all referenced Kamelets and their default configuration (enabled by default)
	Auto *bool `property:"auto"`
	// Comma separated list of Kamelet names to load into the current integration
	List string `property:"list"`
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
	schemaKey  = "schema"

	kameletLabel              = "camel.apache.org/kamelet"
	kameletConfigurationLabel = "camel.apache.org/kamelet.configuration"
)

var (
	kameletNameRegexp = regexp.MustCompile("kamelet:(?://)?([a-z0-9-.]+(/[a-z0-9-.]+)?)(?:$|[^a-z0-9-.].*)")
)

func newKameletsTrait() Trait {
	return &kameletsTrait{
		BaseTrait: NewBaseTrait("kamelets", 450),
	}
}

// IsPlatformTrait overrides base class method
func (t *kameletsTrait) IsPlatformTrait() bool {
	return true
}

func (t *kameletsTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization, v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning) {
		return false, nil
	}

	if t.Auto == nil || *t.Auto {
		if t.List == "" {
			var kamelets []string
			sources, err := kubernetes.ResolveIntegrationSources(e.C, e.Client, e.Integration, e.Resources)
			if err != nil {
				return false, err
			}
			metadata.Each(e.CamelCatalog, sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				util.StringSliceUniqueConcat(&kamelets, extractKamelets(meta.FromURIs))
				util.StringSliceUniqueConcat(&kamelets, extractKamelets(meta.ToURIs))
				return true
			})
			sort.Strings(kamelets)
			t.List = strings.Join(kamelets, ",")
		}

	}

	return len(t.getKameletKeys()) > 0, nil
}

func (t *kameletsTrait) Apply(e *Environment) error {

	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		if err := t.addKamelets(e); err != nil {
			return err
		}
		if err := t.addConfigurationSecrets(e); err != nil {
			return err
		}
	} else if e.IntegrationInPhase(v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning) {
		return t.configureApplicationProperties(e)
	}
	return nil
}

func (t *kameletsTrait) addKamelets(e *Environment) error {
	kameletKeys := t.getKameletKeys()
	if len(kameletKeys) > 0 {
		repo, err := repository.NewForPlatform(e.C, e.Client, e.Platform, e.Integration.Namespace, platform.GetOperatorNamespace())
		if err != nil {
			return err
		}
		for _, k := range t.getKameletKeys() {
			kamelet, err := repo.Get(e.C, k)
			if err != nil {
				return err
			}
			if kamelet == nil {
				return fmt.Errorf("kamelet %s not found in any of the defined repositories: %s", k, repo.String())
			}

			// Initialize remote kamelets
			kamelet, err = kameletutils.Initialize(kamelet)
			if err != nil {
				return err
			}

			if kamelet.Status.Phase != v1alpha1.KameletPhaseReady {
				return fmt.Errorf("kamelet %q is not %s: %s", k, v1alpha1.KameletPhaseReady, kamelet.Status.Phase)
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
		repo, err := repository.NewForPlatform(e.C, e.Client, e.Platform, e.Integration.Namespace, platform.GetOperatorNamespace())
		if err != nil {
			return err
		}
		for _, k := range t.getKameletKeys() {
			kamelet, err := repo.Get(e.C, k)
			if err != nil {
				return err
			}
			if kamelet == nil {
				return fmt.Errorf("kamelet %s not found in any of the defined repositories: %s", k, repo.String())
			}

			// remote Kamelets may not be fully initialized
			kamelet, err = kameletutils.Initialize(kamelet)
			if err != nil {
				return err
			}

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

	if kamelet.Spec.Flow != nil {

		flowData, err := flow.ToYamlDSL([]v1.Flow{*kamelet.Spec.Flow})
		if err != nil {
			return err
		}

		propertyNames := make([]string, 0, len(kamelet.Status.Properties))
		for _, p := range kamelet.Status.Properties {
			propertyNames = append(propertyNames, p.Name)
		}

		flowSource := v1.SourceSpec{
			DataSpec: v1.DataSpec{
				Name:    fmt.Sprintf("%s.yaml", kamelet.Name),
				Content: string(flowData),
			},
			Language:      v1.LanguageYaml,
			Type:          v1.SourceTypeTemplate,
			PropertyNames: propertyNames,
		}
		flowSource, err = integrationSourceFromKameletSource(e, kamelet, flowSource, fmt.Sprintf("%s-kamelet-%s-flow", e.Integration.Name, kamelet.Name))
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

	kameletCounter := 0
	for _, source := range sources {
		if source.Type == v1.SourceTypeTemplate {
			kameletCounter++
		}
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

	if kameletCounter > 1 {
		return fmt.Errorf(`kamelet %s contains %d sources of type "kamelet": at most one is allowed`, kamelet.Name, kameletCounter)
	}

	return nil
}

func (t *kameletsTrait) addConfigurationSecrets(e *Environment) error {
	for _, k := range t.getConfigurationKeys() {
		var options = metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", kameletLabel, k.kamelet),
		}
		if k.configurationID != "" {
			options.LabelSelector = fmt.Sprintf("%s=%s,%s=%s", kameletLabel, k.kamelet, kameletConfigurationLabel, k.configurationID)
		}
		secrets, err := t.Client.CoreV1().Secrets(e.Integration.Namespace).List(e.C, options)
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

func extractKamelets(uris []string) (kamelets []string) {
	for _, uri := range uris {
		matches := kameletNameRegexp.FindStringSubmatch(uri)
		if len(matches) > 1 {
			kamelets = append(kamelets, matches[1])
		}
	}
	return
}
