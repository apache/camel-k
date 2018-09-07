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
	"errors"
	"log"
	"strconv"

	"github.com/spf13/cobra"
)

// NewCmdContext --
func newContextEditCmd(rootCmdOptions *RootCmdOptions) *cobra.Command {
	impl := &contextEditCommand{
		RootCmdOptions: rootCmdOptions,
		discard:        false,
		save:           true,
		dependencies: contextResource{
			toAdd:    make([]string, 0),
			toRemove: make([]string, 0),
		},
		env: contextResource{
			toAdd:    make([]string, 0),
			toRemove: make([]string, 0),
		},
		properties: contextResource{
			toAdd:    make([]string, 0),
			toRemove: make([]string, 0),
		},
	}

	cmd := cobra.Command{
		Use:   "edit",
		Short: "Edit an Integration Context",
		Long:  `Edit an Integration Context.`,
		Args:  impl.validateArgs,
		RunE:  impl.run,
	}

	cmd.Flags().BoolVarP(&impl.discard, "discard", "x", false, "Discard the draft")
	cmd.Flags().BoolVarP(&impl.save, "save", "s", true, "Save the context")

	cmd.Flags().StringSliceVarP(&impl.env.toAdd, "env", "e", nil, "Add an environment variable")
	cmd.Flags().StringSliceVarP(&impl.env.toRemove, "env-rm", "E", nil, "Remove an environment variable")
	cmd.Flags().StringSliceVarP(&impl.properties.toAdd, "property", "p", nil, "Add a system property")
	cmd.Flags().StringSliceVarP(&impl.properties.toRemove, "property-rm", "P", nil, "Remove a system property")
	cmd.Flags().StringSliceVarP(&impl.dependencies.toAdd, "dependency", "d", nil, "Add a dependency")
	cmd.Flags().StringSliceVarP(&impl.dependencies.toRemove, "dependency-rm", "D", nil, "Remove a dependency")

	return &cmd
}

type contextResource struct {
	toAdd    []string
	toRemove []string
}

type contextEditCommand struct {
	*RootCmdOptions

	env          contextResource
	properties   contextResource
	dependencies contextResource

	// rollback the context to the state before it was edited
	discard bool

	// save the context then the operator should rebuild the image, this is
	// set as true by default, if you want to mark a context as a draft,
	// set it to false
	save bool
}

func (command *contextEditCommand) validateArgs(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("accepts 1 arg, received " + strconv.Itoa(len(args)))
	}

	return nil
}

func (command *contextEditCommand) run(cmd *cobra.Command, args []string) error {
	log.Printf("context=%s, config=%+v", args[0], command)
	return nil
}
