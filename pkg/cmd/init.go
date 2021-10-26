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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/resources"
	"github.com/spf13/cobra"
)

func newCmdInit(rootCmdOptions *RootCmdOptions) (*cobra.Command, *initCmdOptions) {
	options := initCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:     "init [flags] IntegrationFile.java",
		Short:   "Initialize empty Camel K files",
		Long:    `Initialize empty Camel K integrations and other resources.`,
		PreRunE: decode(&options),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.validate(cmd, args); err != nil {
				return err
			}
			if err := options.init(cmd, args); err != nil {
				return err
			}
			return nil
		},
		Annotations: map[string]string{
			offlineCommandLabel: "true",
		},
	}

	return &cmd, &options
}

type initCmdOptions struct {
	*RootCmdOptions
}

func (o *initCmdOptions) validate(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("init expects exactly 1 argument, received %d", len(args))
	}

	fileName := args[0]
	if o.extractLanguage(fileName) == nil {
		return fmt.Errorf("unsupported file type: %s", fileName)
	}

	return nil
}

func (o *initCmdOptions) init(_ *cobra.Command, args []string) error {
	fileName := args[0]
	language := o.extractLanguage(fileName)
	return o.writeFromTemplate(*language, fileName)
}

func (o *initCmdOptions) writeFromTemplate(language v1.Language, fileName string) error {
	simpleName := filepath.Base(fileName)
	if idx := strings.Index(simpleName, "."); idx >= 0 {
		simpleName = simpleName[:idx]
	}

	type TemplateParameters struct {
		Name string
	}
	params := TemplateParameters{
		Name: simpleName,
	}
	rawData := resources.ResourceAsString(fmt.Sprintf("/templates/%s.tmpl", language))
	if rawData == "" {
		return fmt.Errorf("cannot find template for language %s", string(language))
	}
	tmpl, err := template.New(string(language)).Parse(rawData)
	if err != nil {
		return err
	}
	out, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0o777)
	if err != nil {
		return err
	}
	defer out.Close()

	return tmpl.Execute(out, params)
}

func (o *initCmdOptions) extractLanguage(fileName string) *v1.Language {
	if strings.HasSuffix(fileName, ".kamelet.yaml") {
		language := v1.LanguageKamelet
		return &language
	}
	for _, l := range v1.Languages {
		if strings.HasSuffix(fileName, fmt.Sprintf(".%s", string(l))) {
			return &l
		}
	}
	return nil
}
