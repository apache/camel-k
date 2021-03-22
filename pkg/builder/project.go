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
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func init() {
	registerSteps(Steps)
}

type steps struct {
	CleanUpBuildDir         Step
	GenerateJavaKeystore    Step
	GenerateProjectSettings Step
	InjectDependencies      Step
	SanitizeDependencies    Step
	StandardImageContext    Step
	IncrementalImageContext Step
}

var Steps = steps{
	CleanUpBuildDir:         NewStep(ProjectGenerationPhase-1, cleanUpBuildDir),
	GenerateJavaKeystore:    NewStep(ProjectGenerationPhase, generateJavaKeystore),
	GenerateProjectSettings: NewStep(ProjectGenerationPhase+1, generateProjectSettings),
	InjectDependencies:      NewStep(ProjectGenerationPhase+2, injectDependencies),
	SanitizeDependencies:    NewStep(ProjectGenerationPhase+3, sanitizeDependencies),
	StandardImageContext:    NewStep(ApplicationPackagePhase, standardImageContext),
	IncrementalImageContext: NewStep(ApplicationPackagePhase, incrementalImageContext),
}

var DefaultSteps = []Step{
	Steps.CleanUpBuildDir,
	Steps.GenerateJavaKeystore,
	Steps.GenerateProjectSettings,
	Steps.InjectDependencies,
	Steps.SanitizeDependencies,
	Steps.IncrementalImageContext,
}

func cleanUpBuildDir(ctx *builderContext) error {
	if ctx.Build.BuildDir == "" {
		return nil
	}

	return os.RemoveAll(ctx.Build.BuildDir)
}

func generateJavaKeystore(ctx *builderContext) error {
	if ctx.Build.Maven.CaCert == nil {
		return nil
	}

	certData, err := kubernetes.GetSecretRefData(ctx.C, ctx.Client, ctx.Namespace, ctx.Build.Maven.CaCert)
	if err != nil {
		return err
	}

	certPath := ctx.Build.Maven.CaCert.Key
	if err := util.WriteFileWithContent(ctx.Path, certPath, certData); err != nil {
		return err
	}

	keystore := "trust.jks"
	ctx.Maven.TrustStorePath = path.Join(ctx.Path, keystore)

	args := strings.Fields(fmt.Sprintf("-importcert -alias maven -file %s -keystore %s", certPath, keystore))
	cmd := exec.CommandContext(ctx.C, "keytool", args...)
	cmd.Dir = ctx.Path
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func generateProjectSettings(ctx *builderContext) error {
	val, err := kubernetes.ResolveValueSource(ctx.C, ctx.Client, ctx.Namespace, &ctx.Build.Maven.Settings)
	if err != nil {
		return err
	}
	if val != "" {
		ctx.Maven.SettingsData = []byte(val)
	}

	return nil
}

func injectDependencies(ctx *builderContext) error {
	// Add dependencies from build
	return camel.ManageIntegrationDependencies(&ctx.Maven.Project, ctx.Build.Dependencies, ctx.Catalog)
}

func sanitizeDependencies(ctx *builderContext) error {
	return camel.SanitizeIntegrationDependencies(ctx.Maven.Project.Dependencies)
}
