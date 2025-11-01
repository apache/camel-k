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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"

	cclient "github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/reference"
	"github.com/apache/camel-k/v2/pkg/util/uri"
	"github.com/spf13/cobra"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// newCmdBind --.
func newCmdBind(rootCmdOptions *RootCmdOptions) (*cobra.Command, *bindCmdOptions) {
	options := bindCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:               "bind [source] [sink] ...",
		Short:             "Bind Kubernetes resources, such as Kamelets, in an integration flow.",
		Long:              "Bind Kubernetes resources, such as Kamelets, in an integration flow. Endpoints are expected in the format \"[[apigroup/]version:]kind:[namespace/]name\" or plain Camel URIs.",
		PersistentPreRunE: decode(&options, options.Flags),
		PreRunE:           options.preRunE,
		RunE:              options.runE,
		Annotations:       make(map[string]string),
	}

	cmd.Flags().String("error-handler", "", `Add error handler (none|log|sink:<endpoint>). Sink endpoints are expected in the format "[[apigroup/]version:]kind:[namespace/]name", plain Camel URIs or Kamelet name.`)
	cmd.Flags().String("name", "", "Name for the binding")
	cmd.Flags().StringP("output", "o", "", "Output format. One of: json|yaml")
	cmd.Flags().StringArrayP("property", "p", nil, `Add a binding property in the form of "source.<key>=<value>", "sink.<key>=<value>", "error-handler.<key>=<value>" or "step-<n>.<key>=<value> where <n> is the step order starting from 1"`)
	cmd.Flags().Bool("skip-checks", false, "Do not verify the binding for compliance with Kamelets and other Kubernetes resources")
	cmd.Flags().StringArray("step", nil, `Add binding steps as Kubernetes resources. Endpoints are expected in the format "[[apigroup/]version:]kind:[namespace/]name", plain Camel URIs or Kamelet name.`)
	cmd.Flags().StringArrayP("trait", "t", nil, `Add a trait to the corresponding Integration.`)
	cmd.Flags().StringP("operator-id", "x", "camel-k", "Operator id selected to manage this Pipe.")
	cmd.Flags().StringArray("annotation", nil, "Add an annotation to the Pipe. E.g. \"--annotation my.company=hello\"")
	cmd.Flags().String("service-account", "", "The SA to use to run this binding")
	cmd.Flags().StringArrayP("dependency", "d", nil, `A dependency that should be included, e.g., "camel:mail" for a Camel component, "mvn:org.my:app:1.0" for a Maven dependency`)

	return &cmd, &options
}

const (
	sourceKey       = "source"
	sinkKey         = "sink"
	stepKeyPrefix   = "step-"
	errorHandlerKey = "error-handler"
)

type bindCmdOptions struct {
	*RootCmdOptions

	ErrorHandler   string   `mapstructure:"error-handler" yaml:",omitempty"`
	Name           string   `mapstructure:"name" yaml:",omitempty"`
	OutputFormat   string   `mapstructure:"output" yaml:",omitempty"`
	Properties     []string `mapstructure:"properties" yaml:",omitempty"`
	SkipChecks     bool     `mapstructure:"skip-checks" yaml:",omitempty"`
	Steps          []string `mapstructure:"steps" yaml:",omitempty"`
	Traits         []string `mapstructure:"traits" yaml:",omitempty"`
	OperatorID     string   `mapstructure:"operator-id" yaml:",omitempty"`
	Annotations    []string `mapstructure:"annotations" yaml:",omitempty"`
	ServiceAccount string   `mapstructure:"service-account" yaml:",omitempty"`
	Dependencies   []string `mapstructure:"dependencies" yaml:",omitempty"`
}

func (o *bindCmdOptions) preRunE(cmd *cobra.Command, args []string) error {
	if o.OutputFormat != "" {
		// let the command work in offline mode
		cmd.Annotations[offlineCommandLabel] = "true"
	}
	return o.preRun(cmd, args)
}

func (o *bindCmdOptions) runE(cmd *cobra.Command, args []string) error {
	if err := o.validate(cmd, args); err != nil {
		return err
	}
	if err := o.run(cmd, args); err != nil {
		fmt.Fprintln(cmd.OutOrStdout(), err.Error())
	}

	return nil
}

func (o *bindCmdOptions) validate(cmd *cobra.Command, args []string) error {
	if len(args) > 2 {
		return errors.New("too many arguments: expected source and sink")
	} else if len(args) < 2 {
		return errors.New("source or sink arguments are missing")
	}

	if o.OperatorID == "" {
		return fmt.Errorf("cannot use empty operator id")
	}

	for _, annotation := range o.Annotations {
		parts := strings.SplitN(annotation, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf(`invalid annotation specification %s. Expected "<annotationkey>=<annotationvalue>"`, annotation)
		}
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

	var client cclient.Client
	var err error
	if !isOfflineCommand(cmd) {
		client, err = o.GetCmdClient()
		if err != nil {
			return err
		}
	}
	catalog := trait.NewCatalog(client)

	return trait.ValidateTraits(catalog, extractTraitNames(o.Traits))
}

func (o *bindCmdOptions) run(cmd *cobra.Command, args []string) error {
	client, err := o.GetCmdClient()
	if err != nil {
		return err
	}

	source, err := o.decode(args[0], sourceKey)
	if err != nil {
		return err
	}

	sink, err := o.decode(args[1], sinkKey)
	if err != nil {
		return err
	}

	name := o.nameFor(source, sink)

	pipe := v1.Pipe{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: o.Namespace,
			Name:      name,
		},
		Spec: v1.PipeSpec{
			Source: source,
			Sink:   sink,
		},
	}

	if o.Dependencies != nil {
		pipe.Spec.Dependencies = o.Dependencies
	}

	if o.ErrorHandler != "" {
		if errorHandler, err := o.parseErrorHandler(); err == nil {
			pipe.Spec.ErrorHandler = errorHandler
		} else {
			return err
		}
	}

	if len(o.Steps) > 0 {
		pipe.Spec.Steps = make([]v1.Endpoint, 0)
		for idx, stepDesc := range o.Steps {
			stepIndex := idx + 1
			stepKey := fmt.Sprintf("%s%d", stepKeyPrefix, stepIndex)
			step, err := o.decode(stepDesc, stepKey)
			if err != nil {
				return err
			}
			pipe.Spec.Steps = append(pipe.Spec.Steps, step)
		}
	}

	if len(o.Traits) > 0 {
		if pipe.Annotations == nil {
			pipe.Annotations = make(map[string]string)
		}

		for _, t := range o.Traits {
			kv := strings.SplitN(t, "=", 2)
			if len(kv) != 2 {
				return fmt.Errorf("could not parse trait configuration %s, expected format 'trait.property=value'", t)
			}
			value := maybeBuildArrayNotation(pipe.Annotations[v1.TraitAnnotationPrefix+kv[0]], kv[1])
			pipe.Annotations[v1.TraitAnnotationPrefix+kv[0]] = value
		}
	}

	if o.ServiceAccount != "" {
		pipe.Spec.ServiceAccountName = o.ServiceAccount
	}

	// --operator-id={id} is a syntax sugar for '--annotation camel.apache.org/operator.id={id}'
	pipe.SetOperatorID(strings.TrimSpace(o.OperatorID))

	for _, annotation := range o.Annotations {
		parts := strings.SplitN(annotation, "=", 2)
		if len(parts) == 2 {
			pipe.Annotations[parts[0]] = parts[1]
		}
	}

	if o.OutputFormat != "" {
		return showPipeOutput(cmd, &pipe, o.OutputFormat, client.GetScheme())
	}

	replaced, err := kubernetes.ReplaceResource(o.Context, client, &pipe)
	if err != nil {
		return err
	}

	if !replaced {
		fmt.Fprintln(cmd.OutOrStdout(), `binding "`+name+`" created`)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), `binding "`+name+`" updated`)
	}
	return nil
}

// buildArrayNotation is used to build an array annotation to support traits array configuration
// for example, `-t camel.properties=a=1 -t camel.properties=b=2` would convert into annotation
// `camel.properties=[a=1,b=2]“.
func maybeBuildArrayNotation(array, value string) string {
	if array == "" {
		return value
	}
	// append
	if strings.HasPrefix(array, "[") && strings.HasSuffix(array, "]") {
		content := array[1:len(array)-1] + "," + value
		return "[" + content + "]"
	}
	// init the array notation
	return "[" + array + "," + value + "]"
}

func showPipeOutput(cmd *cobra.Command, binding *v1.Pipe, outputFormat string, scheme runtime.ObjectTyper) error {
	printer := printers.NewTypeSetter(scheme)
	printer.Delegate = &kubernetes.CLIPrinter{
		Format: outputFormat,
	}
	return printer.PrintObj(binding, cmd.OutOrStdout())
}

func (o *bindCmdOptions) parseErrorHandler() (*v1.ErrorHandlerSpec, error) {
	errHandlMap := make(map[string]interface{})
	errHandlType, errHandlValue, err := parseErrorHandlerByType(o.ErrorHandler)
	if err != nil {
		return nil, err
	}
	switch errHandlType {
	case "none":
		errHandlMap["none"] = nil
	case "log":
		errHandlMap["log"] = nil
	case "sink":
		sinkSpec, err := o.decode(errHandlValue, errorHandlerKey)
		if err != nil {
			return nil, err
		}
		errHandlMap["sink"] = map[string]interface{}{
			"endpoint": sinkSpec,
		}
	default:
		return nil, fmt.Errorf("invalid error handler type %s", o.ErrorHandler)
	}
	errHandlMarshalled, err := json.Marshal(&errHandlMap)
	if err != nil {
		return nil, err
	}
	return &v1.ErrorHandlerSpec{RawMessage: errHandlMarshalled}, nil
}

func parseErrorHandlerByType(value string) (string, string, error) {
	errHandlSplit := strings.SplitN(value, ":", 2)
	if (errHandlSplit[0] == "sink") && len(errHandlSplit) != 2 {
		return "", "", fmt.Errorf("invalid error handler syntax. Type %s needs a configuration (ie %s:value)",
			errHandlSplit[0], errHandlSplit[0])
	}
	if len(errHandlSplit) > 1 {
		return errHandlSplit[0], errHandlSplit[1], nil
	}
	return errHandlSplit[0], "", nil
}

func (o *bindCmdOptions) decode(res string, key string) (v1.Endpoint, error) {
	refConverter := reference.NewConverter(reference.KameletPrefix)
	endpoint := v1.Endpoint{}
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

func (o *bindCmdOptions) nameFor(source, sink v1.Endpoint) string {
	if o.Name != "" {
		return o.Name
	}
	sourcePart := o.nameForEndpoint(source)
	sinkPart := o.nameForEndpoint(sink)
	name := fmt.Sprintf("%s-to-%s", sourcePart, sinkPart)
	return kubernetes.SanitizeName(name)
}

func (o *bindCmdOptions) nameForEndpoint(endpoint v1.Endpoint) string {
	if endpoint.URI != nil {
		return uri.GetComponent(*endpoint.URI)
	}
	if endpoint.Ref != nil {
		return endpoint.Ref.Name
	}
	return ""
}

func (o *bindCmdOptions) asEndpointProperties(props map[string]string) (*v1.EndpointProperties, error) {
	if len(props) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(props)
	if err != nil {
		return nil, err
	}
	return &v1.EndpointProperties{
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
		return "", "", "", fmt.Errorf(`property %q does not follow format "[source|sink|error-handler|step-<n>].<key>=<value>"`, prop)
	}
	keyParts := strings.SplitN(parts[0], ".", 2)
	if len(keyParts) != 2 {
		return "", "", "", fmt.Errorf(`property key %q does not follow format "[source|sink|error-handler|step-<n>].<key>"`, parts[0])
	}
	isSource := keyParts[0] == sourceKey
	isSink := keyParts[0] == sinkKey
	isErrorHandler := keyParts[0] == errorHandlerKey
	isStep := strings.HasPrefix(keyParts[0], stepKeyPrefix)
	if !isSource && !isSink && !isStep && !isErrorHandler {
		return "", "", "", fmt.Errorf(`property key %q does not start with "source.", "sink.", "error-handler." or "step-<n>."`, parts[0])
	}
	return keyParts[0], keyParts[1], parts[1], nil
}

func (o *bindCmdOptions) checkCompliance(cmd *cobra.Command, endpoint v1.Endpoint) error {
	if endpoint.Ref != nil && endpoint.Ref.Kind == "Kamelet" {
		c, err := o.GetCmdClient()
		if err != nil {
			return err
		}
		key := client.ObjectKey{
			Namespace: endpoint.Ref.Namespace,
			Name:      endpoint.Ref.Name,
		}
		kamelet := v1.Kamelet{}
		if err := c.Get(o.Context, key, &kamelet); err != nil {
			if k8serrors.IsNotFound(err) {
				// Kamelet may be in the operator namespace, but we currently don't have a way to determine it: we just warn
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: Kamelet %q not found in namespace %q\n", key.Name, key.Namespace)
				return nil
			}
			return err
		}
	}
	return nil
}
