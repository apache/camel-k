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
	"io/fs"
	"os"
	"runtime"
	"strings"
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
	return isExistingFile(uri)
}

// isGlobCandidate checks if the provided uri doesn't have a supported scheme prefix,
// and is not an existing file, because then it could be a glob pattern like "sources/*.yaml".
func isGlobCandidate(uri string) (bool, error) {
	if hasSupportedScheme(uri) {
		// it's not a local file as it matches one of the supporting schemes
		return false, nil
	}

	exists, err := isExistingFile(uri)
	if err != nil {
		return false, err
	}

	return !exists, nil
}

func isExistingFile(uri string) (bool, error) {
	info, err := os.Stat(uri)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		if _, ok := err.(*fs.PathError); ok && runtime.GOOS == "windows" { // nolint
			// Windows returns a PathError rather than NotExist is path is invalid
			return false, nil
		}

		// If it is a different error (ie, permission denied) we should report it back
		return false, fmt.Errorf("file system error while looking for %s: %w", uri, err)
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
