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
	"path/filepath"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"

	kameletutils "github.com/apache/camel-k/v2/pkg/kamelet"
	"github.com/apache/camel-k/v2/pkg/kamelet/repository"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kamelets"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
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
	if e.Integration == nil || !pointer.BoolDeref(t.Enabled, true) {
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
	}

	return nil
}

func (t *kameletsTrait) collectKamelets(e *Environment) (map[string]*v1.Kamelet, error) {
	repo, err := repository.NewForPlatform(e.Ctx, e.Client, e.Platform, e.Integration.Namespace, platform.GetOperatorNamespace())
	if err != nil {
		return nil, err
	}

	kamelets := make(map[string]*v1.Kamelet)
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
		message := fmt.Sprintf("kamelets [%s] found, [%s] not found in repositories: %s",
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
		fmt.Sprintf("kamelets [%s] found in repositories: %s", strings.Join(availableKamelets, ","), repo.String()),
	)

	return kamelets, nil
}

func (t *kameletsTrait) addKamelets(e *Environment) error {
	if len(t.getKameletKeys()) > 0 {
		kamelets, err := t.collectKamelets(e)
		if err != nil {
			return err
		}

		immutable := true
		kameletConfigmap := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("kamelets-bundle-%s", e.Integration.Name),
				Namespace: e.Integration.Namespace,
				Labels: map[string]string{
					v1.IntegrationLabel:            e.Integration.Name,
					"camel.apache.org/config.type": "kamelets-bundle",
				},
				Annotations: map[string]string{
					"camel.apache.org/generated": "true",
				},
			},
			Data:      map[string]string{},
			Immutable: &immutable,
		}

		for _, key := range t.getKameletKeys() {
			kamelet := kamelets[key]

			if kamelet.Status.Phase != v1.KameletPhaseReady {
				return fmt.Errorf("kamelet %q is not %s: %s", key, v1.KameletPhaseReady, kamelet.Status.Phase)
			}
			// Adding dependencies from Kamelets
			util.StringSliceUniqueConcat(&e.Integration.Status.Dependencies, kamelet.Spec.Dependencies)

			if err := addKamelet(kamelet, kameletConfigmap); err != nil {
				return err
			}
		}
		// resort dependencies
		sort.Strings(e.Integration.Status.Dependencies)
		// set kamelets expected directory
		e.ApplicationProperties["camel.component.kamelet.location"] = fmt.Sprintf("file:%s", filepath.Join(camel.BasePath, "kamelets"))
		e.Resources.Add(kameletConfigmap)
	}

	return nil
}

func addKamelet(kamelet *v1.Kamelet, kameletBundle *corev1.ConfigMap) error {
	serialized, err := kubernetes.ToYAMLNoManagedFields(kamelet)
	if err != nil {
		return err
	}
	kameletBundle.Data[fmt.Sprintf("%s.kamelet.yaml", kamelet.Name)] = string(serialized)

	return nil
}

// addConfigurationSecrets is used to add secrets which are required to be used by the Kamelet implicitly
// as an example
//
// cat mynamedconfig.properties
// camel.kamelet.my-company-log-sink.mynamedconfig.bucket=special
//
// kubectl create secret generic my-company-log-sink.mynamedconfig --from-file=mynamedconfig.properties
// kubectl label secret my-company-log-sink.mynamedconfig camel.apache.org/kamelet=my-company-log-sink camel.apache.org/kamelet.configuration=mynamedconfig
//
// then, this func is in charge to add such a secret to the Integration
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
		if i != "" && v1.ValidKameletName(i) {
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
