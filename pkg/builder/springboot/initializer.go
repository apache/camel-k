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

package springboot

import (
	"github.com/apache/camel-k/pkg/builder"
)

// Initialize --
func Initialize(ctx *builder.Context) error {
	// set the base image
	//ctx.Image = "kamel-k/s2i-boot:" + version.Version

	// no need to compute classpath as we do use spring boot own
	// loader: PropertiesLauncher
	ctx.ComputeClasspath = false

	return nil
}
