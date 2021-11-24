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

package generators

import (
	"path/filepath"
	"strings"

	"k8s.io/gengo/args"
	"k8s.io/gengo/generator"
	"k8s.io/gengo/namer"
)

// CustomArgs --.
type CustomArgs struct {
	DocDir      string
	ResourceDir string
	TraitPath   string
	NavPath     string
	ListPath    string
}

// NameSystems returns the name system used by the generators in this package.
func NameSystems() namer.NameSystems {
	return namer.NameSystems{
		"default": namer.NewPublicNamer(0),
	}
}

// DefaultNameSystem returns the default name system for ordering the types to be
// processed by the generators in this package.
func DefaultNameSystem() string {
	return "default"
}

// Packages --.
func Packages(context *generator.Context, arguments *args.GeneratorArgs) (packages generator.Packages) {
	for _, i := range context.Inputs {
		pkg := context.Universe[i]
		if pkg == nil {
			continue
		}

		packages = append(packages, &generator.DefaultPackage{
			PackageName: strings.Split(filepath.Base(pkg.Path), ".")[0],
			PackagePath: pkg.Path,
			GeneratorFunc: func(c *generator.Context) (generators []generator.Generator) {
				generators = append(generators, NewTraitDocGen(arguments), NewtraitMetaDataGen(arguments))
				return generators
			},
		})
	}
	return packages
}
