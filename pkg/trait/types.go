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
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/pkg/errors"
	"strconv"
)

// ID uniquely identifies a trait
type id string

// A Customizer performs customization of the deployed objects
type customizer interface {
	// The Name of the customizer
	id() id
	// Customize executes the trait customization on the resources and return true if the resources have been changed
	customize(environment environment, resources *kubernetes.Collection) (bool, error)
}

// A environment provides the context where the trait is executed
type environment struct {
	Platform            *v1alpha1.IntegrationPlatform
	Context             *v1alpha1.IntegrationContext
	Integration         *v1alpha1.Integration
	ExecutedCustomizers []id
}

func (e environment) getTraitSpec(traitID id) *v1alpha1.IntegrationTraitSpec {
	if e.Integration.Spec.Traits == nil {
		return nil
	}
	if conf, ok := e.Integration.Spec.Traits[string(traitID)]; ok {
		return &conf
	}
	return nil
}

func (e environment) isEnabled(traitID id) bool {
	conf := e.getTraitSpec(traitID)
	return conf == nil || conf.Enabled == nil || *conf.Enabled
}

func (e environment) getConfig(traitID id, key string) *string {
	conf := e.getTraitSpec(traitID)
	if conf == nil || conf.Configuration == nil {
		return nil
	}
	if v, ok := conf.Configuration[key]; ok {
		return &v
	}
	return nil
}

func (e environment) getConfigOr(traitID id, key string, defaultValue string) string {
	val := e.getConfig(traitID, key)
	if val != nil {
		return *val
	}
	return defaultValue
}

func (e environment) getIntConfig(traitID id, key string) (*int, error) {
	val := e.getConfig(traitID, key)
	if val == nil {
		return nil, nil
	}
	intVal, err := strconv.Atoi(*val)
	if err != nil {
		return nil, errors.Wrap(err, "cannot extract a integer from property "+key+" with value "+*val)
	}
	return &intVal, nil
}

func (e environment) getIntConfigOr(traitID id, key string, defaultValue int) (int, error) {
	val, err := e.getIntConfig(traitID, key)
	if err != nil {
		return 0, err
	}
	if val != nil {
		return *val, nil
	}
	return defaultValue, nil
}
