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

package integration

import (
	"reflect"

	"k8s.io/apimachinery/pkg/api/equality"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// StatusChangedPredicate implements a generic update predicate function on status change.
type StatusChangedPredicate struct {
	predicate.Funcs
}

// Update implements default UpdateEvent filter for validating status change.
func (StatusChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		Log.Error(nil, "Update event has no old object to update", "event", e)
		return false
	}
	if e.ObjectNew == nil {
		Log.Error(nil, "Update event has no new object to update", "event", e)
		return false
	}

	s1 := reflect.ValueOf(e.ObjectOld).Elem().FieldByName("Status")
	if !s1.IsValid() {
		Log.Error(nil, "Update event old object has no Status field", "event", e)
		return false
	}

	s2 := reflect.ValueOf(e.ObjectNew).Elem().FieldByName("Status")
	if !s2.IsValid() {
		Log.Error(nil, "Update event new object has no Status field", "event", e)
		return false
	}

	return !equality.Semantic.DeepDerivative(s1.Interface(), s2.Interface())
}

// NonManagedObjectPredicate implements a generic update predicate function for managed object.
type NonManagedObjectPredicate struct {
	predicate.Funcs
}

// Create --.
func (NonManagedObjectPredicate) Create(e event.CreateEvent) bool {
	return !isManagedObject(e.Object)
}

// Update --.
func (NonManagedObjectPredicate) Update(e event.UpdateEvent) bool {
	return !isManagedObject(e.ObjectNew)
}

// Delete --.
func (NonManagedObjectPredicate) Delete(e event.DeleteEvent) bool {
	return !isManagedObject(e.Object)
}

// Generic --.
func (NonManagedObjectPredicate) Generic(e event.GenericEvent) bool {
	return !isManagedObject(e.Object)
}

// isManagedObject returns true if the object is managed by an Integration.
func isManagedObject(obj ctrl.Object) bool {
	for _, mr := range obj.GetOwnerReferences() {
		if mr.APIVersion == "camel.apache.org/v1" &&
			mr.Kind == "Integration" {
			return true
		}
	}
	return false
}
