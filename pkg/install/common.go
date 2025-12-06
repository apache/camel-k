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

package install

import (
	networking "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourceCustomizer can be used to inject code that changes the objects before they are created.
type ResourceCustomizer func(object ctrl.Object) ctrl.Object

// IdentityResourceCustomizer is a ResourceCustomizer that does nothing.
var IdentityResourceCustomizer = func(object ctrl.Object) ctrl.Object {
	return object
}

var RemoveIngressRoleCustomizer = func(object ctrl.Object) ctrl.Object {
	if role, ok := object.(*rbacv1.Role); ok && role.Name == "camel-k-operator" {
	rules:
		for i, rule := range role.Rules {
			for _, group := range rule.APIGroups {
				if group == networking.GroupName {
					role.Rules = append(role.Rules[:i], role.Rules[i+1:]...)

					break rules
				}
			}
		}
	}

	return object
}
