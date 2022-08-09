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

package source

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
)

const (
	// Supported source schemes.
	gistScheme   = "gist"
	githubScheme = "github"
	httpScheme   = "http"
	httpsScheme  = "https"
)

func IsLocalAndFileExists(uri string) (bool, error) {
	if hasSupportedScheme(uri) {
		// it's not a local file as it matches one of the supporting schemes
		return false, nil
	}
	info, err := os.Stat(uri)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		// If it is a different error (ie, permission denied) we should report it back
		return false, errors.Wrap(err, fmt.Sprintf("file system error while looking for %s", uri))
	}

	return !info.IsDir(), nil
}

func hasSupportedScheme(uri string) bool {
	if strings.HasPrefix(strings.ToLower(uri), gistScheme+":") ||
		strings.HasPrefix(strings.ToLower(uri), githubScheme+":") ||
		strings.HasPrefix(strings.ToLower(uri), httpScheme+":") ||
		strings.HasPrefix(strings.ToLower(uri), httpsScheme+":") {
		return true
	}

	return false
}
