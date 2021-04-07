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

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/reference"
	"github.com/apache/camel-k/pkg/util/uri"
	"github.com/spf13/cobra"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// newCmdBind --
func newCmdBind(rootCmdOptions *RootCmdOptions) (*cobra.Command, *bindCmdOptions) {
	options := bindCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:     "bind [source] [sink] ...",
		Short:   "Bind Kubernetes resources, such as Kamelets, in an integration flow. Endpoints are expected in the format \"[[apigroup/]version:]kind:[namespace/]name\" or plain Camel URIs.",
		PreRunE: decode(&options),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.validate(cmd, args); err != nil {
				return err
			}
			if err := options.run(args); err != nil {
				fmt.Println(err.Error())
			}

			return nil
		},
	}

	cmd.Flags().String("name", "", "Name for the binding")
	cmd.Flags().StringP("output", "o", "", "Output format. One of: json|yaml")
	cmd.Flags().StringArrayP("property", "p", nil, `Add a binding property in the form of "source.<key>=<value>", "sink.<key>=<value>" or "step-<n>.<key>=<value>"`)
	cmd.Flags().Bool("skip-checks", false, "Do not verify the binding for compliance with Kamelets and other Kubernetes resources")
	cmd.Flags().StringArray("step", nil, `Add binding steps as Kubernetes resources, such as Kamelets. Endpoints are expected in the format "[[apigroup/]version:]kind:[namespace/]name" or plain Camel URIs.`)

	return &cmd, &options
}

const (
	sourceKey     = "source"
	sinkKey       = "sink"
	stepKeyPrefix = "step-"
)

type bindCmdOptions struct {
	*RootCmdOptions
	Name         string   `mapstructure:"name" yaml:",omitempty"`
	OutputFormat string   `mapstructure:"output" yaml:",omitempty"`
	Properties   []string `mapstructure:"properties" yaml:",omitempty"`
	SkipChecks   bool     `mapstructure:"skip-checks" yaml:",omitempty"`
	Steps        []string `mapstructure:"steps" yaml:",omitempty"`
}

func (o *bindCmdOptions) validate(cmd *cobra.Command, args []string) error {
	if len(args) > 2 {
		return errors.New("too many arguments: expected source and sink")
	} else if len(args) < 2 {
		return errors.New("source or sink arguments are missing")
	}

	for _, p := range o.Properties {
		if _, _, _, err := o.parseProperty(p); err != nil {
			return err
		}
	}

	if !o.SkipChecks {
		source, err := o.decode(args[0], sourceKey)
		if err != nil {
			return err
		}
		if err := o.checkCompliance(cmd, source); err != nil {
			return err
		}

		sink, err := o.decode(args[1], sinkKey)
		if err != nil {
			return err
		}
		if err := o.checkCompliance(cmd, sink); err != nil {
			return err
		}

		for idx, stepDesc := range o.Steps {
			stepKey := fmt.Sprintf("%s%d", stepKeyPrefix, idx)
			step, err := o.decode(stepDesc, stepKey)
			if err != nil {
				return err
			}
			if err := o.checkCompliance(cmd, step); err != nil {
				return err
			}
		}
	}

	return nil
}

func (o *bindCmdOptions) run(args []string) error {
	source, err := o.decode(args[0], sourceKey)
	if err != nil {
		return err
	}

	sink, err := o.decode(args[1], sinkKey)
	if err != nil {
		return err
	}

	name := o.nameFor(source, sink)

	binding := v1alpha1.KameletBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: o.Namespace,
			Name:      name,
		},
		Spec: v1alpha1.KameletBindingSpec{
			Source: source,
			Sink:   sink,
		},
	}

	if len(o.Steps) > 0 {
		binding.Spec.Steps = make([]v1alpha1.Endpoint, 0)
		for idx, stepDesc := range o.Steps {
			stepKey := fmt.Sprintf("%s%d", stepKeyPrefix, idx)
			step, err := o.decode(stepDesc, stepKey)
			if err != nil {
				return err
			}
			binding.Spec.Steps = append(binding.Spec.Steps, step)
		}
	}

	switch o.OutputFormat {
	case "":
		// continue..
	case "yaml":
		data, err := kubernetes.ToYAML(&binding)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
		return nil

	case "json":
		data, err := kubernetes.ToJSON(&binding)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
		return nil

	default:
		return fmt.Errorf("invalid output format option '%s', should be one of: yaml|json", o.OutputFormat)
	}

	client, err := o.GetCmdClient()
	if err != nil {
		return err
	}

	existed := false
	err = client.Create(o.Context, &binding)
	if err != nil && k8serrors.IsAlreadyExists(err) {
		existed = true
		err = kubernetes.ReplaceResource(o.Context, client, &binding)
	}
	if err != nil {
		return err
	}

	if !existed {
		fmt.Printf("kamelet binding \"%s\" created\n", name)
	} else {
		fmt.Printf("kamelet binding \"%s\" updated\n", name)
	}
	return nil
}

func (o *bindCmdOptions) decode(res string, key string) (v1alpha1.Endpoint, error) {
	refConverter := reference.NewConverter(reference.KameletPrefix)
	endpoint := v1alpha1.Endpoint{}
	explicitProps := o.getProperties(key)
	props, err := o.asEndpointProperties(explicitProps)
	if err != nil {
		return endpoint, err
	}
	endpoint.Properties = props

	ref, err := refConverter.FromString(res)
	if err != nil {
		if uri.HasCamelURIFormat(res) {
			endpoint.URI = &res
			return endpoint, nil
		}
		return endpoint, err
	}
	endpoint.Ref = &ref
	if endpoint.Ref.Namespace == "" {
		endpoint.Ref.Namespace = o.Namespace
	}
	embeddedProps, err := refConverter.PropertiesFromString(res)
	if err != nil {
		return endpoint, err
	}
	if len(embeddedProps) > 0 {
		allProps := make(map[string]string)
		for k, v := range explicitProps {
			allProps[k] = v
		}
		for k, v := range embeddedProps {
			allProps[k] = v
		}

		props, err := o.asEndpointProperties(allProps)
		if err != nil {
			return endpoint, err
		}
		endpoint.Properties = props
	}

	return endpoint, nil
}

func (o *bindCmdOptions) nameFor(source, sink v1alpha1.Endpoint) string {
	if o.Name != "" {
		return o.Name
	}
	sourcePart := o.nameForEndpoint(source)
	sinkPart := o.nameForEndpoint(sink)
	name := fmt.Sprintf("%s-to-%s", sourcePart, sinkPart)
	return kubernetes.SanitizeName(name)
}

func (o *bindCmdOptions) nameForEndpoint(endpoint v1alpha1.Endpoint) string {
	if endpoint.URI != nil {
		return uri.GetComponent(*endpoint.URI)
	}
	if endpoint.Ref != nil {
		return endpoint.Ref.Name
	}
	return ""
}

func (o *bindCmdOptions) asEndpointProperties(props map[string]string) (*v1alpha1.EndpointProperties, error) {
	if len(props) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(props)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.EndpointProperties{
		RawMessage: v1.RawMessage(data),
	}, nil
}

func (o *bindCmdOptions) getProperties(refType string) map[string]string {
	props := make(map[string]string)
	for _, p := range o.Properties {
		tp, k, v, err := o.parseProperty(p)
		if err != nil {
			continue
		}
		if tp == refType {
			props[k] = v
		}
	}
	return props
}

func (o *bindCmdOptions) parseProperty(prop string) (string, string, string, error) {
	parts := strings.SplitN(prop, "=", 2)
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf(`property %q does not follow format "[source|sink|step-<n>].<key>=<value>"`, prop)
	}
	keyParts := strings.SplitN(parts[0], ".", 2)
	if len(keyParts) != 2 {
		return "", "", "", fmt.Errorf(`property key %q does not follow format "[source|sink|step-<n>].<key>"`, parts[0])
	}
	isSource := keyParts[0] == sourceKey
	isSink := keyParts[0] == sinkKey
	isStep := strings.HasPrefix(keyParts[0], stepKeyPrefix)
	if !isSource && !isSink && !isStep {
		return "", "", "", fmt.Errorf(`property key %q does not start with "source.", "sink." or "step-<n>."`, parts[0])
	}
	return keyParts[0], keyParts[1], parts[1], nil
}

func (o *bindCmdOptions) checkCompliance(cmd *cobra.Command, endpoint v1alpha1.Endpoint) error {
	if endpoint.Ref != nil && endpoint.Ref.Kind == "Kamelet" {
		c, err := o.GetCmdClient()
		if err != nil {
			return err
		}
		key := client.ObjectKey{
			Namespace: endpoint.Ref.Namespace,
			Name:      endpoint.Ref.Name,
		}
		kamelet := v1alpha1.Kamelet{}
		if err := c.Get(o.Context, key, &kamelet); err != nil {
			if k8serrors.IsNotFound(err) {
				// Kamelet may be in the operator namespace, but we currently don't have a way to determine it: we just warn
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: Kamelet %q not found in namespace %q\n", key.Name, key.Namespace)
				return nil
			}
			return err
		}
		if kamelet.Spec.Definition != nil && len(kamelet.Spec.Definition.Required) > 0 {
			pMap, err := endpoint.Properties.GetPropertyMap()
			if err != nil {
				return err
			}
			for _, reqProp := range kamelet.Spec.Definition.Required {
				found := false
				if endpoint.Properties != nil {
					if _, contains := pMap[reqProp]; contains {
						found = true
					}
				}
				if !found {
					return fmt.Errorf("binding is missing required property %q for Kamelet %q", reqProp, key.Name)
				}
			}
		}
	}
	return nil
}
