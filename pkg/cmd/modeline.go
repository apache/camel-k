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
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/apache/camel-k/pkg/cmd/source"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/modeline"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	runCmdName        = "run"
	buildCmdName      = "build"
	localCmdName      = "local"
	runCmdSourcesArgs = "source"
	inspectCmdName    = "inspect"
)

var (
	nonRunOptions = map[string]bool{
		"language": true, // language is a marker modeline option for other tools
	}
	disallowedOptions = map[string]bool{
		"dev":  true,
		"wait": true,
		"logs": true,
		"sync": true,
	}

	// file options must be considered relative to the source files they belong to.
	fileOptions = map[string]bool{
		"kube-config":   true,
		"open-api":      true,
		"property-file": true,
	}

	// file format options are those options that admit multiple values, not only files (ie, key=value|configmap|secret|file syntax).
	fileFormatOptions = map[string]bool{
		"resource":       true,
		"config":         true,
		"property":       true,
		"build-property": true,
	}
)

// NewKamelWithModelineCommand ---.
func NewKamelWithModelineCommand(ctx context.Context, osArgs []string) (*cobra.Command, []string, error) {
	originalFlags := osArgs[1:]
	rootCmd, flags, err := createKamelWithModelineCommand(ctx, originalFlags)
	if err != nil {
		fmt.Fprintln(rootCmd.ErrOrStderr(), "Error:", err.Error())
		return rootCmd, flags, err
	}
	if len(originalFlags) != len(flags) {
		// Give a feedback about the actual command that is run
		fmt.Fprintln(rootCmd.ErrOrStderr(), "Modeline options have been loaded from source files")
		fmt.Fprint(rootCmd.ErrOrStderr(), "Full command: kamel ")
		for _, a := range flags {
			fmt.Fprintf(rootCmd.ErrOrStderr(), "%s ", a)
		}
		fmt.Fprintln(rootCmd.ErrOrStderr())
	}
	return rootCmd, flags, nil
}

func createKamelWithModelineCommand(ctx context.Context, args []string) (*cobra.Command, []string, error) {
	rootCmd, err := NewKamelCommand(ctx)
	if err != nil {
		return rootCmd, nil, err
	}

	target, flags, err := rootCmd.Find(args)
	if err != nil {
		return rootCmd, nil, err
	}

	isLocalBuild := target.Name() == buildCmdName && target.Parent().Name() == localCmdName
	isInspect := target.Name() == inspectCmdName

	if target.Name() != runCmdName && !isLocalBuild && !isInspect {
		return rootCmd, args, nil
	}

	err = target.ParseFlags(flags)
	if errors.Is(err, pflag.ErrHelp) {
		return rootCmd, args, nil
	} else if err != nil {
		return rootCmd, nil, err
	}

	fg := target.Flags()

	// Only the run command has source flag (for now). Remove condition when
	// local run also supports source.
	additionalSources := make([]string, 0)
	if target.Name() == runCmdName && target.Parent().Name() != localCmdName {
		additionalSources, err = fg.GetStringArray(runCmdSourcesArgs)
		if err != nil {
			return rootCmd, nil, err
		}
	}

	files := make([]string, 0, len(fg.Args())+len(additionalSources))
	files = append(files, fg.Args()...)
	files = append(files, additionalSources...)

	opts, err := extractModelineOptions(ctx, files, rootCmd)
	if err != nil {
		return rootCmd, nil, errors.Wrap(err, "cannot read sources")
	}

	// Extract list of property/trait names already specified by the user.
	cliParamNames := []string{}
	index := 0
	for _, arg := range args {
		if arg == "-p" || arg == "--property" || arg == "-t" || arg == "--trait" || arg == "--build-property" {
			// Property or trait is assumed to be in the form: <name>=<value>
			splitValues := strings.Split(args[index+1], "=")
			cliParamNames = append(cliParamNames, splitValues[0])
		}
		index++
	}

	// filter out in place non-run options
	nOpts := 0
	for _, o := range opts {
		// Check if property name is given by user.
		paramAlreadySpecifiedByUser := false
		if o.Name == "property" || o.Name == "trait" || o.Name == "build-property" {
			paramComponents := strings.Split(o.Value, "=")
			for _, paramName := range cliParamNames {
				if paramName == paramComponents[0] {
					paramAlreadySpecifiedByUser = true
					break
				}
			}
		}

		// Skip properties already specified by the user otherwise add all options.
		if !paramAlreadySpecifiedByUser && !nonRunOptions[o.Name] {
			opts[nOpts] = o
			nOpts++
		}
	}

	opts = opts[:nOpts]

	for _, o := range opts {
		prefix := "-"
		if len(o.Name) > 1 {
			prefix = "--"
		}
		// Using the k=v syntax to avoid issues with booleans
		if len(o.Value) > 0 {
			args = append(args, fmt.Sprintf("%s%s=%s", prefix, o.Name, o.Value))
		} else {
			args = append(args, fmt.Sprintf("%s%s", prefix, o.Name))
		}
	}

	// Recreating the command as it's dirty
	rootCmd, err = NewKamelCommand(ctx)
	if err != nil {
		return rootCmd, nil, err
	}
	rootCmd.SetArgs(args)

	return rootCmd, args, nil
}

func extractModelineOptions(ctx context.Context, sources []string, cmd *cobra.Command) ([]modeline.Option, error) {
	opts := make([]modeline.Option, 0)

	resolvedSources, err := source.Resolve(ctx, sources, false, cmd)
	if err != nil {
		return opts, errors.Wrap(err, "failed to resolve sources")
	}

	for _, resolvedSource := range resolvedSources {
		ops, err := extractModelineOptionsFromSource(resolvedSource)
		if err != nil {
			return opts, err
		}

		ops, err = expandModelineEnvVarOptions(ops)
		if err != nil {
			return opts, err
		}

		opts = append(opts, ops...)
	}

	return opts, nil
}

func extractModelineOptionsFromSource(resolvedSource source.Source) ([]modeline.Option, error) {
	ops, err := modeline.Parse(resolvedSource.Name, resolvedSource.Content)
	if err != nil {
		return ops, errors.Wrapf(err, "cannot process file %s", resolvedSource.Location)
	}
	for i, o := range ops {
		if disallowedOptions[o.Name] {
			return ops, fmt.Errorf("option %q is disallowed in modeline", o.Name)
		}

		if fileOptions[o.Name] && resolvedSource.Local {
			baseDir := filepath.Dir(resolvedSource.Origin)
			refPath := o.Value
			if !filepath.IsAbs(refPath) {
				full := path.Join(baseDir, refPath)
				o.Value = full
				ops[i] = o
			}
		} else if fileFormatOptions[o.Name] && resolvedSource.Local {
			baseDir := filepath.Dir(resolvedSource.Origin)
			refPath := getRefPathOrProperty(o.Value)
			if !filepath.IsAbs(refPath) {
				full := getFullPathOrProperty(o.Value, path.Join(baseDir, refPath))
				o.Value = full
				ops[i] = o
			}
		}
	}

	return ops, nil
}

func getRefPathOrProperty(pathOrProperty string) string {
	if strings.HasPrefix(pathOrProperty, "file:") {
		return strings.Replace(pathOrProperty, "file:", "", 1)
	}
	return pathOrProperty
}

func getFullPathOrProperty(pathOrProperty string, fullPath string) string {
	if strings.HasPrefix(pathOrProperty, "file:") {
		return fmt.Sprintf("file:%s", fullPath)
	}
	return pathOrProperty
}

func expandModelineEnvVarOptions(ops []modeline.Option) ([]modeline.Option, error) {
	// List of additional command line options with expanded values for variables
	// marked as immediate environment variables.
	//
	// Immediate values are marked as: ${ENV_VAR}
	//
	// Values marked as {{env:ENV_VAR}} should be added too in their un-evaluated state.
	// Their evaluation will occur at integration runtime.
	//
	listWithExpandedOptions := []modeline.Option{}
	for _, opt := range ops {
		// Eliminate white spaces:
		compactOptValue := strings.ReplaceAll(opt.Value, " ", "")

		// Check if value can be immediately evaluated.
		if strings.Contains(compactOptValue, "${") {
			// Split value into 2 substrings, first contains the start of the string,
			// second contains the remainder of the string.
			splitOptBeforeEV := strings.SplitN(compactOptValue, "${", 2)

			// Remainder of the string is split into the environment variable we want
			// to replace with its value, and the tail of the string.
			splitOptAfterEV := strings.SplitN(splitOptBeforeEV[1], "}", 2)

			// Evaluate variable.
			envVarValue, err := util.GetEnvironmentVariable(splitOptAfterEV[0])
			if err != nil {
				return nil, err
			}

			// Save evaluated version.
			opt.Value = splitOptBeforeEV[0] + envVarValue + splitOptAfterEV[1]
		} else if strings.Contains(compactOptValue, "{{env:") {
			// Split value into 2 substrings, first contains the start of the string,
			// second contains the remainder of the string.
			splitOptBeforeEV := strings.SplitN(compactOptValue, "{{env:", 2)

			// Remainder of the string is split into the environment variable we want
			// to replace with its value, and the tail of the string.
			splitOptEVName := strings.SplitN(splitOptBeforeEV[1], "}}", 2)

			// Add the variable to a list of lazy environment variables.
			util.ListOfLazyEvaluatedEnvVars = append(util.ListOfLazyEvaluatedEnvVars, splitOptEVName[0])
		}

		// Add option to list whether it contained environment variables or not.
		listWithExpandedOptions = append(listWithExpandedOptions, opt)
	}

	return listWithExpandedOptions, nil
}
