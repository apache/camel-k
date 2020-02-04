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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MatchingSelector struct {
	Selector labels.Selector
}

func (s MatchingSelector) ApplyToList(opts *client.ListOptions) {
	opts.LabelSelector = s.Selector
}

func (s MatchingSelector) ApplyToDeleteAllOf(opts *client.DeleteAllOfOptions) {
	opts.LabelSelector = s.Selector
}

func NewLabelSelector(key string, op selection.Operator, values []string) MatchingSelector {
	provider, _ := labels.NewRequirement(key, op, values)
	selector := labels.NewSelector().Add(*provider)

	return MatchingSelector{
		Selector: selector,
	}
}
