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

package defaults

import (
	"os"
	"strconv"

	"github.com/apache/camel-k/pkg/util/log"
)

func BaseImage() string {
	return envOrDefault(baseImage, "KAMEL_BASE_IMAGE", "RELATED_IMAGE_BASE")
}

func InstallDefaultKamelets() bool {
	return boolEnvOrDefault(installDefaultKamelets, "KAMEL_INSTALL_DEFAULT_KAMELETS")
}

func boolEnvOrDefault(def bool, envs ...string) bool {
	strVal := envOrDefault(strconv.FormatBool(def), envs...)
	res, err := strconv.ParseBool(strVal)
	if err != nil {
		log.Error(err, "cannot parse boolean property", "property", def, "value", strVal)
	}
	return res
}

func envOrDefault(def string, envs ...string) string {
	for i := range envs {
		if val := os.Getenv(envs[i]); val != "" {
			return val
		}
	}
	return def
}
