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

	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/jvm"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func init() {
	registerSteps(Project)

	Project.CommonSteps = []Step{
		Project.CleanUpBuildDir,
		Project.GenerateJavaKeystore,
		Project.GenerateProjectSettings,
		Project.InjectDependencies,
		Project.SanitizeDependencies,
	}
}

type projectSteps struct {
	CleanUpBuildDir         Step
	GenerateJavaKeystore    Step
	GenerateProjectSettings Step
	InjectDependencies      Step
	SanitizeDependencies    Step

	CommonSteps []Step
}

var Project = projectSteps{
	CleanUpBuildDir:         NewStep(ProjectGenerationPhase-1, cleanUpBuildDir),
	GenerateJavaKeystore:    NewStep(ProjectGenerationPhase, generateJavaKeystore),
	GenerateProjectSettings: NewStep(ProjectGenerationPhase+1, generateProjectSettings),
	InjectDependencies:      NewStep(ProjectGenerationPhase+2, injectDependencies),
	SanitizeDependencies:    NewStep(ProjectGenerationPhase+3, sanitizeDependencies),
}

func cleanUpBuildDir(ctx *builderContext) error {
	if ctx.Build.BuildDir == "" {
		return nil
	}

	err := os.RemoveAll(ctx.Build.BuildDir)
	if err != nil {
		return err
	}

	return os.MkdirAll(ctx.Build.BuildDir, 0777)
}

func generateJavaKeystore(ctx *builderContext) error {
	if ctx.Build.Maven.CASecret == nil {
		return nil
	}

	certData, err := kubernetes.GetSecretRefData(ctx.C, ctx.Client, ctx.Namespace, ctx.Build.Maven.CASecret)
	if err != nil {
		return err
	}

	ctx.Maven.TrustStoreName = "trust.jks"
	ctx.Maven.TrustStorePass = jvm.NewKeystorePassword()

	return jvm.GenerateKeystore(ctx.C, ctx.Path, ctx.Maven.TrustStoreName, ctx.Maven.TrustStorePass, certData)
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
