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
	"github.com/apache/camel-k/cmd/util/doc-gen/generators"
	"github.com/spf13/pflag"
	"k8s.io/gengo/args"

	_ "github.com/apache/camel-k/addons"
)

func main() {
	arguments := args.Default()

	// Custom args.
	customArgs := &generators.CustomArgs{}
	pflag.CommandLine.StringVar(&customArgs.DocDir, "doc-dir",
		"./docs", "Root of the document directory.")
	pflag.CommandLine.StringVar(&customArgs.ResourceDir, "resource-dir",
		"./resources", "Root of the resource directory.")
	pflag.CommandLine.StringVar(&customArgs.TraitPath, "traits-path",
		"modules/traits/pages", "Path to the traits directory.")
	pflag.CommandLine.StringVar(&customArgs.NavPath, "nav-path",
		"modules/ROOT/nav.adoc", "Path to the navigation file.")
	pflag.CommandLine.StringVar(&customArgs.ListPath, "list-path",
		"modules/traits/pages/traits.adoc", "Path to the trait list file.")
	arguments.CustomArgs = customArgs

	if err := arguments.Execute(
		generators.NameSystems(),
		generators.DefaultNameSystem(),
		generators.Packages,
	); err != nil {
		panic(err)
	}
}
