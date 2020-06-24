// +build !ignore_autogenerated

// Code generated by operator-sdk. DO NOT EDIT.

package v1alpha1

import (
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthorizationSpec) DeepCopyInto(out *AuthorizationSpec) {
	*out = *in
	return
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
func (in *EventTypeSpec) DeepCopyInto(out *EventTypeSpec) {
	*out = *in
	if in.Schema != nil {
		in, out := &in.Schema, &out.Schema
		*out = new(JSONSchemaProps)
		(*in).DeepCopyInto(*out)
	}
	return
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
	return
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
	if in.Raw != nil {
		in, out := &in.Raw, &out.Raw
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
	return
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
		return
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
			(*out)[key] = *val.DeepCopy()
		}
		return
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
		*out = new(float64)
		**out = **in
	}
	if in.Minimum != nil {
		in, out := &in.Minimum, &out.Minimum
		*out = new(float64)
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
		*out = new(float64)
		**out = **in
	}
	if in.Enum != nil {
		in, out := &in.Enum, &out.Enum
		*out = make([]JSON, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
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
		*out = new(JSONSchemaPropsOrArray)
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
		*out = new(JSONSchemaPropsOrBool)
		(*in).DeepCopyInto(*out)
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
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.AdditionalItems != nil {
		in, out := &in.AdditionalItems, &out.AdditionalItems
		*out = new(JSONSchemaPropsOrBool)
		(*in).DeepCopyInto(*out)
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
	return
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
func (in *JSONSchemaPropsOrArray) DeepCopyInto(out *JSONSchemaPropsOrArray) {
	*out = *in
	if in.Schema != nil {
		in, out := &in.Schema, &out.Schema
		*out = new(JSONSchemaProps)
		(*in).DeepCopyInto(*out)
	}
	if in.JSONSchemas != nil {
		in, out := &in.JSONSchemas, &out.JSONSchemas
		*out = make([]JSONSchemaProps, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JSONSchemaPropsOrArray.
func (in *JSONSchemaPropsOrArray) DeepCopy() *JSONSchemaPropsOrArray {
	if in == nil {
		return nil
	}
	out := new(JSONSchemaPropsOrArray)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JSONSchemaPropsOrBool) DeepCopyInto(out *JSONSchemaPropsOrBool) {
	*out = *in
	if in.Schema != nil {
		in, out := &in.Schema, &out.Schema
		*out = new(JSONSchemaProps)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JSONSchemaPropsOrBool.
func (in *JSONSchemaPropsOrBool) DeepCopy() *JSONSchemaPropsOrBool {
	if in == nil {
		return nil
	}
	out := new(JSONSchemaPropsOrBool)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JSONSchemaPropsOrStringArray) DeepCopyInto(out *JSONSchemaPropsOrStringArray) {
	*out = *in
	if in.Schema != nil {
		in, out := &in.Schema, &out.Schema
		*out = new(JSONSchemaProps)
		(*in).DeepCopyInto(*out)
	}
	if in.Property != nil {
		in, out := &in.Property, &out.Property
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JSONSchemaPropsOrStringArray.
func (in *JSONSchemaPropsOrStringArray) DeepCopy() *JSONSchemaPropsOrStringArray {
	if in == nil {
		return nil
	}
	out := new(JSONSchemaPropsOrStringArray)
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
	return
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
func (in *KameletCondition) DeepCopyInto(out *KameletCondition) {
	*out = *in
	in.LastUpdateTime.DeepCopyInto(&out.LastUpdateTime)
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
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
	return
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
	in.Flow.DeepCopyInto(&out.Flow)
	out.Authorization = in.Authorization
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
	return
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
	return
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
