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

package knative

import "regexp"

var channelRegexp = regexp.MustCompile("^knative:[/]*channel/([a-z0-9.-]+)(?:[/?].*|$)")

// ExtractChannelNames extracts all Knative named channels from the given URIs
func ExtractChannelNames(uris []string) []string {
	channels := make([]string, 0)
	for _, uri := range uris {
		channel := ExtractChannelName(uri)
		if channel != "" {
			channels = append(channels, channel)
		}
	}
	return channels
}

// ExtractChannelName returns a channel name from the Knative URI if present
func ExtractChannelName(uri string) string {
	match := channelRegexp.FindStringSubmatch(uri)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}
