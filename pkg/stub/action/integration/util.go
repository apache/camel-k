package action

import (
	"fmt"
	"strings"

	"github.com/apache/camel-k/pkg/util"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"k8s.io/api/core/v1"

	"github.com/pkg/errors"
)

// LookupContextForIntegration --
func LookupContextForIntegration(integration *v1alpha1.Integration) (*v1alpha1.IntegrationContext, error) {
	if integration.Spec.Context != "" {
		name := integration.Spec.Context
		ctx := v1alpha1.NewIntegrationContext(integration.Namespace, name)

		if err := sdk.Get(&ctx); err != nil {
			return nil, errors.Wrapf(err, "unable to find integration context %s, %s", ctx.Name, err)
		}

		return &ctx, nil
	}

	ctxList := v1alpha1.NewIntegrationContextList()
	if err := sdk.List(integration.Namespace, &ctxList); err != nil {
		return nil, err
	}

	for _, ctx := range ctxList.Items {
		if ctx.Labels["camel.apache.org/context.type"] == "platform" {
			ideps := len(integration.Spec.Dependencies)
			cdeps := len(ctx.Spec.Dependencies)

			if ideps != cdeps {
				continue
			}

			if util.StringSliceContains(ctx.Spec.Dependencies, integration.Spec.Dependencies) {
				return &ctx, nil
			}
		}
	}

	return nil, nil
}

// PropertiesString --
func PropertiesString(m map[string]string) string {
	properties := ""
	for k, v := range m {
		properties += fmt.Sprintf("%s=%s\n", k, v)
	}

	return properties
}

// EnvironmentAsEnvVarSlice --
func EnvironmentAsEnvVarSlice(m map[string]string) []v1.EnvVar {
	env := make([]v1.EnvVar, 0, len(m))

	for k, v := range m {
		env = append(env, v1.EnvVar{Name: k, Value: v})
	}

	return env
}

// CombineConfigurationAsMap --
func CombineConfigurationAsMap(configurationType string, context *v1alpha1.IntegrationContext, integration *v1alpha1.Integration) map[string]string {
	result := make(map[string]string)
	if context != nil {
		// Add context properties first so integrations can
		// override it
		for _, c := range context.Spec.Configuration {
			if c.Type == configurationType {
				pair := strings.Split(c.Value, "=")
				if len(pair) == 2 {
					result[pair[0]] = pair[1]
				}
			}
		}
	}

	if integration != nil {
		for _, c := range integration.Spec.Configuration {
			if c.Type == configurationType {
				pair := strings.Split(c.Value, "=")
				if len(pair) == 2 {
					result[pair[0]] = pair[1]
				}
			}
		}
	}

	return result
}

// CombineConfigurationAsSlice --
func CombineConfigurationAsSlice(configurationType string, context *v1alpha1.IntegrationContext, integration *v1alpha1.Integration) []string {
	result := make(map[string]bool, 0)
	if context != nil {
		// Add context properties first so integrations can
		// override it
		for _, c := range context.Spec.Configuration {
			if c.Type == configurationType {
				result[c.Value] = true
			}
		}
	}

	for _, c := range integration.Spec.Configuration {
		if c.Type == configurationType {
			result[c.Value] = true
		}
	}

	keys := make([]string, 0, len(result))
	for k := range result {
		keys = append(keys, k)
	}

	return keys
}
