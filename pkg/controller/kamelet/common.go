package kamelet

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

func updateStatus(kamelet *v1alpha1.Kamelet) (*v1alpha1.Kamelet, error) {
	target := kamelet.DeepCopy()
	target.Status.Phase = v1alpha1.KameletPhaseReady
	target.Status.SetCondition(
		v1alpha1.KameletConditionReady,
		corev1.ConditionTrue,
		"",
		"",
	)
	if err := recomputeProperties(target); err != nil {
		return nil, err
	}
	return target, nil
}

func recomputeProperties(kamelet *v1alpha1.Kamelet) error {
	kamelet.Status.Properties = make([]v1alpha1.KameletProperty, 0, len(kamelet.Spec.Definition.Properties))
	propSet := make(map[string]bool)
	for k, v := range kamelet.Spec.Definition.Properties {
		if propSet[k] {
			continue
		}
		propSet[k] = true
		defValue := ""
		if v.Default != nil {
			var val interface{}
			if err := json.Unmarshal(v.Default.RawMessage, &val); err != nil {
				return errors.Wrapf(err, "cannot decode default value for property %q", k)
			}
			defValue = fmt.Sprintf("%v", val)
		}
		kamelet.Status.Properties = append(kamelet.Status.Properties, v1alpha1.KameletProperty{
			Name:    k,
			Default: defValue,
		})
	}
	sort.SliceStable(kamelet.Status.Properties, func(i, j int) bool {
		pi := kamelet.Status.Properties[i].Name
		pj := kamelet.Status.Properties[j].Name
		return pi < pj
	})
	return nil
}
