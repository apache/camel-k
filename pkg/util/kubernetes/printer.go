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

package kubernetes

import (
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/runtime"
)

// CLIPrinter is delegated to print the runtime object
type CLIPrinter struct {
	// It accepts either yaml or json format
	Format string
}

// PrintObj prints the obj in json|yaml format according to the type of the obj.
func (p *CLIPrinter) PrintObj(obj runtime.Object, output io.Writer) error {
	var data []byte
	var err error
	switch p.Format {
	case "yaml":
		data, err = ToYAML(obj)
	case "json":
		data, err = ToJSON(obj)
	default:
		err = fmt.Errorf("invalid output format option '%s', should be one of: yaml|json", p.Format)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(output, string(data))
	return nil
}
