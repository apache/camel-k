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
	ID          string `json:"id,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
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
	Format string `json:"format,omitempty"`
	Title  string `json:"title,omitempty"`
	// default is a default value for undefined object fields.
	Default          *JSON        `json:"default,omitempty"`
	Maximum          *json.Number `json:"maximum,omitempty"`
	ExclusiveMaximum bool         `json:"exclusiveMaximum,omitempty"`
	Minimum          *json.Number `json:"minimum,omitempty"`
	ExclusiveMinimum bool         `json:"exclusiveMinimum,omitempty"`
	MaxLength        *int64       `json:"maxLength,omitempty"`
	MinLength        *int64       `json:"minLength,omitempty"`
	Pattern          string       `json:"pattern,omitempty"`
	MaxItems         *int64       `json:"maxItems,omitempty"`
	MinItems         *int64       `json:"minItems,omitempty"`
	UniqueItems      bool         `json:"uniqueItems,omitempty"`
	MaxProperties    *int64       `json:"maxProperties,omitempty"`
	MinProperties    *int64       `json:"minProperties,omitempty"`
	MultipleOf       *json.Number `json:"multipleOf,omitempty"`
	Enum             []JSON       `json:"enum,omitempty"`
	Example          *JSON        `json:"example,omitempty"`
	Nullable         bool         `json:"nullable,omitempty"`
	// The list of descriptors that determine which UI components to use on different views
	XDescriptors []string `json:"x-descriptors,omitempty"`
}

// JSONSchemaProps is a JSON-Schema following Specification Draft 4 (http://json-schema.org/).
type JSONSchemaProps struct {
	ID           string                    `json:"id,omitempty"`
	Description  string                    `json:"description,omitempty"`
	Title        string                    `json:"title,omitempty"`
	Properties   map[string]JSONSchemaProp `json:"properties,omitempty"`
	Required     []string                  `json:"required,omitempty"`
	Example      *JSON                     `json:"example,omitempty"`
	ExternalDocs *ExternalDocumentation    `json:"externalDocs,omitempty"`
	Schema       JSONSchemaURL             `json:"$schema,omitempty"`
	Type         string                    `json:"type,omitempty"`
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
	RawMessage `json:",inline"`
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

// String returns a string representation of RawMessage
func (m *RawMessage) String() string {
	if m == nil {
		return ""
	}
	b, err := m.MarshalJSON()
	if err != nil {
		return ""
	}
	return string(b)
}

var _ json.Marshaler = (*RawMessage)(nil)
var _ json.Unmarshaler = (*RawMessage)(nil)

// JSONSchemaURL represents a schema url.
type JSONSchemaURL string

// ExternalDocumentation allows referencing an external resource for extended documentation.
type ExternalDocumentation struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}
