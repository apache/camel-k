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

	return t.List != "", nil
}

func (t *kameletsTrait) Apply(e *Environment) error {

	return nil
}

// IsPlatformTrait overrides base class method
func (t *kameletsTrait) IsPlatformTrait() bool {
	return true
}

func (t *kameletsTrait) addKameletAsSource(e *Environment, kamelet *v1alpha1.Kamelet) error {
	var sources []v1.SourceSpec

	flowData, err := flows.Marshal([]v1.Flow{kamelet.Spec.Flow})
	if err != nil {
		return err
	}
	flowSource := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name:    "flow.yaml",
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

	for idx, s := range kamelet.Spec.Sources {
		intSource, err := integrationSourceFromKameletSource(e, kamelet, s, fmt.Sprintf("%s-kamelet-%s-source-%03d", e.Integration.Name, kamelet.Name, idx))
		if err != nil {
			return err
		}
		sources = append(sources, intSource)
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

func integrationSourceFromKameletSource(e *Environment, kamelet *v1alpha1.Kamelet, source v1.SourceSpec, name string) (v1.SourceSpec, error) {
	if source.DataSpec.ContentRef != "" {
		return renameSource(kamelet, source), nil
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

	target := renameSource(kamelet, source)
	target.Content = ""
	target.ContentRef = name
	target.ContentKey = "content"
	return target, nil
}

func renameSource(kamelet *v1alpha1.Kamelet, source v1.SourceSpec) v1.SourceSpec {
	target := source.DeepCopy()
	if !strings.HasPrefix(target.Name, fmt.Sprintf("kamelet-%s-", kamelet.Name)) {
		target.Name = fmt.Sprintf("kamelet-%s-%s", kamelet.Name, target.Name)
	}
	return *target
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
