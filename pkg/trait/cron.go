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

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/label"
	"github.com/apache/camel-k/pkg/util/uri"
)

type cronTrait struct {
	BaseTrait
	v1.CronTrait `property:",squash"`
}

var _ ControllerStrategySelector = &cronTrait{}

// cronInfo contains information about cron schedules present in the code.
type cronInfo struct {
	components []string
	schedule   string
}

// cronExtractor extracts cron information from a Camel URI.
type cronExtractor func(string) *cronInfo

const (
	genericCronComponent               = "cron"
	genericCronComponentFallbackScheme = "quartz"
)

var (
	camelTimerPeriodMillis = regexp.MustCompile(`^[0-9]+$`)

	supportedCamelComponents = map[string]cronExtractor{
		"timer":  timerToCronInfo,
		"quartz": quartzToCronInfo,
		"cron":   cronToCronInfo,
	}
)

func newCronTrait() Trait {
	return &cronTrait{
		BaseTrait: NewBaseTrait("cron", 1000),
	}
}

func (t *cronTrait) Configure(e *Environment) (bool, error) {
	if !pointer.BoolDeref(t.Enabled, true) {
		e.Integration.Status.SetCondition(
			v1.IntegrationConditionCronJobAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionCronJobNotAvailableReason,
			"explicitly disabled",
		)

		return false, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !e.IntegrationInRunningPhases() {
		return false, nil
	}

	if _, ok := e.CamelCatalog.Runtime.Capabilities[v1.CapabilityCron]; !ok {
		e.Integration.Status.SetCondition(
			v1.IntegrationConditionCronJobAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionCronJobNotAvailableReason,
			"the runtime provider %s does not declare 'cron' capability",
		)

		return false, nil
	}

	if pointer.BoolDeref(t.Auto, true) {
		globalCron, err := t.getGlobalCron(e)
		if err != nil {
			e.Integration.Status.SetErrorCondition(
				v1.IntegrationConditionCronJobAvailable,
				v1.IntegrationConditionCronJobNotAvailableReason,
				err,
			)
			return false, err
		}

		if t.Schedule == "" && globalCron != nil {
			t.Schedule = globalCron.schedule
		}

		if globalCron != nil {
			configuredComponents := strings.FieldsFunc(t.Components, func(c rune) bool { return c == ',' })
			for _, c := range globalCron.components {
				util.StringSliceUniqueAdd(&configuredComponents, c)
			}
			t.Components = strings.Join(configuredComponents, ",")
		}

		if t.ConcurrencyPolicy == "" {
			t.ConcurrencyPolicy = string(batchv1.ForbidConcurrent)
		}

		if (t.Schedule == "" && t.Components == "") && t.Fallback == nil {
			// If there's at least a `cron` endpoint, add a fallback implementation
			fromURIs, err := t.getSourcesFromURIs(e)
			if err != nil {
				return false, err
			}
			for _, fromURI := range fromURIs {
				if uri.GetComponent(fromURI) == genericCronComponent {
					t.Fallback = pointer.Bool(true)
					break
				}
			}
		}
	}

	// Fallback strategy can be implemented in any other controller
	if pointer.BoolDeref(t.Fallback, false) {
		if e.IntegrationInPhase(v1.IntegrationPhaseDeploying) {
			e.Integration.Status.SetCondition(
				v1.IntegrationConditionCronJobAvailable,
				corev1.ConditionFalse,
				v1.IntegrationConditionCronJobNotAvailableReason,
				"fallback strategy selected",
			)
		}
		return true, nil
	}

	// CronJob strategy requires common schedule
	strategy, err := e.DetermineControllerStrategy()
	if err != nil {
		e.Integration.Status.SetErrorCondition(
			v1.IntegrationConditionCronJobAvailable,
			v1.IntegrationConditionCronJobNotAvailableReason,
			err,
		)
		return false, err
	}
	if strategy != ControllerStrategyCronJob {
		if e.IntegrationInPhase(v1.IntegrationPhaseDeploying) {
			e.Integration.Status.SetCondition(
				v1.IntegrationConditionCronJobAvailable,
				corev1.ConditionFalse,
				v1.IntegrationConditionCronJobNotAvailableReason,
				fmt.Sprintf("different controller strategy used (%s)", string(strategy)),
			)
		}
		return false, nil
	}

	return t.Schedule != "", nil
}

func (t *cronTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		util.StringSliceUniqueAdd(&e.Integration.Status.Capabilities, v1.CapabilityCron)

		if pointer.BoolDeref(t.Fallback, false) {
			fallbackArtifact := e.CamelCatalog.GetArtifactByScheme(genericCronComponentFallbackScheme)
			if fallbackArtifact == nil {
				return fmt.Errorf("no fallback artifact for scheme %q has been found in camel catalog", genericCronComponentFallbackScheme)
			}
			util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, fallbackArtifact.GetDependencyID())
			util.StringSliceUniqueConcat(&e.Integration.Status.Dependencies, fallbackArtifact.GetConsumerDependencyIDs(genericCronComponentFallbackScheme))
		}
	}

	if !pointer.BoolDeref(t.Fallback, false) && e.IntegrationInRunningPhases() {
		if e.ApplicationProperties == nil {
			e.ApplicationProperties = make(map[string]string)
		}

		e.ApplicationProperties["camel.main.duration-max-idle-seconds"] = "5"
		e.ApplicationProperties["loader.interceptor.cron.overridable-components"] = t.Components
		e.Interceptors = append(e.Interceptors, "cron")

		cronJob := t.getCronJobFor(e)
		e.Resources.Add(cronJob)

		e.Integration.Status.SetCondition(
			v1.IntegrationConditionCronJobAvailable,
			corev1.ConditionTrue,
			v1.IntegrationConditionCronJobAvailableReason,
			fmt.Sprintf("CronJob name is %s", cronJob.Name))
	}

	return nil
}

func (t *cronTrait) getCronJobFor(e *Environment) *batchv1.CronJob {
	annotations := make(map[string]string)
	if e.Integration.Annotations != nil {
		for k, v := range filterTransferableAnnotations(e.Integration.Annotations) {
			annotations[k] = v
		}
	}

	activeDeadline := int64(60)
	if t.ActiveDeadlineSeconds != nil {
		activeDeadline = *t.ActiveDeadlineSeconds
	}

	backoffLimit := int32(2)
	if t.BackoffLimit != nil {
		backoffLimit = *t.BackoffLimit
	}

	cronjob := batchv1.CronJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CronJob",
			APIVersion: batchv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Integration.Name,
			Namespace: e.Integration.Namespace,
			Labels: map[string]string{
				v1.IntegrationLabel: e.Integration.Name,
			},
			Annotations: e.Integration.Annotations,
		},
		Spec: batchv1.CronJobSpec{
			Schedule:                t.Schedule,
			ConcurrencyPolicy:       batchv1.ConcurrencyPolicy(t.ConcurrencyPolicy),
			StartingDeadlineSeconds: t.StartingDeadlineSeconds,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					ActiveDeadlineSeconds: &activeDeadline,
					BackoffLimit:          &backoffLimit,
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels:      label.AddLabels(e.Integration.Name),
							Annotations: annotations,
						},
						Spec: corev1.PodSpec{
							ServiceAccountName: e.Integration.Spec.ServiceAccountName,
							RestartPolicy:      corev1.RestartPolicyNever,
						},
					},
				},
			},
		},
	}

	return &cronjob
}

// SelectControllerStrategy can be used to check if a CronJob can be generated given the integration and trait settings.
func (t *cronTrait) SelectControllerStrategy(e *Environment) (*ControllerStrategy, error) {
	cronStrategy := ControllerStrategyCronJob
	if !pointer.BoolDeref(t.Enabled, true) {
		return nil, nil
	}
	if pointer.BoolDeref(t.Fallback, false) {
		return nil, nil
	}
	if t.Schedule != "" {
		return &cronStrategy, nil
	}
	if pointer.BoolDeref(t.Auto, true) {
		globalCron, err := t.getGlobalCron(e)
		if err == nil && globalCron != nil {
			return &cronStrategy, nil
		}
	}
	return nil, nil
}

func (t *cronTrait) ControllerStrategySelectorOrder() int {
	return 1000
}

// Gathering cron information

func newCronInfo() *cronInfo {
	return &cronInfo{}
}

func (c *cronInfo) withComponents(components ...string) *cronInfo {
	for _, comp := range components {
		util.StringSliceUniqueAdd(&c.components, comp)
	}
	return c
}

func (c *cronInfo) withSchedule(schedule string) *cronInfo {
	c.schedule = schedule
	return c
}

func (t *cronTrait) getGlobalCron(e *Environment) (*cronInfo, error) {
	fromURIs, err := t.getSourcesFromURIs(e)
	if err != nil {
		return nil, err
	}

	passiveComponents := make(map[string]bool)
	e.CamelCatalog.VisitSchemes(func(id string, scheme v1.CamelScheme) bool {
		if scheme.Passive {
			passiveComponents[id] = true
		}
		return true
	})

	var cron []string
	for _, from := range fromURIs {
		comp := uri.GetComponent(from)
		if supportedCamelComponents[comp] != nil {
			cron = append(cron, from)
		} else if !passiveComponents[comp] {
			return nil, nil
		}
	}

	globalCron := getCronForURIs(cron)
	return globalCron, nil
}

func (t *cronTrait) getSourcesFromURIs(e *Environment) ([]string, error) {
	var sources []v1.SourceSpec
	var err error
	if sources, err = kubernetes.ResolveIntegrationSources(e.Ctx, t.Client, e.Integration, e.Resources); err != nil {
		return nil, err
	}
	meta := metadata.ExtractAll(e.CamelCatalog, sources)
	return meta.FromURIs, nil
}

func getCronForURIs(camelURIs []string) (globalCron *cronInfo) {
	for _, camelURI := range camelURIs {
		cr := getCronForURI(camelURI)
		if cr == nil {
			return nil
		}
		if globalCron == nil {
			globalCron = cr
		} else {
			if !cronEquivalent(globalCron.schedule, cr.schedule) {
				return nil
			}
			globalCron = globalCron.withComponents(cr.components...)
		}
	}
	return globalCron
}

func getCronForURI(camelURI string) *cronInfo {
	comp := uri.GetComponent(camelURI)
	extractor := supportedCamelComponents[comp]
	return extractor(camelURI)
}

// Specific extractors

// timerToCronInfo converts a timer endpoint to a Kubernetes cron schedule

func timerToCronInfo(camelURI string) *cronInfo {
	if uri.GetQueryParameter(camelURI, "delay") != "" ||
		uri.GetQueryParameter(camelURI, "repeatCount") != "" ||
		uri.GetQueryParameter(camelURI, "time") != "" {
		return nil
	}
	periodStr := uri.GetQueryParameter(camelURI, "period")
	var period uint64
	if camelTimerPeriodMillis.MatchString(periodStr) {
		period = checkedStringToUint64(periodStr)
	} else {
		return nil
	}

	if period == 0 || period%1000 != 0 {
		return nil
	}
	seconds := period / 1000

	if seconds%3600 == 0 {
		hours := seconds / 3600
		if hours == 24 {
			return newCronInfo().withComponents("timer").withSchedule("0 0 * * ?")
		} else if hours < 24 && 24%hours == 0 {
			return newCronInfo().withComponents("timer").withSchedule(fmt.Sprintf("0 0/%d * * ?", hours))
		}
	} else if seconds%60 == 0 {
		minutes := seconds / 60
		if minutes < 60 && 60%minutes == 0 {
			return newCronInfo().withComponents("timer").withSchedule(fmt.Sprintf("0/%d * * * ?", minutes))
		}
	}
	return nil
}

// quartzToCronInfo converts a quartz endpoint to a Kubernetes cron schedule.
func quartzToCronInfo(camelURI string) *cronInfo {
	if uri.GetQueryParameter(camelURI, "fireNow") != "" ||
		uri.GetQueryParameter(camelURI, "customCalendar") != "" ||
		uri.GetQueryParameter(camelURI, "startDelayedSeconds") != "" {
		return nil
	}
	// Quartz URI has 6 or 7 components instead of the 5 expected by Kubernetes (starts with seconds, ends with year).
	cron := uri.GetQueryParameter(camelURI, "cron")
	normalized := toKubernetesCronSchedule(cron)
	if normalized != "" {
		return newCronInfo().withComponents("quartz").withSchedule(normalized)
	}

	return nil
}

// cronToCronInfo converts a cron endpoint to a Kubernetes cron schedule.
func cronToCronInfo(camelURI string) *cronInfo {
	// Camel cron URIs have 5 to 7 components.
	schedule := uri.GetQueryParameter(camelURI, "schedule")
	normalized := toKubernetesCronSchedule(schedule)
	if normalized != "" {
		return newCronInfo().withComponents("cron").withSchedule(normalized)
	}

	return nil
}

// Utility

func cronEquivalent(cron1, cron2 string) bool {
	// best effort to determine if two crons are equivalent
	cron1 = strings.ReplaceAll(cron1, "?", "*")
	cron2 = strings.ReplaceAll(cron2, "?", "*")
	return cron1 == cron2
}

func toKubernetesCronSchedule(cron string) string {
	parts := strings.Split(cron, " ")

	if len(parts) > 5 {
		// drop seconds if they can be ignored
		if parts[0] == "0" {
			parts = parts[1:]
		} else {
			return ""
		}
	}

	if len(parts) == 6 && (parts[5] == "*" || parts[5] == "?") {
		// drop year if present
		parts = parts[0:5]
	}

	if len(parts) == 5 {
		return strings.Join(parts, " ")
	}
	return ""
}

func checkedStringToUint64(str string) uint64 {
	res, err := strconv.ParseUint(str, 10, 0)
	if err != nil {
		panic(err)
	}
	return res
}
