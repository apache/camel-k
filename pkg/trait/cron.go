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
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/envvar"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/uri"
)

// The Cron trait can be used to customize the behaviour of periodic timer/cron based integrations.
//
// While normally an integration requires a pod to be always up and running, some periodic tasks, such as batch jobs,
// require to be activated at specific hours of the day or with a periodic delay of minutes.
// For such tasks, the cron trait can materialize the integration as a Kubernetes CronJob instead of a standard deployment,
// in order to save resources when the integration does not need to be executed.
//
// Integrations that start from the following components are evaluated by the cron trait: `timer`, `cron`, `quartz`.
//
// The rules for using a Kubernetes CronJob are the following:
// - `timer`: when periods can be written as cron expressions. E.g. `timer:tick?period=1m`.
// - `cron`, `quartz`: when the cron expression does not contain seconds (or the "seconds" part is set to 0). E.g.
//   `cron:tab?schedule=0/2+*+*+*+?` or `quartz:trigger?cron=0+0/2+*+*+*+?`.
//
// +camel-k:trait=cron
type cronTrait struct {
	BaseTrait `property:",squash"`
	// The CronJob schedule for the whole integration. If multiple routes are declared, they must have the same schedule for this
	// mechanism to work correctly.
	Schedule string `property:"schedule"`
	// A comma separated list of the Camel components that need to be customized in order for them to work when the schedule is triggered externally by Kubernetes.
	// A specific customizer is activated for each specified component. E.g. for the `timer` component, the `cron-timer` customizer is
	// activated (it's present in the `org.apache.camel.k:camel-k-runtime-cron` library).
	//
	// Supported components are currently: `cron`, `timer` and `quartz`.
	Components string `property:"components"`
	// Use the default Camel implementation of the `cron` endpoint (`quartz`) instead of trying to materialize the integration
	// as Kubernetes CronJob.
	Fallback *bool `property:"fallback"`
	// Automatically deploy the integration as CronJob when all routes are
	// either starting from a periodic consumer (only `cron`, `timer` and `quartz` are supported) or a passive consumer (e.g. `direct` is a passive consumer).
	//
	// It's required that all periodic consumers have the same period and it can be expressed as cron schedule (e.g. `1m` can be expressed as `0/1 * * * *`,
	// while `35m` or `50s` cannot).
	Auto     *bool `property:"auto"`
	deployer deployerTrait
}

var _ ControllerStrategySelector = &cronTrait{}

// cronInfo contains information about cron schedules present in the code
type cronInfo struct {
	components []string
	schedule   string
}

// cronExtractor extracts cron information from a Camel URI
type cronExtractor func(string) *cronInfo

const (
	genericCronComponent         = "cron"
	genericCronComponentFallback = "camel:quartz"
)

var (
	camelTimerPeriodMillis        = regexp.MustCompile(`^[0-9]+$`)
	camelTimerPeriodHumanReadable = regexp.MustCompile(`^(?:([0-9]+)h)?(?:([0-9]+)m)?(?:([0-9]+)s)?$`)

	supportedCamelComponents = map[string]cronExtractor{
		"timer":  timerToCronInfo,
		"quartz": quartzToCronInfo,
		"cron":   cronToCronInfo,
	}
)

func newCronTrait() *cronTrait {
	return &cronTrait{
		BaseTrait: newBaseTrait("cron"),
	}
}

func (t *cronTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !e.InPhase(v1.IntegrationKitPhaseReady, v1.IntegrationPhaseDeploying) {
		return false, nil
	}

	if t.Auto == nil || *t.Auto {
		globalCron, err := t.getGlobalCron(e)
		if err != nil {
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

		if t.Schedule == "" && t.Components == "" && t.Fallback == nil {
			// If there's at least a `cron` endpoint, add a fallback implementation
			fromURIs, err := t.getSourcesFromURIs(e)
			if err != nil {
				return false, err
			}
			for _, fromURI := range fromURIs {
				if uri.GetComponent(fromURI) == genericCronComponent {
					_true := true
					t.Fallback = &_true
					break
				}
			}

		}
	}

	dt := e.Catalog.GetTrait("deployer")
	if dt != nil {
		t.deployer = *dt.(*deployerTrait)
	}

	// Fallback strategy can be implemented in any other controller
	if t.Fallback != nil && *t.Fallback {
		return true, nil
	}

	// CronJob strategy requires common schedule
	strategy, err := e.DetermineControllerStrategy(t.ctx, t.client)
	if err != nil {
		return false, err
	}
	if strategy != ControllerStrategyCronJob {
		return false, nil
	}

	return t.Schedule != "", nil
}

func (t *cronTrait) Apply(e *Environment) error {
	if t.Fallback != nil && *t.Fallback {
		if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
			util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, genericCronComponentFallback)
		}
	} else {
		if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
			util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "mvn:org.apache.camel.k/camel-k-runtime-cron")
		} else if e.InPhase(v1.IntegrationKitPhaseReady, v1.IntegrationPhaseDeploying) {
			cronJob := t.getCronJobFor(e)
			maps := e.ComputeConfigMaps()

			e.Resources.AddAll(maps)
			e.Resources.Add(cronJob)

			envvar.SetVal(&e.EnvVars, "CAMEL_K_CRON_OVERRIDE", t.Components)
		}
	}
	return nil
}

func (t *cronTrait) getCronJobFor(e *Environment) *v1beta1.CronJob {
	labels := map[string]string{
		"camel.apache.org/integration": e.Integration.Name,
	}

	annotations := make(map[string]string)

	// Copy annotations from the integration resource
	if e.Integration.Annotations != nil {
		for k, v := range FilterTransferableAnnotations(e.Integration.Annotations) {
			annotations[k] = v
		}
	}

	// Resolve registry host names when used
	annotations["alpha.image.policy.openshift.io/resolve-names"] = "*"

	cronjob := v1beta1.CronJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CronJob",
			APIVersion: v1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        e.Integration.Name,
			Namespace:   e.Integration.Namespace,
			Labels:      labels,
			Annotations: e.Integration.Annotations,
		},
		Spec: v1beta1.CronJobSpec{
			Schedule: t.Schedule,
			JobTemplate: v1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels:      labels,
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

// SelectControllerStrategy can be used to check if a CronJob can be generated given the integration and trait settings
func (t *cronTrait) SelectControllerStrategy(e *Environment) (*ControllerStrategy, error) {
	cronStrategy := ControllerStrategyCronJob
	if t.Enabled != nil && !*t.Enabled {
		return nil, nil
	}
	if t.Fallback != nil && *t.Fallback {
		return nil, nil
	}
	if t.Schedule != "" {
		return &cronStrategy, nil
	}
	if t.Auto == nil || *t.Auto {
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
	if sources, err = kubernetes.ResolveIntegrationSources(t.ctx, t.client, e.Integration, e.Resources); err != nil {
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
// nolint: gocritic
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
	} else if camelTimerPeriodHumanReadable.MatchString(periodStr) {
		res := camelTimerPeriodHumanReadable.FindStringSubmatch(periodStr)
		if len(res) == 4 {
			period = 0
			if res[1] != "" { // hours
				period += checkedStringToUint64(res[1]) * 3600000
			}
			if res[2] != "" { // minutes
				period += checkedStringToUint64(res[2]) * 60000
			}
			if res[3] != "" { // seconds
				period += checkedStringToUint64(res[3]) * 1000
			}
		}
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

// quartzToCronInfo converts a quartz endpoint to a Kubernetes cron schedule
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

// cronToCronInfo converts a cron endpoint to a Kubernetes cron schedule
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
