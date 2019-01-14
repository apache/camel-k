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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/apache/camel-k/version"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/gzip"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/maven"
)

type restTrait struct {
	BaseTrait `property:",squash"`
}

func newRestTrait() *restTrait {
	return &restTrait{
		BaseTrait: BaseTrait{
			id: ID("rest"),
		},
	}
}

func (t *restTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	return e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying), nil
}

func (t *restTrait) Apply(e *Environment) error {
	if len(e.Integration.Spec.Resources) == 0 {
		return nil
	}

	root := os.TempDir()
	tmpDir, err := ioutil.TempDir(root, "rest-dsl")
	if err != nil {
		return err
	}

	defer os.RemoveAll(tmpDir)

	for i, resource := range e.Integration.Spec.Resources {
		if resource.Type != v1alpha1.ResourceTypeOpenAPI {
			continue
		}

		content := []byte(resource.Content)
		if resource.Compression {
			content, err = gzip.UncompressBase64(content)
			if err != nil {
				return err
			}
		}

		in := path.Join(tmpDir, "openapi-spec.json")
		out := path.Join(tmpDir, "openapi-dsl.xml")

		if err := ioutil.WriteFile(in, content, 0644); err != nil {
			return err
		}

		goal := fmt.Sprintf("org.apache.camel.k:camel-k-maven-plugin:%s:generate-rest-xml", version.Version)
		iArg := "-Dopenapi.spec=" + in
		oArg := "-Ddsl.out=" + out

		if err := maven.Run(tmpDir, maven.ExtraOptions(), goal, iArg, oArg); err != nil {
			return err
		}

		content, err := ioutil.ReadFile(out)
		if err != nil {
			return err
		}

		if resource.Compression {
			c, err := gzip.CompressBase64(content)
			if err != nil {
				return nil
			}

			content = c
		}

		name := fmt.Sprintf("%s-openapi-%03d", e.Integration.Name, i)

		//
		// Store the generated rest xml in a separate config map in order
		// not to pollute the integration with generated data
		//
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
				},
				Annotations: map[string]string{
					"camel.apache.org/source.language":    string(v1alpha1.LanguageXML),
					"camel.apache.org/source.name":        resource.Name,
					"camel.apache.org/source.compression": strconv.FormatBool(resource.Compression),
					"camel.apache.org/source.generated":   "true",
				},
			},
			Data: map[string]string{
				"content": string(content),
			},
		}

		//
		// Add an additional source that references the previously
		// created config map
		//
		e.Integration.Status.GeneratedSources = append(e.Integration.Status.GeneratedSources, v1alpha1.SourceSpec{
			DataSpec: v1alpha1.DataSpec{
				Name:        strings.TrimSuffix(resource.Name, filepath.Ext(resource.Name)) + ".xml",
				ContentRef:  name,
				Compression: resource.Compression,
			},
			Language: v1alpha1.LanguageXML,
		})

		e.Resources.Add(&cm)
	}

	return nil
}
