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

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	clientscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func main() {
	if len(os.Args) != 6 {
		fmt.Println(`Use "json-schema-gen <crd> <schema> <path> <isArray> <destination>`)
		os.Exit(1)
	}
	crd := os.Args[1]
	schema := os.Args[2]
	path := os.Args[3]
	isArray := os.Args[4] == "true"
	destination := os.Args[5]

	if err := generate(crd, schema, path, isArray, destination); err != nil {
		panic(err)
	}
}

func generate(crdFilename, dslFilename, path string, isArray bool, destination string) error {
	dslSchema, err := loadDslSchema(dslFilename)
	if err != nil {
		return err
	}
	if !isArray && dslSchema["type"] == "array" {
		// nolint: forcetypeassert
		dslSchema = dslSchema["items"].(map[string]interface{})
	}

	rebaseRefs(dslSchema)

	bytes, err := json.Marshal(dslSchema)
	if err != nil {
		return err
	}
	schema := apiextensionsv1.JSONSchemaProps{}
	err = json.Unmarshal(bytes, &schema)
	if err != nil {
		return err
	}

	crdSchema, err := loadCrdSchema(crdFilename)
	if err != nil {
		return err
	}
	// relocate definitions
	if len(crdSchema.Definitions) > 0 {
		panic("unexpected definitions found in CRD")
	}
	if isArray {
		crdSchema.Definitions = schema.Items.Schema.Definitions
		schema.Items.Schema.Definitions = apiextensionsv1.JSONSchemaDefinitions{}
	} else {
		crdSchema.Definitions = schema.Definitions
		schema.Definitions = apiextensionsv1.JSONSchemaDefinitions{}
	}

	// merge DSL schema into the CRD schema
	ref := *crdSchema
	paths := pathComponents(path)
	for _, p := range paths[:len(paths)-1] {
		ref = ref.Properties[p]
	}
	ref.Properties[paths[len(paths)-1]] = schema

	result, err := json.MarshalIndent(crdSchema, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(destination, result, 0o666)
}

func remapRef(ref string) string {
	return "#" + strings.TrimPrefix(ref, "#/items")
}

func rebaseRefs(schema map[string]interface{}) {
	for k, v := range schema {
		switch {
		case k == "$ref" && reflect.TypeOf(v).Kind() == reflect.String:
			schema[k] = remapRef(fmt.Sprintf("%v", v))
		case reflect.TypeOf(v).Kind() == reflect.Map:
			rebaseRefs(v.(map[string]interface{}))
		case reflect.TypeOf(v).Kind() == reflect.Slice:
			for _, vv := range v.([]interface{}) {
				if reflect.TypeOf(vv).Kind() == reflect.Map {
					rebaseRefs(vv.(map[string]interface{}))
				}
			}
		}
	}
}

func loadDslSchema(filename string) (map[string]interface{}, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var dslSchema map[string]interface{}
	if err := json.Unmarshal(bytes, &dslSchema); err != nil {
		return nil, err
	}
	return dslSchema, nil
}

func loadCrdSchema(filename string) (*apiextensionsv1.JSONSchemaProps, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	scheme := clientscheme.Scheme
	err = apiextensionsv1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	obj, err := kubernetes.LoadResourceFromYaml(scheme, string(bytes))
	if err != nil {
		return nil, err
	}
	crd, ok := obj.(*apiextensionsv1.CustomResourceDefinition)
	if !ok {
		return nil, fmt.Errorf("type assertion failed: %v", obj)
	}

	return crd.Spec.Versions[0].Schema.OpenAPIV3Schema, nil
}

func pathComponents(path string) []string {
	res := make([]string, 0)
	for _, p := range strings.Split(path, ".") {
		if len(strings.TrimSpace(p)) == 0 {
			continue
		}
		res = append(res, p)
	}
	return res
}
