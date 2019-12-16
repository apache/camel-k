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

package kaniko

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/tar"
)

func publisher(ctx *builder.Context) error {
	baseDir, _ := path.Split(ctx.Archive)
	contextDir := path.Join(baseDir, "context")

	err := os.MkdirAll(contextDir, 0777)
	if err != nil {
		return err
	}

	if err := tar.Extract(ctx.Archive, contextDir); err != nil {
		return err
	}

	// #nosec G202
	dockerFileContent := []byte(`
		FROM ` + ctx.BaseImage + `
		ADD . /deployments
	`)

	err = ioutil.WriteFile(path.Join(contextDir, "Dockerfile"), dockerFileContent, 0777)
	if err != nil {
		return err
	}

	return nil
}
