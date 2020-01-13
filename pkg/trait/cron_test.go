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
	"strings"
	"testing"

	"github.com/magiconair/properties/assert"
)

func TestCronFromURI(t *testing.T) {
	tests := []struct {
		uri        string
		uri2       string
		uri3       string
		cron       string
		components string
	}{
		// Timer only
		{
			uri: "timer:tick?period=60000&delay=12", // invalid
		},
		{
			uri: "timer:tick?period=60000&repeatCount=10", // invalid
		},
		{
			uri:        "timer:tick?period=60000",
			cron:       "0/1 * * * ?",
			components: "timer",
		},
		{
			uri:        "timer:tick?period=28800000",
			cron:       "0 0/8 * * ?",
			components: "timer",
		},
		{
			uri:        "timer:tick?period=120000",
			cron:       "0/2 * * * ?",
			components: "timer",
		},
		{
			uri: "timer:tick?period=120001", // invalid
		},
		{
			uri:        "timer:tick?period=1m",
			cron:       "0/1 * * * ?",
			components: "timer",
		},
		{
			uri:        "timer:tick?period=5m",
			cron:       "0/5 * * * ?",
			components: "timer",
		},
		{
			uri:        "timer:tick?period=10m",
			cron:       "0/10 * * * ?",
			components: "timer",
		},
		{
			uri: "timer:tick?period=61m", // invalid
		},
		{
			uri:        "timer:tick?period=2h",
			cron:       "0 0/2 * * ?",
			components: "timer",
		},
		{
			uri:        "timer:tick?period=2h60m",
			cron:       "0 0/3 * * ?",
			components: "timer",
		},
		{
			uri:        "timer:tick?period=24h",
			cron:       "0 0 * * ?",
			components: "timer",
		},
		{
			uri: "timer:tick?period=3h60s", // invalid
		},
		{
			uri:        "timer:tick?period=3h59m60s",
			cron:       "0 0/4 * * ?",
			components: "timer",
		},

		// Quartz only
		{
			uri:        "quartz:trigger?cron=0 0 0/4 * * ?",
			cron:       "0 0/4 * * ?",
			components: "quartz",
		},
		{
			uri:        "quartz:trigger?cron=0+0+0/4+*+*+?",
			cron:       "0 0/4 * * ?",
			components: "quartz",
		},
		{
			uri: "quartz:trigger?cron=*+0+0/4+*+*+?", // invalid
		},
		{
			uri: "quartz:trigger?cron=0+0+0/4+*+*+?+2020", // invalid
		},
		{
			uri: "quartz:trigger?cron=1+0+0/4+*+*+?", // invalid
		},
		{
			uri: "quartz:trigger?cron=0+0+0/4+*+*+?&fireNow=true", // invalid
		},

		// Cron only
		{
			uri:        "cron:tab?schedule=1/2 * * * ?",
			cron:       "1/2 * * * ?",
			components: "cron",
		},
		{
			uri:        "cron:tab?schedule=0 0 0/4 * * ?",
			cron:       "0 0/4 * * ?",
			components: "cron",
		},
		{
			uri:        "cron:tab?schedule=0+0+0/4+*+*+?",
			cron:       "0 0/4 * * ?",
			components: "cron",
		},
		{
			uri: "cron:tab?schedule=*+0+0/4+*+*+?", // invalid
		},
		{
			uri:        "cron:tab?schedule=0+0,6+0/4+*+*+MON-THU",
			cron:       "0,6 0/4 * * MON-THU",
			components: "cron",
		},
		{
			uri: "cron:tab?schedule=0+0+0/4+*+*+?+2020", // invalid
		},
		{
			uri: "cron:tab?schedule=1+0+0/4+*+*+?", // invalid
		},

		// Mixed scenarios
		{
			uri:        "cron:tab?schedule=0/2 * * * ?",
			uri2:       "timer:tick?period=2m",
			cron:       "0/2 * * * ?",
			components: "cron,timer",
		},
		{
			uri:        "cron:tab?schedule=0 0/2 * * ?",
			uri2:       "timer:tick?period=2h",
			uri3:       "quartz:trigger?cron=0 0 0/2 * * ? ?",
			cron:       "0 0/2 * * ?",
			components: "cron,timer,quartz",
		},
		{
			uri:  "cron:tab?schedule=1 0/2 * * ?",
			uri2: "timer:tick?period=2h",
			uri3: "quartz:trigger?cron=0 0 0/2 * * ? ?",
			// invalid
		},
		{
			uri:  "cron:tab?schedule=0 0/2 * * ?",
			uri2: "timer:tick?period=3h",
			uri3: "quartz:trigger?cron=0 0 0/2 * * ? ?",
			// invalid
		},
	}

	for _, test := range tests {
		t.Run(test.uri, func(t *testing.T) {
			uris := []string{test.uri, test.uri2, test.uri3}
			filtered := make([]string, 0, len(uris))
			for _, uri := range uris {
				if uri != "" {
					filtered = append(filtered, uri)
				}
			}

			res := getCronForURIs(filtered)
			gotCron := ""
			if res != nil {
				gotCron = res.schedule
			}
			assert.Equal(t, gotCron, test.cron)

			gotComponents := ""
			if res != nil {
				gotComponents = strings.Join(res.components, ",")
			}
			assert.Equal(t, gotComponents, test.components)
		})
	}
}
