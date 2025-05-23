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

package v1

import (
	"errors"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewBuild(namespace string, name string) *Build {
	return &Build{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       BuildKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// BuilderPodNamespace returns the namespace of the operator in charge to reconcile this Build.
func (build *Build) BuilderPodNamespace() string {
	for _, t := range build.Spec.Tasks {
		if t.Builder != nil {
			return t.Builder.Configuration.BuilderPodNamespace
		}
	}
	return ""
}

// BuilderConfiguration returns the builder configuration for this Build.
func (build *Build) BuilderConfiguration() *BuildConfiguration {
	return build.TaskConfiguration("builder")
}

// TaskConfiguration returns the task configuration of this Build.
func (build *Build) TaskConfiguration(name string) *BuildConfiguration {
	return ConfigurationTasksByName(build.Spec.Tasks, name)
}

// BuilderDependencies returns the list of dependencies configured on by the builder task for this Build.
func (build *Build) BuilderDependencies() []string {
	if builder, ok := FindBuilderTask(build.Spec.Tasks); ok {
		return builder.Dependencies
	}

	return []string{}
}

func (build *Build) RuntimeVersion() *string {
	if builder, ok := FindBuilderTask(build.Spec.Tasks); ok {
		return &builder.Runtime.Version
	}

	return nil
}

// FindBuilderTask returns the 1st builder task from the task list.
func FindBuilderTask(tasks []Task) (*BuilderTask, bool) {
	for _, t := range tasks {
		if t.Builder != nil {
			return t.Builder, true
		}
	}
	return nil, false
}

// ConfigurationTasksByName returns the container configuration from the task list.
func ConfigurationTasksByName(tasks []Task, name string) *BuildConfiguration {
	for _, t := range tasks {
		if t.Builder != nil && t.Builder.Name == name {
			return &t.Builder.Configuration
		}
		if t.Custom != nil && t.Custom.Name == name {
			return &t.Custom.Configuration
		}
		if t.Package != nil && t.Package.Name == name {
			return &t.Package.Configuration
		}
		if t.Spectrum != nil && t.Spectrum.Name == name {
			return &t.Spectrum.Configuration
		}
		if t.S2i != nil && t.S2i.Name == name {
			return &t.S2i.Configuration
		}
		if t.Jib != nil && t.Jib.Name == name {
			return &t.Jib.Configuration
		}
	}
	return &BuildConfiguration{}
}

// SetBuilderConfigurationTasks set the configuration required for the builder in the list of tasks.
func SetBuilderConfigurationTasks(tasks []Task, conf *BuildConfiguration) {
	for _, t := range tasks {
		if t.Builder != nil {
			t.Builder.Configuration = *conf
			return
		}
	}
}

func (buildPhase *BuildPhase) String() string {
	return string(*buildPhase)
}

// GetRunnable returns the executable jar (if available).
func (in *BuildStatus) GetRunnable() (*Artifact, error) {
	if len(in.Artifacts) == 0 {
		return nil, errors.New("build has no artifact")
	}
	if len(in.Artifacts) > 1 {
		return nil, errors.New("build has more than a single expected artifact")
	}

	return &in.Artifacts[0], nil
}

// GetCondition returns the condition with the provided type.
func (in *BuildStatus) GetCondition(condType BuildConditionType) *BuildCondition {
	for i := range in.Conditions {
		c := in.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

func (in *BuildStatus) Failed(err error) BuildStatus {
	in.Error = err.Error()
	in.Phase = BuildPhaseFailed
	return *in
}

func (in *BuildStatus) IsFinished() bool {
	return in.Phase == BuildPhaseSucceeded || in.Phase == BuildPhaseFailed ||
		in.Phase == BuildPhaseInterrupted || in.Phase == BuildPhaseError
}

func (in *BuildStatus) SetCondition(condType BuildConditionType, status corev1.ConditionStatus, reason string, message string) {
	in.SetConditions(BuildCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

func (in *BuildStatus) SetErrorCondition(condType BuildConditionType, reason string, err error) {
	in.SetConditions(BuildCondition{
		Type:               condType,
		Status:             corev1.ConditionFalse,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            err.Error(),
	})
}

// SetConditions updates the resource to include the provided conditions.
//
// If a condition that we are about to add already exists and has the same status and
// reason then we are not going to update.
func (in *BuildStatus) SetConditions(conditions ...BuildCondition) {
	for _, condition := range conditions {
		if condition.LastUpdateTime.IsZero() {
			condition.LastUpdateTime = metav1.Now()
		}
		if condition.LastTransitionTime.IsZero() {
			condition.LastTransitionTime = metav1.Now()
		}

		currentCond := in.GetCondition(condition.Type)

		if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
			return
		}
		// Do not update lastTransitionTime if the status of the condition doesn't change.
		if currentCond != nil && currentCond.Status == condition.Status {
			condition.LastTransitionTime = currentCond.LastTransitionTime
		}

		in.RemoveCondition(condition.Type)
		in.Conditions = append(in.Conditions, condition)
	}
}

// RemoveCondition removes the resource condition with the provided type.
func (in *BuildStatus) RemoveCondition(condType BuildConditionType) {
	newConditions := in.Conditions[:0]
	for _, c := range in.Conditions {
		if c.Type != condType {
			newConditions = append(newConditions, c)
		}
	}

	in.Conditions = newConditions
}

var _ ResourceCondition = &BuildCondition{}

func (in *BuildStatus) GetConditions() []ResourceCondition {
	res := make([]ResourceCondition, 0, len(in.Conditions))
	for _, c := range in.Conditions {
		res = append(res, &c)
	}
	return res
}

func (c *BuildCondition) GetType() string {
	return string(c.Type)
}

func (c *BuildCondition) GetStatus() corev1.ConditionStatus {
	return c.Status
}

func (c *BuildCondition) GetLastUpdateTime() metav1.Time {
	return c.LastUpdateTime
}

func (c *BuildCondition) GetLastTransitionTime() metav1.Time {
	return c.LastTransitionTime
}

func (c *BuildCondition) GetReason() string {
	return c.Reason
}

func (c *BuildCondition) GetMessage() string {
	return c.Message
}

func (bl *BuildList) HasRunningBuilds() bool {
	for _, b := range bl.Items {
		if b.Status.Phase == BuildPhasePending || b.Status.Phase == BuildPhaseRunning {
			return true
		}
	}

	return false
}

func (bl *BuildList) HasScheduledBuildsBefore(build *Build) (bool, *Build) {
	for _, b := range bl.Items {
		if b.Name == build.Name {
			continue
		}

		if (b.Status.Phase == BuildPhaseInitialization || b.Status.Phase == BuildPhaseScheduling) &&
			b.CreationTimestamp.Before(&build.CreationTimestamp) {
			return true, &b
		}
	}

	return false, nil
}

// HasMatchingBuild visit all items in the list of builds and search for a scheduled build that matches the given build's dependencies.
// It returns the first matching build found regardless it may have any one more appropriate.
func (bl *BuildList) HasMatchingBuild(build *Build) (bool, *Build) {
	required := build.BuilderDependencies()
	if len(required) == 0 {
		return false, nil
	}
	requiredMap := make(map[string]int, len(required))
	for i, item := range required {
		requiredMap[item] = i
	}
	runtimeVersion := build.RuntimeVersion()

	var bestBuild Build
	bestBuildCommonDependencies := 0
buildLoop:
	for i := range bl.Items {
		b := bl.Items[i]
		if b.Name == build.Name || b.Status.IsFinished() {
			continue
		}
		bRuntimeVersion := b.RuntimeVersion()

		if *runtimeVersion != *bRuntimeVersion {
			continue
		}

		dependencies := b.BuilderDependencies()
		dependencyMap := make(map[string]int, len(dependencies))
		for i, item := range dependencies {
			dependencyMap[item] = i
		}

		// check if this build has any extra dependency. If it does, we can't use it
		for _, item := range dependencies {
			if _, ok := requiredMap[item]; !ok {
				continue buildLoop
			}
		}

		// now check how many common dependencies this build has
		missing := 0
		commonDependencies := 0
		for _, item := range required {
			if _, ok := dependencyMap[item]; !ok {
				missing++
			} else {
				commonDependencies++
			}
		}

		if commonDependencies == 0 {
			// no common dependencies
			continue
		}

		switch b.Status.Phase {
		case BuildPhasePending, BuildPhaseRunning:
			// handle suitable build that has started already
			return true, &b
		case BuildPhaseInitialization, BuildPhaseScheduling:
			// handle suitable scheduled build

			if missing == 0 {
				// seems like both builds require exactly the same list of dependencies
				// additionally check for the creation timestamp
				if compareBuilds(&b, build) < 0 {
					return true, &b
				}
			} else if commonDependencies > bestBuildCommonDependencies {
				bestBuildCommonDependencies = commonDependencies
				bestBuild = b
				continue
			}
		}
	}

	if bestBuildCommonDependencies > 0 {
		return true, &bestBuild
	}
	return false, nil
}

func compareBuilds(b1 *Build, b2 *Build) int {
	if b1.CreationTimestamp.Before(&b2.CreationTimestamp) {
		return -1
	}
	if b2.CreationTimestamp.Before(&b1.CreationTimestamp) {
		return 1
	}
	return strings.Compare(b1.Name, b2.Name)
}
