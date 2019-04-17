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

package build

import (
	"context"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	logger "github.com/apache/camel-k/pkg/util/log"
)

// UpdateBuildStatus --
func UpdateBuildStatus(ctx context.Context, build *v1alpha1.Build, status v1alpha1.BuildStatus, c client.Client, log logger.Logger) error {
	target := build.DeepCopy()
	target.Status = status
	// Copy the failure field from the build to persist recovery state
	target.Status.Failure = build.Status.Failure
	err := c.Status().Update(ctx, target)
	if err != nil {
		if k8serrors.IsConflict(err) {
			// Refresh the build
			err := c.Get(ctx, types.NamespacedName{Namespace: build.Namespace, Name: build.Name}, build)
			if err != nil {
				log.Error(err, "Build refresh failed")
				return err
			}
			return UpdateBuildStatus(ctx, build, status, c, log)
		}
		log.Error(err, "Build update failed")
		return err
	}
	build.Status = status
	return nil
}
