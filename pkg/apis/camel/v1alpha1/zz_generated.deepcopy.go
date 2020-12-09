// +build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"encoding/json"
	"github.com/apache/camel-k/pkg/apis/camel/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthorizationSpec) DeepCopyInto(out *AuthorizationSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthorizationSpec.
func (in *AuthorizationSpec) DeepCopy() *AuthorizationSpec {
	if in == nil {
		return nil
	}
	out := new(AuthorizationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Endpoint) DeepCopyInto(out *Endpoint) {
	*out = *in
	if in.Ref != nil {
		in, out := &in.Ref, &out.Ref
		*out = new(corev1.ObjectReference)
		**out = **in
	}
	if in.URI != nil {
		in, out := &in.URI, &out.URI
		*out = new(string)
		**out = **in
	}
	if in.Properties != nil {
		in, out := &in.Properties, &out.Properties
		*out = new(EndpointProperties)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Endpoint.
func (in *Endpoint) DeepCopy() *Endpoint {
	if in == nil {
		return nil
	}
	out := new(Endpoint)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EndpointProperties) DeepCopyInto(out *EndpointProperties) {
	*out = *in
	if in.RawMessage != nil {
		in, out := &in.RawMessage, &out.RawMessage
		*out = make(v1.RawMessage, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EndpointProperties.
func (in *EndpointProperties) DeepCopy() *EndpointProperties {
	if in == nil {
		return nil
	}
	out := new(EndpointProperties)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EventTypeSpec) DeepCopyInto(out *EventTypeSpec) {
	*out = *in
	if in.Schema != nil {
		in, out := &in.Schema, &out.Schema
		*out = new(JSONSchemaProps)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EventTypeSpec.
func (in *EventTypeSpec) DeepCopy() *EventTypeSpec {
	if in == nil {
		return nil
	}
	out := new(EventTypeSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalDocumentation) DeepCopyInto(out *ExternalDocumentation) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalDocumentation.
func (in *ExternalDocumentation) DeepCopy() *ExternalDocumentation {
	if in == nil {
		return nil
	}
	out := new(ExternalDocumentation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JSON) DeepCopyInto(out *JSON) {
	*out = *in
	if in.RawMessage != nil {
		in, out := &in.RawMessage, &out.RawMessage
		*out = make(json.RawMessage, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JSON.
func (in *JSON) DeepCopy() *JSON {
	if in == nil {
		return nil
	}
	out := new(JSON)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in JSONSchemaDefinitions) DeepCopyInto(out *JSONSchemaDefinitions) {
	{
		in := &in
		*out = make(JSONSchemaDefinitions, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JSONSchemaDefinitions.
func (in JSONSchemaDefinitions) DeepCopy() JSONSchemaDefinitions {
	if in == nil {
		return nil
	}
	out := new(JSONSchemaDefinitions)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in JSONSchemaDependencies) DeepCopyInto(out *JSONSchemaDependencies) {
	{
		in := &in
		*out = make(JSONSchemaDependencies, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make([]string, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JSONSchemaDependencies.
func (in JSONSchemaDependencies) DeepCopy() JSONSchemaDependencies {
	if in == nil {
		return nil
	}
	out := new(JSONSchemaDependencies)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JSONSchemaProps) DeepCopyInto(out *JSONSchemaProps) {
	*out = *in
	if in.Ref != nil {
		in, out := &in.Ref, &out.Ref
		*out = new(string)
		**out = **in
	}
	if in.Default != nil {
		in, out := &in.Default, &out.Default
		*out = new(JSON)
		(*in).DeepCopyInto(*out)
	}
	if in.Maximum != nil {
		in, out := &in.Maximum, &out.Maximum
		*out = new(json.Number)
		**out = **in
	}
	if in.Minimum != nil {
		in, out := &in.Minimum, &out.Minimum
		*out = new(json.Number)
		**out = **in
	}
	if in.MaxLength != nil {
		in, out := &in.MaxLength, &out.MaxLength
		*out = new(int64)
		**out = **in
	}
	if in.MinLength != nil {
		in, out := &in.MinLength, &out.MinLength
		*out = new(int64)
		**out = **in
	}
	if in.MaxItems != nil {
		in, out := &in.MaxItems, &out.MaxItems
		*out = new(int64)
		**out = **in
	}
	if in.MinItems != nil {
		in, out := &in.MinItems, &out.MinItems
		*out = new(int64)
		**out = **in
	}
	if in.MultipleOf != nil {
		in, out := &in.MultipleOf, &out.MultipleOf
		*out = new(json.Number)
		**out = **in
	}
	if in.Enum != nil {
		in, out := &in.Enum, &out.Enum
		*out = make([]*JSON, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(JSON)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	if in.MaxProperties != nil {
		in, out := &in.MaxProperties, &out.MaxProperties
		*out = new(int64)
		**out = **in
	}
	if in.MinProperties != nil {
		in, out := &in.MinProperties, &out.MinProperties
		*out = new(int64)
		**out = **in
	}
	if in.Required != nil {
		in, out := &in.Required, &out.Required
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = new(JSONSchemaProps)
		(*in).DeepCopyInto(*out)
	}
	if in.AllOf != nil {
		in, out := &in.AllOf, &out.AllOf
		*out = make([]JSONSchemaProps, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.OneOf != nil {
		in, out := &in.OneOf, &out.OneOf
		*out = make([]JSONSchemaProps, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.AnyOf != nil {
		in, out := &in.AnyOf, &out.AnyOf
		*out = make([]JSONSchemaProps, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Not != nil {
		in, out := &in.Not, &out.Not
		*out = new(JSONSchemaProps)
		(*in).DeepCopyInto(*out)
	}
	if in.Properties != nil {
		in, out := &in.Properties, &out.Properties
		*out = make(map[string]JSONSchemaProps, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.AdditionalProperties != nil {
		in, out := &in.AdditionalProperties, &out.AdditionalProperties
		*out = new(bool)
		**out = **in
	}
	if in.PatternProperties != nil {
		in, out := &in.PatternProperties, &out.PatternProperties
		*out = make(map[string]JSONSchemaProps, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.Dependencies != nil {
		in, out := &in.Dependencies, &out.Dependencies
		*out = make(JSONSchemaDependencies, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make([]string, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
	if in.AdditionalItems != nil {
		in, out := &in.AdditionalItems, &out.AdditionalItems
		*out = new(bool)
		**out = **in
	}
	if in.Definitions != nil {
		in, out := &in.Definitions, &out.Definitions
		*out = make(JSONSchemaDefinitions, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.ExternalDocs != nil {
		in, out := &in.ExternalDocs, &out.ExternalDocs
		*out = new(ExternalDocumentation)
		**out = **in
	}
	if in.Example != nil {
		in, out := &in.Example, &out.Example
		*out = new(JSON)
		(*in).DeepCopyInto(*out)
	}
	if in.XPreserveUnknownFields != nil {
		in, out := &in.XPreserveUnknownFields, &out.XPreserveUnknownFields
		*out = new(bool)
		**out = **in
	}
	if in.XListMapKeys != nil {
		in, out := &in.XListMapKeys, &out.XListMapKeys
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.XListType != nil {
		in, out := &in.XListType, &out.XListType
		*out = new(string)
		**out = **in
	}
	if in.XMapType != nil {
		in, out := &in.XMapType, &out.XMapType
		*out = new(string)
		**out = **in
	}
	if in.XDescriptors != nil {
		in, out := &in.XDescriptors, &out.XDescriptors
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JSONSchemaProps.
func (in *JSONSchemaProps) DeepCopy() *JSONSchemaProps {
	if in == nil {
		return nil
	}
	out := new(JSONSchemaProps)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Kamelet) DeepCopyInto(out *Kamelet) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Kamelet.
func (in *Kamelet) DeepCopy() *Kamelet {
	if in == nil {
		return nil
	}
	out := new(Kamelet)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Kamelet) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KameletBinding) DeepCopyInto(out *KameletBinding) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KameletBinding.
func (in *KameletBinding) DeepCopy() *KameletBinding {
	if in == nil {
		return nil
	}
	out := new(KameletBinding)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KameletBinding) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KameletBindingCondition) DeepCopyInto(out *KameletBindingCondition) {
	*out = *in
	in.LastUpdateTime.DeepCopyInto(&out.LastUpdateTime)
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KameletBindingCondition.
func (in *KameletBindingCondition) DeepCopy() *KameletBindingCondition {
	if in == nil {
		return nil
	}
	out := new(KameletBindingCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KameletBindingList) DeepCopyInto(out *KameletBindingList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]KameletBinding, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KameletBindingList.
func (in *KameletBindingList) DeepCopy() *KameletBindingList {
	if in == nil {
		return nil
	}
	out := new(KameletBindingList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KameletBindingList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KameletBindingSpec) DeepCopyInto(out *KameletBindingSpec) {
	*out = *in
	if in.Integration != nil {
		in, out := &in.Integration, &out.Integration
		*out = new(v1.IntegrationSpec)
		(*in).DeepCopyInto(*out)
	}
	in.Source.DeepCopyInto(&out.Source)
	in.Sink.DeepCopyInto(&out.Sink)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KameletBindingSpec.
func (in *KameletBindingSpec) DeepCopy() *KameletBindingSpec {
	if in == nil {
		return nil
	}
	out := new(KameletBindingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KameletBindingStatus) DeepCopyInto(out *KameletBindingStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]KameletBindingCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KameletBindingStatus.
func (in *KameletBindingStatus) DeepCopy() *KameletBindingStatus {
	if in == nil {
		return nil
	}
	out := new(KameletBindingStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KameletCondition) DeepCopyInto(out *KameletCondition) {
	*out = *in
	in.LastUpdateTime.DeepCopyInto(&out.LastUpdateTime)
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KameletCondition.
func (in *KameletCondition) DeepCopy() *KameletCondition {
	if in == nil {
		return nil
	}
	out := new(KameletCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KameletList) DeepCopyInto(out *KameletList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Kamelet, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KameletList.
func (in *KameletList) DeepCopy() *KameletList {
	if in == nil {
		return nil
	}
	out := new(KameletList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KameletList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KameletProperty) DeepCopyInto(out *KameletProperty) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KameletProperty.
func (in *KameletProperty) DeepCopy() *KameletProperty {
	if in == nil {
		return nil
	}
	out := new(KameletProperty)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KameletSpec) DeepCopyInto(out *KameletSpec) {
	*out = *in
	in.Definition.DeepCopyInto(&out.Definition)
	if in.Sources != nil {
		in, out := &in.Sources, &out.Sources
		*out = make([]v1.SourceSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Flow != nil {
		in, out := &in.Flow, &out.Flow
		*out = new(v1.Flow)
		(*in).DeepCopyInto(*out)
	}
	if in.Authorization != nil {
		in, out := &in.Authorization, &out.Authorization
		*out = new(AuthorizationSpec)
		**out = **in
	}
	if in.Types != nil {
		in, out := &in.Types, &out.Types
		*out = make(map[EventSlot]EventTypeSpec, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.Dependencies != nil {
		in, out := &in.Dependencies, &out.Dependencies
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KameletSpec.
func (in *KameletSpec) DeepCopy() *KameletSpec {
	if in == nil {
		return nil
	}
	out := new(KameletSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KameletStatus) DeepCopyInto(out *KameletStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]KameletCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Properties != nil {
		in, out := &in.Properties, &out.Properties
		*out = make([]KameletProperty, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KameletStatus.
func (in *KameletStatus) DeepCopy() *KameletStatus {
	if in == nil {
		return nil
	}
	out := new(KameletStatus)
	in.DeepCopyInto(out)
	return out
}
