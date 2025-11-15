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

package trait

import (
	"fmt"
	"strings"
)

const defaultAgentDir = "/agents"
const defaultAgentVolume = "agents"
const defaultAgentInitContainerName = "agents-download"

type jvmAgent struct {
	name    string
	url     string
	options string
}

func (a *jvmAgent) arg() string {
	arg := fmt.Sprintf("-javaagent:%s/%s.jar", defaultAgentDir, a.name)
	if a.options != "" {
		arg += "=" + a.options
	}

	return arg
}

func (t *jvmTrait) parseJvmAgents() ([]jvmAgent, error) {
	agents := make([]jvmAgent, 0, len(t.Agents))
	for _, task := range t.Agents {
		split := strings.SplitN(task, ";", 3)
		if len(split) < 2 {
			return nil, fmt.Errorf(`could not parse JVM agent "%s": format expected "agent;agent-url[;jvm-agent-options]"`, task)
		}
		agent := jvmAgent{
			name: split[0],
			url:  split[1],
		}
		if len(split) == 3 {
			agent.options = split[2]
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

func (t *jvmTrait) parseJvmAgentsArgs() ([]string, error) {
	jvmAgents, err := t.parseJvmAgents()
	if err != nil {
		return nil, err
	}
	jvmAgentsArgs := make([]string, 0, len(jvmAgents))
	for _, jvmAgent := range jvmAgents {
		jvmAgentsArgs = append(jvmAgentsArgs, jvmAgent.arg())
	}

	return jvmAgentsArgs, nil
}

func (t *jvmTrait) hasJavaAgents() bool {
	return t.Agents != nil
}
