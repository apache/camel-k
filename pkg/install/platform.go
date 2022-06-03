/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements. See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package install

import (
	"context"

	"github.com/apache/camel-k/pkg/client"
)

// IntegrationPlatformViewerRole installs the role that allows any user ro access integrationplatforms in the global namespace.
func IntegrationPlatformViewerRole(ctx context.Context, c client.Client, namespace string) error {
	if err := Resource(ctx, c, namespace, true, IdentityResourceCustomizer, "/viewer/user-global-platform-viewer-role.yaml"); err != nil {
		return err
	}
	return Resource(ctx, c, namespace, true, IdentityResourceCustomizer, "/viewer/user-global-platform-viewer-role-binding.yaml")
}
