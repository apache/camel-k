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

package trait

// The GitOps Trait is used to configure the repository where you want to push a GitOps Kustomize overlay configuration of the Integration built.
// If the trait is enabled but no pull configuration is provided, then, the operator will use the values stored in Integration `.spec.git` field used
// to pull the project.
//
// +camel-k:trait=gitops.
//
//nolint:godoclint
type GitOpsTrait struct {
	Trait `json:",inline" property:",squash"`

	// the URL of the repository where the project is stored.
	URL string `json:"url,omitempty" property:"url"`
	// the Kubernetes secret where the Git token is stored. The operator will pick up the first secret key only, whichever the name it is.
	Secret string `json:"secret,omitempty" property:"secret"`
	// the git branch to check out.
	Branch string `json:"branch,omitempty" property:"branch"`
	// the git tag to check out.
	Tag string `json:"tag,omitempty" property:"tag"`
	// the git commit (full SHA) to check out.
	Commit string `json:"commit,omitempty" property:"commit"`
	// the git branch to push to. If omitted, the operator will push to a new branch named as `cicd/release-candidate-<datetime>`.
	BranchPush string `json:"branchPush,omitempty" property:"branch-push"`
	// a list of overlays to provide (default {"dev","stag","prod"}).
	Overlays []string `json:"overlays,omitempty" property:"overlays"`
	// a flag (default, false) to overwrite any existing overlay.
	OverwriteOverlay bool `json:"overwriteOverlay,omitempty" property:"overwrite-overlay"`
	// The root path where to store Kustomize overlays (default `integrations`).
	IntegrationDirectory string `json:"integrationDirectory,omitempty" property:"integration-directory"`
	// The name used to commit the GitOps changes (default `Camel K Operator`).
	CommiterName string `json:"committerName,omitempty" property:"committed-name"`
	// The email used to commit the GitOps changes (default `camel-k-operator@apache.org`).
	CommiterEmail string `json:"committerEmail,omitempty" property:"committed-email"`
}
