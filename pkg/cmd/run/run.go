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

package run

import (
	"github.com/spf13/cobra"
	"errors"
	"strconv"
	"os"
	"github.com/apache/camel-k/pkg/cmd/config"
	"fmt"
)

type runCmdFlags struct {
	language string
}

func NewCmdRun() *cobra.Command {
	flags := runCmdFlags{}
	cmd := cobra.Command{
		Use:   "run [file to run]",
		Short: "Run a integration on Kubernetes",
		Long:  `Deploys and execute a integration pod on Kubernetes.`,
		Args: validateArgs,
		RunE: run,
	}

	cmd.Flags().StringVarP(&flags.language, "language", "l", "", "Programming Language used to write the file")

	return &cmd
}

func validateArgs(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("accepts 1 arg, received " + strconv.Itoa(len(args)))
	}
	fileName := args[0]
	if _, err := os.Stat(fileName); err != nil && os.IsNotExist(err) {
		return errors.New("file " + fileName + " does not exist")
	} else if err != nil {
		return errors.New("error while accessing file " + fileName)
	}
	return nil
}

func run(cmd *cobra.Command, args []string) error {
	_, err := config.NewKubeClient(cmd)
	if err != nil {
		return err
	}

	fmt.Println("Now something should run")
	return nil
}
