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

	"sigs.k8s.io/yaml"
)

func main() {
	if len(os.Args) != 6 {
		fmt.Println(`Use "json-schema-gen <crd> <schema> <path> <isArray> <destination>`)
		os.Exit(1)
	}
	crd := os.Args[1]
	schema := os.Args[2]
	path := os.Args[3]
	isArray := "true" == os.Args[4]
	destination := os.Args[5]

	if err := generate(crd, schema, path, isArray, destination); err != nil {
		panic(err)
	}
}

func generate(crd, schema, path string, isArray bool, destination string) error {
	crdData, err := ioutil.ReadFile(crd)
	if err != nil {
		return err
	}

	var crdObj map[string]interface{}
	if err := yaml.Unmarshal(crdData, &crdObj); err != nil {
		return err
	}

	bigSchema := getSchemaFromCRD(crdObj)
	ref := bigSchema
	for _, p := range pathComponents(path) {
		ref = ref["properties"].(map[string]interface{})[p].(map[string]interface{})
	}

	dslSchema, err := loadDSLSchema(schema)
	dslObjectSchema := dslSchema["items"].(map[string]interface{})
	if err != nil {
		return err
	}
	if !isArray && dslSchema["type"] == "array" {
		dslSchema = dslObjectSchema
	}

	// merge schemas
	for k, v := range dslSchema {
		if k != "definitions" {
			ref[k] = v
		}
	}
	// readd definitions
	if _, alreadyHasDefs := bigSchema["definitions"]; alreadyHasDefs {
		panic("unexpected definitions found in CRD")
	}
	bigSchema["definitions"] = dslObjectSchema["definitions"]
	rebaseRefs(dslSchema)

	result, err := json.MarshalIndent(bigSchema, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(destination, result, 0666)
}

func remapRef(ref string) string {
	return "#" + strings.TrimPrefix(ref, "#/items")
}

func rebaseRefs(schema map[string]interface{}) {
	for k, v := range schema {
		if k == "$ref" && reflect.TypeOf(v).Kind() == reflect.String {
			schema[k] = remapRef(fmt.Sprintf("%v", v))
		} else if reflect.TypeOf(v).Kind() == reflect.Map {
			rebaseRefs(v.(map[string]interface{}))
		} else if reflect.TypeOf(v).Kind() == reflect.Slice {
			for _, vv := range v.([]interface{}) {
				if reflect.TypeOf(vv).Kind() == reflect.Map {
					rebaseRefs(vv.(map[string]interface{}))
				}
			}
		}
	}
}

func loadDSLSchema(schema string) (map[string]interface{}, error) {
	content, err := ioutil.ReadFile(schema)
	if err != nil {
		return nil, err
	}
	var dslSchema map[string]interface{}
	if err := json.Unmarshal(content, &dslSchema); err != nil {
		return nil, err
	}
	return dslSchema, nil
}

func getSchemaFromCRD(crd map[string]interface{}) map[string]interface{} {
	res := crd["spec"].(map[string]interface{})
	res = res["validation"].(map[string]interface{})
	res = res["openAPIV3Schema"].(map[string]interface{})
	return res
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
