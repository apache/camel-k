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

package jib

import (
	"context"
	"os"
	"path/filepath"

	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util"
	"go.uber.org/multierr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	JibCliCmdBinary = "/opt/jib/bin/jib"

	JibCliCmdBuild              = "build"
	JibCliParamBuildFile        = "--build-file="
	JibCliParamTarget           = "--target="
	JibCliParamOutput           = "--image-metadata-out="
	JibCliParamInsecureRegistry = "--allow-insecure-registries"
)

// Mount the secret containing the docker auth configuration into the folder ${jibContextDir}/.docker/.
func MountJibSecret(ctx context.Context, c client.Client, namespace, name string, jibContextDir string) (string, error) {
	dockerConfigDir := filepath.Join(jibContextDir, ".docker")

	err := util.CreateDirectory(dockerConfigDir)
	if err != nil {
		return "", err
	}

	secret, err := c.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if removeErr := os.RemoveAll(dockerConfigDir); removeErr != nil {
			err = multierr.Append(err, removeErr)
		}
		return "", err
	}

	for file, content := range secret.Data {
		if err := os.WriteFile(filepath.Join(dockerConfigDir, remap(file)), content, 0o600); err != nil {
			if removeErr := os.Remove(filepath.Join(dockerConfigDir, remap(file))); removeErr != nil {
				err = multierr.Append(err, removeErr)
			}
			return "", err
		}
	}
	return dockerConfigDir, nil
}

func remap(name string) string {
	if name == ".dockerconfigjson" {
		return "config.json"
	}
	return name
}
