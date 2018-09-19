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
	"time"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

// ResourceRetrieveFunction --
type ResourceRetrieveFunction func() (interface{}, error)

// ResourceCheckFunction --
type ResourceCheckFunction func(interface{}) (bool, error)

const (
	sleepTime = 400 * time.Millisecond
)

// WaitCondition --
func WaitCondition(obj runtime.Object, condition ResourceCheckFunction, maxDuration time.Duration) error {
	start := time.Now()

	for start.Add(maxDuration).After(time.Now()) {
		err := sdk.Get(obj)
		if err != nil {
			time.Sleep(sleepTime)
			continue
		}

		satisfied, err := condition(obj)
		if err != nil {
			return errors.Wrap(err, "error while evaluating condition")
		} else if !satisfied {
			time.Sleep(sleepTime)
			continue
		}

		return nil
	}
	return errors.New("timeout while waiting condition")
}
