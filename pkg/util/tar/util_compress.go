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
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func CreateTarFile(fileNames []string, archiveName string, cmd *cobra.Command) {
	out, err := os.Create(archiveName)
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "Error writing archive:", err.Error())
	}
	defer out.Close()

	err = createArchiveFile(fileNames, out)
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "Error writing archive:", err.Error())
	}
}

func createArchiveFile(files []string, buf io.Writer) error {
	gw := gzip.NewWriter(buf)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Iterate over files and add them to the tar archive
	for _, file := range files {
		err := addEntryToArchive(tw, file)
		if err != nil {
			return err
		}
	}
	return nil
}

func addEntryToArchive(tw *tar.Writer, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return err
	}
	header.Name = filename
	err = tw.WriteHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(tw, file)
	if err != nil {
		return err
	}
	defer os.Remove(filename)
	return nil
}
