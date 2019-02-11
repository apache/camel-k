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

package camel

import (
	"context"
	"fmt"

	"github.com/apache/camel-k/pkg/client"
)

// R --
var R Runtime

func init() {
	R = NewRuntime()
}

// Catalog --
func Catalog(ctx context.Context, client client.Client, namespace string, version string) (*RuntimeCatalog, error) {
	c, err := R.LoadCatalog(ctx, client, namespace, version)
	if c == nil && err != nil {
		return nil, fmt.Errorf("unable to find catalog matching version requirement: %s", version)
	}

	return c, err
}
