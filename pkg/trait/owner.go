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
	"github.com/apache/camel-k/pkg/util/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ownerTrait ensures that all created resources belong to the integration being created
type ownerTrait struct {
	BaseTrait `property:",squash"`
}

func newOwnerTrait() *ownerTrait {
	return &ownerTrait{
		BaseTrait: newBaseTrait("owner"),
	}
}

func (*ownerTrait) customize(e *environment, resources *kubernetes.Collection) error {
	controller := true
	blockOwnerDeletion := true
	resources.VisitMetaObject(func(res metav1.Object) {
		references := []metav1.OwnerReference{
			{
				APIVersion:         e.Integration.APIVersion,
				Kind:               e.Integration.Kind,
				Name:               e.Integration.Name,
				UID:                e.Integration.UID,
				Controller:         &controller,
				BlockOwnerDeletion: &blockOwnerDeletion,
			},
		}
		res.SetOwnerReferences(references)
	})
	return nil
}
