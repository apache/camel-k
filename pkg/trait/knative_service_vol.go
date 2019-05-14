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
	"strings"

	"github.com/apache/camel-k/pkg/util/envvar"
	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
)

func (t *knativeServiceTrait) bindToVolumes(e *Environment, service *serving.Service) {
	e.ConfigureVolumesAndMounts(
		&service.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Volumes,
		&service.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.VolumeMounts,
	)

	paths := e.ComputeSourcesURI()
	environment := &service.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.Env

	envvar.SetVal(environment, "CAMEL_K_ROUTES", strings.Join(paths, ","))
	envvar.SetVal(environment, "CAMEL_K_CONF", "/etc/camel/conf/application.properties")
	envvar.SetVal(environment, "CAMEL_K_CONF_D", "/etc/camel/conf.d")
}
