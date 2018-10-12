// Package tools contains common tools used throughout this application.
package tools

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

// Files returns all files found in paths. If recursive is false, it only
// returns the immediate files in the paths.
func Files(recursive bool, paths ...string) ([]string, error) {
	var (
		fileList []string
		fn       = files
	)
	if recursive {
		fn = rfiles
	}

	for _, p := range paths {
		f, err := fn(p)
		if err != nil {
			return nil, err
		}
		fileList = append(fileList, f...)
	}
	if len(fileList) == 0 {
		return nil, errors.New("no files found")
	}
	fileList = unique(fileList)
	fileList = nonBinary(fileList)
	return fileList, nil
}

func unique(fileList []string) []string {
	var (
		ret  []string
		seen = make(map[string]struct{}, len(fileList))
	)
	for _, f := range fileList {
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		ret = append(ret, f)
	}
	return ret
}

func nonBinary(fileList []string) []string {
	var (
		ret []string
	)
	for _, f := range fileList {
		if isPlainText(f) {
			ret = append(ret, f)
		}
	}
	return ret
}

func rfiles(location string) ([]string, error) {
	fileList := []string{}
	err := filepath.Walk(location, func(location string, f os.FileInfo, err error) error {
		if os.IsPermission(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if !f.IsDir() {
			fileList = append(fileList, location)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return fileList, nil
}

func files(location string) ([]string, error) {
	if s, err := os.Stat(location); err == nil && !s.IsDir() {
		return []string{location}, nil
	}
	fileList := []string{}
	files, err := ioutil.ReadDir(location)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if !f.IsDir() {
			p := path.Join(location, f.Name())
			fileList = append(fileList, p)
		}
	}
	return fileList, nil
}

// TODO: we should ignore the line in search stage instead.
func isPlainText(name string) bool {
	f, err := os.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()
	header := make([]byte, 512)
	_, err = f.Read(header)
	if err != nil && err != io.EOF {
		return false
	}

	return IsPlainText(string(header))
}
