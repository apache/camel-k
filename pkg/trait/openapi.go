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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"

	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/gzip"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/maven"
)

// OpenAPITraitName ---
const OpenAPITraitName = "openapi"

// CamelRestPortProperty ---
const CamelRestPortProperty = "camel.context.rest-configuration.port"

// CamelRestDefaultPort ---
const CamelRestDefaultPort = "8080"

// The OpenAPI DSL trait is internally used to allow creating integrations from a OpenAPI specs.
//
// +camel-k:trait=openapi
type openAPITrait struct {
	BaseTrait `property:",squash"`
}

func newOpenAPITrait() Trait {
	return &openAPITrait{
		BaseTrait: NewBaseTrait(OpenAPITraitName, 300),
	}
}

// IsPlatformTrait overrides base class method
func (t *openAPITrait) IsPlatformTrait() bool {
	return true
}

func (t *openAPITrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if e.Integration == nil {
		return false, nil
	}

	// check if the runtime provides 'rest' capabilities
	if _, ok := e.CamelCatalog.Runtime.Capabilities["rest"]; !ok {
		t.L.Infof("the runtime provider %s does not declare 'rest' capability", e.CamelCatalog.Runtime.Provider)
	}

	for _, resource := range e.Integration.Spec.Resources {
		if resource.Type == v1.ResourceTypeOpenAPI {
			return e.IntegrationInPhase(
				v1.IntegrationPhaseInitialization,
				v1.IntegrationPhaseDeploying,
				v1.IntegrationPhaseRunning,
			), nil
		}
	}

	return false, nil
}

func (t *openAPITrait) Apply(e *Environment) error {
	if len(e.Integration.Spec.Resources) == 0 {
		return nil
	}

	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		t.computeDependencies(e)
		return t.generateRestDSL(e)
	}

	if e.IntegrationInPhase(v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning) {
		e.ApplicationProperties[CamelRestPortProperty] = CamelRestDefaultPort
		return nil
	}

	return nil
}

func (t *openAPITrait) generateRestDSL(e *Environment) error {
	root := os.TempDir()
	tmpDir, err := ioutil.TempDir(root, "openapi")
	if err != nil {
		return err
	}

	defer os.RemoveAll(tmpDir)

	for i, resource := range e.Integration.Spec.Resources {
		if resource.Type != v1.ResourceTypeOpenAPI {
			continue
		}
		if resource.Name == "" {
			return fmt.Errorf("no name defined for the openapi resource: %v", resource)
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

		in := path.Join(tmpDir, resource.Name)
		out := path.Join(tmpDir, "openapi-dsl.xml")

		err = ioutil.WriteFile(in, content, 0644)
		if err != nil {
			return err
		}

		project, err := t.generateMavenProject(e)
		if err != nil {
			return err
		}

		mc := maven.NewContext(tmpDir, project)
		mc.LocalRepository = e.Platform.Status.Build.Maven.LocalRepository
		mc.Timeout = e.Platform.Status.Build.Maven.GetTimeout().Duration
		mc.AddArgument("-Dopenapi.spec=" + in)
		mc.AddArgument("-Ddsl.out=" + out)

		settings, err := kubernetes.ResolveValueSource(e.C, e.Client, e.Integration.Namespace, &e.Platform.Status.Build.Maven.Settings)
		if err != nil {
			return err
		}
		if settings != "" {
			mc.SettingsContent = []byte(settings)
		}

		err = maven.Run(mc)
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
		generatedSources := make([]v1.SourceSpec, 0, len(e.Integration.Status.GeneratedSources))

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
		generatedSources = append(generatedSources, v1.SourceSpec{
			DataSpec: v1.DataSpec{
				Name:        generatedSourceName,
				ContentRef:  generatedContentName,
				Compression: resource.Compression,
			},
			Language: v1.LanguageXML,
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
					"camel.apache.org/source.language":    string(v1.LanguageXML),
					"camel.apache.org/source.name":        resource.Name,
					"camel.apache.org/source.compression": strconv.FormatBool(resource.Compression),
					"camel.apache.org/source.generated":   "true",
					"camel.apache.org/source.type":        string(v1.ResourceTypeOpenAPI),
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

func (t *openAPITrait) generateMavenProject(e *Environment) (maven.Project, error) {
	if e.CamelCatalog == nil {
		return maven.Project{}, errors.New("unknown camel catalog")
	}

	p := maven.NewProjectWithGAV("org.apache.camel.k.integration", "camel-k-rest-dsl-generator", defaults.Version)
	p.Build = &maven.Build{
		DefaultGoal: "generate-resources",
		Plugins: []maven.Plugin{
			{
				GroupID:    "org.apache.camel.k",
				ArtifactID: "camel-k-maven-plugin",
				Version:    e.CamelCatalog.Runtime.Version,
				Executions: []maven.Execution{
					{
						Phase: "generate-resources",
						Goals: []string{
							"generate-rest-xml",
						},
					},
				},
			},
		},
	}

	return p, nil
}

func (t *openAPITrait) computeDependencies(e *Environment) {
	// check if the runtime provides 'rest' capabilities
	if capability, ok := e.CamelCatalog.Runtime.Capabilities["rest"]; ok {
		for _, dependency := range capability.Dependencies {
			util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, fmt.Sprintf("mvn:%s/%s", dependency.GroupID, dependency.ArtifactID))
		}

		// sort the dependencies to get always the same list if they don't change
		sort.Strings(e.Integration.Status.Dependencies)
	}
}
