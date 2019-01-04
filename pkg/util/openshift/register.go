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
	apps "github.com/openshift/api/apps/v1"
	authorization "github.com/openshift/api/authorization/v1"
	build "github.com/openshift/api/build/v1"
	image "github.com/openshift/api/image/v1"
	route "github.com/openshift/api/route/v1"
	template "github.com/openshift/api/template/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
)

type registerFunction func(*runtime.Scheme) error

// AddToScheme adds OpenShift types to the scheme
func AddToScheme(scheme *runtime.Scheme) error {

	var err error

	// Standardized groups
	err = doAdd(apps.AddToScheme, scheme, err)
	err = doAdd(template.AddToScheme, scheme, err)
	err = doAdd(image.AddToScheme, scheme, err)
	err = doAdd(route.AddToScheme, scheme, err)
	err = doAdd(build.AddToScheme, scheme, err)
	err = doAdd(authorization.AddToScheme, scheme, err)

	return err
}

func doAdd(addToScheme registerFunction, scheme *runtime.Scheme, err error) error {
	callErr := addToScheme(scheme)
	if callErr != nil {
		logrus.Error("Error while registering Openshift types", callErr)
	}

	if err == nil {
		return callErr
	}
	return err
}
