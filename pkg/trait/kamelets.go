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
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/apache/camel-k/pkg/util/flows"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
	"strconv"
	"strings"
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

var (
	kameletNameRegexp = regexp.MustCompile("kamelet:(?://)?([a-z0-9-.]+)(?:$|[^a-z0-9-.].*)")
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

	if t.Auto == nil || *t.Auto {
		if t.List == "" {
			var kamelets []string
			metadata.Each(e.CamelCatalog, e.Integration.Sources(), func(_ int, meta metadata.IntegrationMetadata) bool {
				util.StringSliceUniqueConcat(&kamelets, extractKamelets(meta.FromURIs))
				util.StringSliceUniqueConcat(&kamelets, extractKamelets(meta.ToURIs))
				return true
			})
			sort.Strings(kamelets)
			t.List = strings.Join(kamelets, ",")
		}

	}

	return len(t.getKamelets()) > 0, nil
}

func (t *kameletsTrait) Apply(e *Environment) error {
	if err := t.addKamelets(e); err != nil {
		return err
	}
	return nil
}

func (t *kameletsTrait) addKamelets(e *Environment) error {
	for _, k := range t.getKamelets() {
		var kamelet v1alpha1.Kamelet
		key := client.ObjectKey{
			Namespace: e.Integration.Namespace,
			Name:      k,
		}
		if err := t.Client.Get(t.Ctx, key, &kamelet); err != nil {
			return err
		}
		if err := t.addKameletAsSource(e, kamelet); err != nil {
			return err
		}
	}
	return nil
}

func (t *kameletsTrait) addKameletAsSource(e *Environment, kamelet v1alpha1.Kamelet) error {
	var sources []v1.SourceSpec

	if kamelet.Spec.Flow != nil {
		flowData, err := flows.Marshal([]v1.Flow{*kamelet.Spec.Flow})
		if err != nil {
			return err
		}
		flowSource := v1.SourceSpec{
			DataSpec: v1.DataSpec{
				Name:    fmt.Sprintf("%s.yaml", kamelet.Name),
				Content: string(flowData),
			},
			Language: v1.LanguageYaml,
			Type:     v1.SourceTypeKamelet,
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
		if source.Type == v1.SourceTypeKamelet {
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

func (t *kameletsTrait) getKamelets() []string {
	answer := make([]string, 0)
	for _, item := range strings.Split(t.List, ",") {
		i := strings.Trim(item, " \t\"")
		if i != "" {
			answer = append(answer, i)
		}
	}
	return answer
}

func integrationSourceFromKameletSource(e *Environment, kamelet v1alpha1.Kamelet, source v1.SourceSpec, name string) (v1.SourceSpec, error) {
	if source.Type == v1.SourceTypeKamelet {
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
			"content": source.Content,
		},
	}

	e.Resources.Add(&cm)

	target := source.DeepCopy()
	target.Content = ""
	target.ContentRef = name
	target.ContentKey = "content"
	return *target, nil
}

func extractKamelets(uris []string) (kamelets []string) {
	for _, uri := range uris {
		matches := kameletNameRegexp.FindStringSubmatch(uri)
		if len(matches) == 2 {
			kamelets = append(kamelets, matches[1])
		}
	}
	return
}
