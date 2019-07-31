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
	"encoding/xml"
	"errors"
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

type restDslTrait struct {
	BaseTrait `property:",squash"`
}

func newRestDslTrait() *restDslTrait {
	return &restDslTrait{
		BaseTrait: newBaseTrait("rest-dsl"),
	}
}

func (t *restDslTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if e.Integration == nil {
		return false, nil
	}

	for _, resource := range e.Integration.Spec.Resources {
		if resource.Type == v1alpha1.ResourceTypeOpenAPI {
			return e.IntegrationInPhase(""), nil
		}
	}

	return false, nil
}

func (t *restDslTrait) Apply(e *Environment) error {
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

		tmpDir = path.Join(tmpDir, strconv.Itoa(i))
		err := os.MkdirAll(tmpDir, os.ModePerm)
		if err != nil {
			return err
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

		err = ioutil.WriteFile(in, content, 0644)
		if err != nil {
			return err
		}

		opts := make([]string, 0, 4)
		opts = append(opts, maven.ExtraOptions(e.Platform.Spec.Build.LocalRepository)...)
		opts = append(opts, "-Dopenapi.spec="+in)
		opts = append(opts, "-Ddsl.out="+out)

		project, err := t.generateProject(e)
		if err != nil {
			return err
		}

		err = maven.CreateStructure(tmpDir, project)
		if err != nil {
			return err
		}

		err = maven.Run(tmpDir, opts...)
		if err != nil {
			return err
		}

		content, err = ioutil.ReadFile(out)
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

		generatedContentName := fmt.Sprintf("%s-openapi-%03d", e.Integration.Name, i)
		generatedSourceName := strings.TrimSuffix(resource.Name, filepath.Ext(resource.Name)) + ".xml"
		generatedSources := make([]v1alpha1.SourceSpec, 0, len(e.Integration.Status.GeneratedSources))

		if e.Integration.Status.GeneratedSources != nil {
			//
			// Filter out the previously generated source
			//
			for _, x := range e.Integration.Status.GeneratedSources {
				if x.Name != generatedSourceName {
					generatedSources = append(generatedSources, x)
				}
			}
		}

		//
		// Add an additional source that references the config map
		//
		generatedSources = append(generatedSources, v1alpha1.SourceSpec{
			DataSpec: v1alpha1.DataSpec{
				Name:        generatedSourceName,
				ContentRef:  generatedContentName,
				Compression: resource.Compression,
			},
			Language: v1alpha1.LanguageXML,
		})

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
				Name:      generatedContentName,
				Namespace: e.Integration.Namespace,
				Labels: map[string]string{
					"camel.apache.org/integration": e.Integration.Name,
				},
				Annotations: map[string]string{
					"camel.apache.org/source.language":    string(v1alpha1.LanguageXML),
					"camel.apache.org/source.name":        resource.Name,
					"camel.apache.org/source.compression": strconv.FormatBool(resource.Compression),
					"camel.apache.org/source.generated":   "true",
					"camel.apache.org/source.type":        string(v1alpha1.ResourceTypeOpenAPI),
				},
			},
			Data: map[string]string{
				"content": string(content),
			},
		}

		e.Integration.Status.GeneratedSources = generatedSources
		e.Resources.Add(&cm)
	}

	return nil
}

func (t *restDslTrait) generateProject(e *Environment) (maven.Project, error) {
	if e.CamelCatalog == nil {
		return maven.Project{}, errors.New("unknown camel catalog")
	}

	p := maven.Project{
		XMLName:           xml.Name{Local: "project"},
		XMLNs:             "http://maven.apache.org/POM/4.0.0",
		XMLNsXsi:          "http://www.w3.org/2001/XMLSchema-instance",
		XsiSchemaLocation: "http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd",
		ModelVersion:      "4.0.0",
		GroupID:           "org.apache.camel.k.integration",
		ArtifactID:        "camel-k-red-dsl-generator",
		Version:           version.Version,
		Build: maven.Build{
			DefaultGoal: "generate-resources",
			Plugins: []maven.Plugin{
				{
					GroupID:    "org.apache.camel.k",
					ArtifactID: "camel-k-maven-plugin",
					Version:    version.Version,
					Executions: []maven.Execution{
						{
							Phase: "generate-resources",
							Goals: []string{
								"generate-rest-xml",
							},
						},
					},
					Dependencies: []maven.Dependency{
						{
							GroupID:    "org.apache.camel",
							ArtifactID: "camel-swagger-rest-dsl-generator",
							Version:    e.CamelCatalog.Version,
						},
					},
				},
			},
		},
	}

	//
	// Repositories
	//

	p.Repositories = make([]maven.Repository, 0, len(e.Platform.Spec.Build.Repositories))

	for i, r := range e.Platform.Spec.Build.Repositories {
		repo := maven.NewRepository(r)
		if repo.ID == "" {
			repo.ID = fmt.Sprintf("repo-%03d", i)
		}

		p.Repositories = append(p.Repositories, repo)
	}

	return p, nil
}
