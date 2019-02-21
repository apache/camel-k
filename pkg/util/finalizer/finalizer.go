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
package finalizer

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// CamelIntegrationFinalizer --
	CamelIntegrationFinalizer = "finalizer.integration.camel.apache.org"

	// ForegroundDeletion --
	ForegroundDeletion = "foregroundDeletion"
)

// Add --
func Add(obj runtime.Object, value string) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	finalizers := sets.NewString(accessor.GetFinalizers()...)
	finalizers.Insert(value)
	accessor.SetFinalizers(finalizers.List())

	return nil
}

// Exists --
func Exists(obj runtime.Object, finalizer string) (bool, error) {
	fzs, err := GetAll(obj)
	if err != nil {
		return false, err
	}
	for _, fin := range fzs {
		if fin == finalizer {
			return true, nil
		}
	}
	return false, nil
}

// GetAll --
func GetAll(obj runtime.Object) ([]string, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	return accessor.GetFinalizers(), nil
}

// Remove --
func Remove(obj runtime.Object, value string) ([]string, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	finalizers := sets.NewString(accessor.GetFinalizers()...)
	finalizers.Delete(value)
	newFinalizers := finalizers.List()
	accessor.SetFinalizers(newFinalizers)
	return newFinalizers, nil
}
