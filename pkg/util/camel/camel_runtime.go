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
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/log"
)

var (
	logger         = log.WithName("camel-deps")
	dependencyList = make([]string, 0)

	loggerInfo = func(s string) string {
		if strings.HasPrefix(s, "camel:") {
			dependencyList = append(dependencyList, s)
		}
		return ""
	}

	loggerError = func(s string) string { logger.Error(nil, s); return "" }
)

var ExecCommand = exec.Command

// LoadCatalog --.
func LoadCatalog(ctx context.Context, client client.Client, namespace string, runtime v1.RuntimeSpec) (*RuntimeCatalog, error) {
	options := []k8sclient.ListOption{
		k8sclient.InNamespace(namespace),
	}

	list := v1.NewCamelCatalogList()
	err := client.List(ctx, &list, options...)
	if err != nil {
		return nil, err
	}

	catalog, err := findBestMatch(list.Items, runtime)
	if err != nil {
		return nil, err
	}

	return catalog, nil
}

// DependencyList Uses a invokes kamelet-main to calculate the required dependencies found in the source integration.
// The program is in java/camel-deps package.
func DependencyList(source v1.SourceSpec) ([]string, error) {
	err := util.WithTempDir("camel-deps-tmp-", func(tmpDir string) error {
		integrationFilePath := filepath.Join(tmpDir, source.Name)
		if err := os.WriteFile(integrationFilePath, []byte(source.Content), os.ModePerm); err != nil {
			return fmt.Errorf("failure while writing camel-deps source file %s: %w", source.Name, err)
		}
		// the execution path is defined in the Dockerfile
		ctx := context.Background()
		cmd := ExecCommand("/usr/share/local/camel-deps/run.sh", integrationFilePath)
		err := util.RunAndLog(ctx, cmd, loggerInfo, loggerError)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return dependencyList, nil
}
