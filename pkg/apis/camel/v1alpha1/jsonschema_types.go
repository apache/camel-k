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

// NOTE: this file has been originally copied from https://github.com/kubernetes/apiextensions-apiserver/blob/33bb2d8b009bae408e40818a93877459efeb4cb1/pkg/apis/apiextensions/v1/types_jsonschema.go

package v1alpha1

import (
	"encoding/json"
	"errors"
)

type JSONSchemaProp struct {
	ID          string `json:"id,omitempty" protobuf:"bytes,1,opt,name=id"`
	Description string `json:"description,omitempty" protobuf:"bytes,4,opt,name=description"`
	Type        string `json:"type,omitempty" protobuf:"bytes,5,opt,name=type"`
	// format is an OpenAPI v3 format string. Unknown formats are ignored. The following formats are validated:
	//
	// - bsonobjectid: a bson object ID, i.e. a 24 characters hex string
	// - uri: an URI as parsed by Golang net/url.ParseRequestURI
	// - email: an email address as parsed by Golang net/mail.ParseAddress
	// - hostname: a valid representation for an Internet host name, as defined by RFC 1034, section 3.1 [RFC1034].
	// - ipv4: an IPv4 IP as parsed by Golang net.ParseIP
	// - ipv6: an IPv6 IP as parsed by Golang net.ParseIP
	// - cidr: a CIDR as parsed by Golang net.ParseCIDR
	// - mac: a MAC address as parsed by Golang net.ParseMAC
	// - uuid: an UUID that allows uppercase defined by the regex (?i)^[0-9a-f]{8}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{12}$
	// - uuid3: an UUID3 that allows uppercase defined by the regex (?i)^[0-9a-f]{8}-?[0-9a-f]{4}-?3[0-9a-f]{3}-?[0-9a-f]{4}-?[0-9a-f]{12}$
	// - uuid4: an UUID4 that allows uppercase defined by the regex (?i)^[0-9a-f]{8}-?[0-9a-f]{4}-?4[0-9a-f]{3}-?[89ab][0-9a-f]{3}-?[0-9a-f]{12}$
	// - uuid5: an UUID5 that allows uppercase defined by the regex (?i)^[0-9a-f]{8}-?[0-9a-f]{4}-?5[0-9a-f]{3}-?[89ab][0-9a-f]{3}-?[0-9a-f]{12}$
	// - isbn: an ISBN10 or ISBN13 number string like "0321751043" or "978-0321751041"
	// - isbn10: an ISBN10 number string like "0321751043"
	// - isbn13: an ISBN13 number string like "978-0321751041"
	// - creditcard: a credit card number defined by the regex ^(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|6(?:011|5[0-9][0-9])[0-9]{12}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|(?:2131|1800|35\\d{3})\\d{11})$ with any non digit characters mixed in
	// - ssn: a U.S. social security number following the regex ^\\d{3}[- ]?\\d{2}[- ]?\\d{4}$
	// - hexcolor: an hexadecimal color code like "#FFFFFF: following the regex ^#?([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$
	// - rgbcolor: an RGB color code like rgb like "rgb(255,255,2559"
	// - byte: base64 encoded binary data
	// - password: any kind of string
	// - date: a date string like "2006-01-02" as defined by full-date in RFC3339
	// - duration: a duration string like "22 ns" as parsed by Golang time.ParseDuration or compatible with Scala duration format
	// - datetime: a date time string like "2014-12-15T19:30:20.000Z" as defined by date-time in RFC3339.
	Format string `json:"format,omitempty" protobuf:"bytes,6,opt,name=format"`
	Title  string `json:"title,omitempty" protobuf:"bytes,7,opt,name=title"`
	// default is a default value for undefined object fields.
	Default          *JSON        `json:"default,omitempty" protobuf:"bytes,8,opt,name=default"`
	Maximum          *json.Number `json:"maximum,omitempty" protobuf:"bytes,9,opt,name=maximum"`
	ExclusiveMaximum bool         `json:"exclusiveMaximum,omitempty" protobuf:"bytes,10,opt,name=exclusiveMaximum"`
	Minimum          *json.Number `json:"minimum,omitempty" protobuf:"bytes,11,opt,name=minimum"`
	ExclusiveMinimum bool         `json:"exclusiveMinimum,omitempty" protobuf:"bytes,12,opt,name=exclusiveMinimum"`
	MaxLength        *int64       `json:"maxLength,omitempty" protobuf:"bytes,13,opt,name=maxLength"`
	MinLength        *int64       `json:"minLength,omitempty" protobuf:"bytes,14,opt,name=minLength"`
	Pattern          string       `json:"pattern,omitempty" protobuf:"bytes,15,opt,name=pattern"`
	MaxItems         *int64       `json:"maxItems,omitempty" protobuf:"bytes,16,opt,name=maxItems"`
	MinItems         *int64       `json:"minItems,omitempty" protobuf:"bytes,17,opt,name=minItems"`
	UniqueItems      bool         `json:"uniqueItems,omitempty" protobuf:"bytes,18,opt,name=uniqueItems"`
	MaxProperties    *int64       `json:"maxProperties,omitempty" protobuf:"bytes,21,opt,name=maxProperties"`
	MinProperties    *int64       `json:"minProperties,omitempty" protobuf:"bytes,22,opt,name=minProperties"`
	MultipleOf       *json.Number `json:"multipleOf,omitempty" protobuf:"bytes,19,opt,name=multipleOf"`
	Enum             []JSON       `json:"enum,omitempty" protobuf:"bytes,20,rep,name=enum"`
	Example          *JSON        `json:"example,omitempty" protobuf:"bytes,36,opt,name=example"`
	Nullable         bool         `json:"nullable,omitempty" protobuf:"bytes,37,opt,name=nullable"`
}

// JSONSchemaProps is a JSON-Schema following Specification Draft 4 (http://json-schema.org/).
type JSONSchemaProps struct {
	ID           string                    `json:"id,omitempty" protobuf:"bytes,1,opt,name=id"`
	Description  string                    `json:"description,omitempty" protobuf:"bytes,4,opt,name=description"`
	Title        string                    `json:"title,omitempty" protobuf:"bytes,7,opt,name=title"`
	Properties   map[string]JSONSchemaProp `json:"properties,omitempty" protobuf:"bytes,29,rep,name=properties"`
	Required     []string                  `json:"required,omitempty" protobuf:"bytes,23,rep,name=required"`
	Example      *JSON                     `json:"example,omitempty" protobuf:"bytes,36,opt,name=example"`
	ExternalDocs *ExternalDocumentation    `json:"externalDocs,omitempty" protobuf:"bytes,35,opt,name=externalDocs"`
	Schema       JSONSchemaURL             `json:"$schema,omitempty" protobuf:"bytes,2,opt,name=schema"`
	Type         string                    `json:"type,omitempty" protobuf:"bytes,5,opt,name=type"`
}

// RawMessage is a raw encoded JSON value.
// It implements Marshaler and Unmarshaler and can
// be used to delay JSON decoding or precompute a JSON encoding.
// +kubebuilder:validation:Type=""
// +kubebuilder:validation:Format=""
// +kubebuilder:pruning:PreserveUnknownFields
type RawMessage []byte

// +kubebuilder:validation:Type=""
// JSON represents any valid JSON value.
// These types are supported: bool, int64, float64, string, []interface{}, map[string]interface{} and nil.
type JSON struct {
	RawMessage `json:",inline" protobuf:"bytes,1,opt,name=raw"`
}

// MarshalJSON returns m as the JSON encoding of m.
func (m RawMessage) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return m, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (m *RawMessage) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}

var _ json.Marshaler = (*RawMessage)(nil)
var _ json.Unmarshaler = (*RawMessage)(nil)

// JSONSchemaURL represents a schema url.
type JSONSchemaURL string

// ExternalDocumentation allows referencing an external resource for extended documentation.
type ExternalDocumentation struct {
	Description string `json:"description,omitempty" protobuf:"bytes,1,opt,name=description"`
	URL         string `json:"url,omitempty" protobuf:"bytes,2,opt,name=url"`
}
