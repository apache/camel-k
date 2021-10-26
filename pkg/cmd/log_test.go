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
	"testing"

	"github.com/apache/camel-k/pkg/util/test"
)

func TestLogsAlias(t *testing.T) {
	options, rootCommand := kamelTestPreAddCommandInit()
	logCommand, _ := newCmdLog(options)
	rootCommand.AddCommand(logCommand)

	kamelTestPostAddCommandInit(t, rootCommand)

	_, err := test.ExecuteCommand(rootCommand, "logs")

	// in case of error we expect this to be the log default message
	if err != nil && err.Error() != "log expects an integration name argument" {
		t.Fatalf("Expected error result for invalid alias `logs`")
	}
}
