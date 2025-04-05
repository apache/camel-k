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

package builder

import (
	"os"
	"path/filepath"
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/digest"
	"github.com/apache/camel-k/v2/pkg/util/maven"
)

// newMavenContext returns a maven Context.
func newMavenContext(ctx *builderContext) *maven.Context {
	mc := maven.NewContext(filepath.Join(ctx.Path, "maven"))
	mc.GlobalSettings = ctx.Maven.GlobalSettings
	mc.UserSettings = ctx.Maven.UserSettings
	mc.SettingsSecurity = ctx.Maven.SettingsSecurity
	mc.LocalRepository = ctx.Build.Maven.LocalRepository
	mc.AdditionalArguments = ctx.Build.Maven.CLIOptions

	if ctx.Maven.TrustStoreName != "" {
		mc.ExtraMavenOpts = append(mc.ExtraMavenOpts,
			"-Djavax.net.ssl.trustStore="+filepath.Join(ctx.Path, ctx.Maven.TrustStoreName),
			"-Djavax.net.ssl.trustStorePassword="+ctx.Maven.TrustStorePass,
		)
	}

	return &mc
}

// buildMavenContextSettings create the maven project structure.
func buildMavenContextSettings(ctx *builderContext) error {
	mc := newMavenContext(ctx)

	return ctx.Maven.Project.Command(*mc).DoSettings(ctx.C)
}

// executeMavenPackageCommand is in charge to execute the maven command.
func executeMavenPackageCommand(ctx *builderContext) error {
	mc := newMavenContext(ctx)
	mc.AddArgument("package")

	return ctx.Maven.Project.Command(*mc).Do(ctx.C)
}

// computeDependencies sets in context all dependencies found in target directory.
func computeFatJarDependency(ctx *builderContext) error {
	dir := filepath.Join(ctx.Path, "maven", "target")
	artifacts, err := processDependencies(dir, true)
	if err != nil {
		return err
	}
	ctx.Artifacts = append(ctx.Artifacts, artifacts...)

	return nil
}

// processDependencies walks a dir and return a list of all found jar artifacts. It can skip nested directories.
func processDependencies(dir string, skipNestedDirs bool) ([]v1.Artifact, error) {
	var artifacts []v1.Artifact
	err := filepath.WalkDir(dir, func(filePath string, d os.DirEntry, err error) error {
		if dir == filePath {
			return nil
		}
		if err != nil {
			return err
		}
		if skipNestedDirs && d.IsDir() && d.Name() != dir {
			return filepath.SkipDir
		}
		fileRelPath := strings.Replace(filePath, dir, "", 1)
		if !d.IsDir() && strings.HasSuffix(d.Name(), "jar") {
			sha1, err := digest.ComputeSHA1(filePath)
			if err != nil {
				return err
			}
			artifacts = append(artifacts, v1.Artifact{
				ID:       filepath.Base(fileRelPath),
				Location: filePath,
				Target:   filepath.Join(DependenciesDir, fileRelPath),
				Checksum: "sha1:" + sha1,
			})
		}

		return nil
	})

	return artifacts, err
}
