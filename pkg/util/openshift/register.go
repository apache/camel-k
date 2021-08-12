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

package openshift

import (
	"k8s.io/apimachinery/pkg/runtime"

	apps "github.com/openshift/api/apps/v1"
	authorization "github.com/openshift/api/authorization/v1"
	build "github.com/openshift/api/build/v1"
	config "github.com/openshift/api/config/v1"
	console "github.com/openshift/api/console/v1"
	image "github.com/openshift/api/image/v1"
	project "github.com/openshift/api/project/v1"
	route "github.com/openshift/api/route/v1"
	template "github.com/openshift/api/template/v1"

	"github.com/apache/camel-k/pkg/util/log"
)

type registerFunction func(*runtime.Scheme) error

// AddToScheme adds OpenShift types to the scheme
func AddToScheme(scheme *runtime.Scheme) error {
	var err error

	// Standardized groups
	err = doAdd(apps.Install, scheme, err)
	err = doAdd(template.Install, scheme, err)
	err = doAdd(image.Install, scheme, err)
	err = doAdd(route.Install, scheme, err)
	err = doAdd(build.Install, scheme, err)
	err = doAdd(authorization.Install, scheme, err)
	err = doAdd(project.Install, scheme, err)
	err = doAdd(config.Install, scheme, err)

	// OpenShift console API
	err = doAdd(console.Install, scheme, err)

	return err
}

func doAdd(addToScheme registerFunction, scheme *runtime.Scheme, err error) error {
	callErr := addToScheme(scheme)
	if callErr != nil {
		log.Error(callErr, "Error while registering OpenShift types")
	}

	if err == nil {
		return callErr
	}
	return err
}
