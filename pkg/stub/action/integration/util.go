package action

import (
	"fmt"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"

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

// CombinePropertiesAsMap --
func CombinePropertiesAsMap(context *v1alpha1.IntegrationContext, integration *v1alpha1.Integration) map[string]string {
	properties := make(map[string]string)
	if context != nil {
		// Add context properties first so integrations can
		// override it
		for _, p := range context.Spec.Properties {
			properties[p.Name] = p.Value
		}
	}

	if integration != nil {
		for _, p := range integration.Spec.Properties {
			properties[p.Name] = p.Value
		}
	}

	return properties
}
