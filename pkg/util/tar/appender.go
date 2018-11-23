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
	atar "archive/tar"
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
)

// Appender provides a high level abstraction over writing tar files
type Appender struct {
	tarFile *os.File
	writer  *atar.Writer
}

// NewAppender creates a new tar appender
func NewAppender(fileName string) (*Appender, error) {
	tarFile, err := os.Create(fileName)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create tar file "+fileName)
	}

	writer := atar.NewWriter(tarFile)
	appender := Appender{
		tarFile: tarFile,
		writer:  writer,
	}
	return &appender, nil
}

// Close closes all handles managed by the appender
func (t *Appender) Close() error {
	if err := t.writer.Close(); err != nil {
		return err
	}
	err := t.tarFile.Close()
	if err != nil {
		return err
	}
	return nil
}

// AddFile adds a file content to the tarDir, using the original file name.
// It returns the full path of the file inside the tar.
func (t *Appender) AddFile(filePath string, tarDir string) (string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}
	_, fileName := path.Split(filePath)
	if tarDir != "" {
		fileName = path.Join(tarDir, fileName)
	}

	t.writer.WriteHeader(&atar.Header{
		Name:    fileName,
		Size:    info.Size(),
		Mode:    int64(info.Mode()),
		ModTime: info.ModTime(),
	})

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(t.writer, file)
	if err != nil {
		return "", errors.Wrap(err, "cannot add file to the tar archive")
	}

	return fileName, nil
}

// AddFileWithName adds a file content to the tarDir, using the fiven file name.
// It returns the full path of the file inside the tar.
func (t *Appender) AddFileWithName(fileName string, filePath string, tarDir string) (string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}
	if tarDir != "" {
		fileName = path.Join(tarDir, fileName)
	}

	t.writer.WriteHeader(&atar.Header{
		Name:    fileName,
		Size:    info.Size(),
		Mode:    int64(info.Mode()),
		ModTime: info.ModTime(),
	})

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(t.writer, file)
	if err != nil {
		return "", errors.Wrap(err, "cannot add file to the tar archive")
	}

	return fileName, nil
}

// AppendData appends the given content to a file inside the tar, creating it if it does not exist
func (t *Appender) AppendData(data []byte, tarPath string) error {
	t.writer.WriteHeader(&atar.Header{
		Name: tarPath,
		Size: int64(len(data)),
		Mode: 0644,
	})

	_, err := t.writer.Write(data)
	if err != nil {
		return errors.Wrap(err, "cannot add data to the tar archive")
	}
	return nil
}
