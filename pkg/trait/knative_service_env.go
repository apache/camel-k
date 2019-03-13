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
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/envvar"
	"github.com/apache/camel-k/pkg/util/kubernetes"

	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
)

func (t *knativeServiceTrait) bindToEnvVar(e *Environment, service *serving.Service) error {
	environment := &service.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.Env

	//
	// Properties
	//

	properties := make(map[string]string)

	VisitKeyValConfigurations("property", e.IntegrationContext, e.Integration, func(key string, val string) {
		properties[key] = val
	})

	VisitConfigurations("configmap", e.IntegrationContext, e.Integration, func(cmName string) {
		cm, err := kubernetes.GetConfigMap(e.C, e.Client, cmName, e.Integration.Namespace)
		if err != nil {
			t.L.Errorf(err, "failed to lookup ConfigMap %s", cmName)
		}
		if cm != nil {
			err = util.ExtractApplicationPropertiesString(cm.Data, func(key string, val string) {
				properties[key] = val
			})
			if err != nil {
				t.L.Errorf(err, "failed to extract properties from ConfigMap %s", cmName)
			}
		}
	})

	VisitConfigurations("secret", e.IntegrationContext, e.Integration, func(secretName string) {
		cm, err := kubernetes.GetSecret(e.C, e.Client, secretName, e.Integration.Namespace)
		if err != nil {
			t.L.Errorf(err, "failed to lookup Secret %s", secretName)
		}
		if cm != nil {
			err = util.ExtractApplicationPropertiesBytes(cm.Data, func(key string, val string) {
				properties[key] = val
			})
			if err != nil {
				t.L.Errorf(err, "failed to extract properties from Secret %s", secretName)
			}
		}
	})

	p := ""

	for k, v := range properties {
		p += fmt.Sprintf("%s=%s\n", k, v)
	}

	envvar.SetVal(environment, "CAMEL_K_CONF", "env:CAMEL_K_PROPERTIES")
	envvar.SetVal(environment, "CAMEL_K_PROPERTIES", p)

	//
	// Sources
	//

	sourcesSpecs, err := kubernetes.ResolveIntegrationSources(t.ctx, t.client, e.Integration, e.Resources)
	if err != nil {
		return err
	}

	sources := make([]string, 0, len(e.Integration.Spec.Sources))

	for i, s := range sourcesSpecs {
		if s.Content == "" {
			t.L.Debug("Source %s has and empty content", s.Name)
		}

		envName := fmt.Sprintf("CAMEL_K_ROUTE_%03d", i)
		envvar.SetVal(environment, envName, s.Content)

		params := make([]string, 0)
		params = append(params, "name="+s.Name)

		if s.InferLanguage() != "" {
			params = append(params, "language="+string(s.InferLanguage()))
		}
		if s.Compression {
			params = append(params, "compression=true")
		}

		src := fmt.Sprintf("env:%s", envName)
		if len(params) > 0 {
			src = fmt.Sprintf("%s?%s", src, strings.Join(params, "&"))
		}

		sources = append(sources, src)
	}

	// camel-k runtime
	envvar.SetVal(environment, "CAMEL_K_ROUTES", strings.Join(sources, ","))

	//
	// Resources
	//

	resourcesSpecs, err := kubernetes.ResolveIntegrationResources(t.ctx, t.client, e.Integration, e.Resources)
	if err != nil {
		return err
	}

	for i, r := range resourcesSpecs {
		if r.Type != v1alpha1.ResourceTypeData {
			continue
		}

		envName := fmt.Sprintf("CAMEL_K_RESOURCE_%03d", i)
		envvar.SetVal(environment, envName, r.Content)

		params := make([]string, 0)
		if r.Compression {
			params = append(params, "compression=true")
		}

		envValue := fmt.Sprintf("env:%s", envName)
		if len(params) > 0 {
			envValue = fmt.Sprintf("%s?%s", envValue, strings.Join(params, "&"))
		}

		envName = r.Name
		envName = strings.ToUpper(envName)
		envName = strings.Replace(envName, "-", "_", -1)
		envName = strings.Replace(envName, ".", "_", -1)
		envName = strings.Replace(envName, " ", "_", -1)

		envvar.SetVal(environment, envName, envValue)
	}

	return nil
}
