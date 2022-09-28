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

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/apache/camel-k/pkg/client"
)

// addToManager is a list of functions to add all Controllers to the Manager.
var addToManager []func(ctrl.Manager, client.Client) error

// AddToManager adds all Controllers to the Manager.
func AddToManager(manager ctrl.Manager, client client.Client) error {
	for _, f := range addToManager {
		if err := f(manager, client); err != nil {
			return err
		}
	}
	return nil
}
