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

package tar

import (
	"io"
	"io/ioutil"
	"os"
	"path"

	tarutils "archive/tar"
)

// Extract --
func Extract(source string, destinationBase string) error {
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := tarutils.NewReader(file)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		targetName := path.Join(destinationBase, header.Name)
		targetDir, _ := path.Split(targetName)
		if err := os.MkdirAll(targetDir, 0777); err != nil {
			return err
		}
		buffer, err := ioutil.ReadAll(reader)
		if err != nil {
			return err
		}
		if err := ioutil.WriteFile(targetName, buffer, os.FileMode(header.Mode)); err != nil {
			return err
		}
	}
	return nil
}
