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
	"bytes"
	"encoding/xml"
	"os"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/jvm"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/maven"
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

	return os.MkdirAll(ctx.Build.BuildDir, 0o700)
}

func generateJavaKeystore(ctx *builderContext) error {
	// nolint: staticcheck
	secrets := mergeSecrets(ctx.Build.Maven.CASecrets, ctx.Build.Maven.CASecret)
	if secrets == nil {
		return nil
	}
	certsData, err := kubernetes.GetSecretsRefData(ctx.C, ctx.Client, ctx.Namespace, secrets)
	if err != nil {
		return err
	}

	ctx.Maven.TrustStoreName = "trust.jks"
	ctx.Maven.TrustStorePass = jvm.NewKeystorePassword()

	return jvm.GenerateKeystore(ctx.C, ctx.Path, ctx.Maven.TrustStoreName, ctx.Maven.TrustStorePass, certsData)
}

func mergeSecrets(secrets []corev1.SecretKeySelector, secret *corev1.SecretKeySelector) []corev1.SecretKeySelector {
	if secrets == nil && secret == nil {
		return nil
	}
	if secret == nil {
		return secrets
	}
	return append(secrets, *secret)
}

func generateProjectSettings(ctx *builderContext) error {
	val, err := kubernetes.ResolveValueSource(ctx.C, ctx.Client, ctx.Namespace, &ctx.Build.Maven.Settings)
	if err != nil {
		return err
	}
	val = injectServersIntoMavenSettings(val, ctx.Build.Maven.Servers)
	if val != "" {
		ctx.Maven.UserSettings = []byte(val)
	}

	settings, err := maven.NewSettings(maven.DefaultRepositories, maven.ProxyFromEnvironment)
	if err != nil {
		return err
	}
	data, err := settings.MarshalBytes()
	if err != nil {
		return err
	}
	ctx.Maven.GlobalSettings = data

	return nil
}

func injectServersIntoMavenSettings(settings string, servers []v1.Server) string {
	if servers == nil || len(servers) < 1 {
		return settings
	}
	newSettings, i := getServerTagIndex(settings)
	if i < 0 {
		log.Infof("Could not find a place to store Server information in Maven settings, skipping")
		return settings
	}
	content, err := encodeXMLNoHeader(servers)
	if err != nil {
		log.Infof("Could not marshall extra Servers into Maven settings, skipping")
		return settings
	}
	return newSettings[:i] + string(content) + newSettings[i:]
}

func encodeXMLNoHeader(content interface{}) ([]byte, error) {
	w := &bytes.Buffer{}
	w.WriteString("\n")
	e := xml.NewEncoder(w)
	e.Indent("    ", "  ")

	if err := e.Encode(content); err != nil {
		return []byte{}, err
	}
	w.WriteString("\n  ")
	return w.Bytes(), nil
}

// Return Index of </server> Tag in val. Creates Tag if necessary.
func getServerTagIndex(val string) (string, int) {
	serversTag := "\n  <servers></servers>\n"
	val = strings.Replace(val, "<servers/>", serversTag, 1)
	endServerTag := "</servers>"
	i := strings.Index(val, endServerTag)
	if i > 0 {
		return val, i
	}
	// create necessary tags
	tags := []string{"</proxies>", "<proxies/>", "</offline>", "<offline/>", "</usePluginRegistry>", "<usePluginRegistry/>", "</interactiveMode>", "<interactiveMode/>", "</localRepository>", "<localRepository/>"}
	i = -1
	for _, tag := range tags {
		i = strings.Index(val, tag)
		if i > 0 {
			i += len(tag)
			break
		}
	}
	if i < 0 {
		regexp := regexp.MustCompile(`<settings.*>`)
		loc := regexp.FindStringIndex(val)
		if loc == nil {
			return val, i
		}
		i = loc[1]
	}
	val = val[:i] + serversTag + val[i:]
	return val, strings.Index(val, endServerTag)
}

func injectDependencies(ctx *builderContext) error {
	// Add dependencies from build
	return camel.ManageIntegrationDependencies(&ctx.Maven.Project, ctx.Build.Dependencies, ctx.Catalog)
}

func sanitizeDependencies(ctx *builderContext) error {
	return camel.SanitizeIntegrationDependencies(ctx.Maven.Project.Dependencies)
}
