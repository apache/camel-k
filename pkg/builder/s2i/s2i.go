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

package s2i

import (
	"github.com/apache/camel-k/pkg/builder"
)

func init() {
	builder.RegisterSteps(Steps)
}

type steps struct {
	Publisher builder.Step
}

// Steps --
var Steps = steps{
	Publisher: builder.NewStep(
		builder.ApplicationPublishPhase,
		publisher,
	),
}

// DefaultSteps --
var DefaultSteps = []builder.Step{
	builder.Steps.GenerateProject,
	builder.Steps.GenerateProjectSettings,
	builder.Steps.InjectDependencies,
	builder.Steps.SanitizeDependencies,
	builder.Steps.ComputeDependencies,
	builder.Steps.IncrementalPackager,
	Steps.Publisher,
}
