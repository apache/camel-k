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

package kubernetes

import (
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Register all OpenShift types that we want to manage.
func init() {
	k8sutil.AddToSDKScheme(addKnownTypes)
}

type registerFunction func(*runtime.Scheme) error

func addKnownTypes(scheme *runtime.Scheme) error {
	gv := schema.GroupVersion{
		Group:   "apiextensions.k8s.io",
		Version: "v1beta1",
	}
	scheme.AddKnownTypes(gv, &apiextensions.CustomResourceDefinition{}, &apiextensions.CustomResourceDefinitionList{})
	return nil
}
