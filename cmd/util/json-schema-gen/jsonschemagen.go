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
	"path"
	"reflect"

	"github.com/alecthomas/jsonschema"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/spf13/cobra"
)

// Publishes predefined images for all Camel components
func main() {
	sources := []interface{}{
		v1alpha1.Integration{},
		v1alpha1.IntegrationContext{},
		v1alpha1.IntegrationPlatform{},
		v1alpha1.CamelCatalog{},
	}

	var out string

	var cmd = cobra.Command{
		Use: "jsonschemagen --out=${path}",
		Run: func(_ *cobra.Command, _ []string) {
			for _, source := range sources {
				//pin
				source := source

				schema := jsonschema.Reflect(source)
				b, err := json.MarshalIndent(schema, "", "  ")
				if err != nil {
					fmt.Println("error:", err)
				}

				v := reflect.ValueOf(source)
				t := reflect.Indirect(v).Type().Name()
				o := path.Join(out, t+".json")

				fmt.Println("Write", t, "json-schema to:", o)

				if err := ioutil.WriteFile(o, b, 0644); err != nil {
					fmt.Println("error:", err)
					os.Exit(-1)
				}
			}
		},
	}

	cmd.Flags().StringVar(&out, "out", ".", "the path where to generate the json schema for cr")

	if err := cmd.Execute(); err != nil {
		fmt.Println("error:", err)
		os.Exit(-1)
	}
}
