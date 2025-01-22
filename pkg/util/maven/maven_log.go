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

package maven

import (
	"regexp"

	"github.com/apache/camel-k/v2/pkg/util/log"
)

type mavenLog struct {
	Level string `json:"level"`
	Msg   string `json:"msg"`
}

const (
	TRACE   = "TRACE"
	DEBUG   = "DEBUG"
	INFO    = "INFO"
	WARNING = "WARNING"
	ERROR   = "ERROR"
	FATAL   = "FATAL"
)

var mavenLogger = log.WithName("maven.build")
var mavenLoggingFormat = regexp.MustCompile(`^\[(TRACE|DEBUG|INFO|WARNING|ERROR|FATAL)\] (.*)$`)

// LogHandler is in charge to log the text passed and, if the trace is an error, to return the message to the caller.
func LogHandler(s string) string {
	l := parseLog(s)
	normalizeLog(l)

	if l.Level == ERROR {
		return l.Msg
	}

	return ""
}

func parseLog(line string) mavenLog {
	var l mavenLog

	matches := mavenLoggingFormat.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 || len(matches[0]) != 3 {
		// If this is happening, then, we have a problem with parsing the maven output
		// however we are printing the output in its plain format
		l = mavenLog{
			Level: INFO,
			Msg:   line,
		}
	} else {
		l = mavenLog{
			Level: matches[0][1],
			Msg:   matches[0][2],
		}
	}

	return l
}

func normalizeLog(mavenLog mavenLog) {
	switch mavenLog.Level {
	case DEBUG, TRACE:
		mavenLogger.Debug(mavenLog.Msg)
	case INFO, WARNING:
		mavenLogger.Info(mavenLog.Msg)
	case ERROR, FATAL:
		mavenLogger.Error(nil, mavenLog.Msg)
	}
}
