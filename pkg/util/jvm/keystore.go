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

package jvm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/apache/camel-k/pkg/util"
)

func GenerateJavaKeystore(ctx context.Context, keystoreDir, keystoreName string, data []byte) error {
	tmpFile := "ca-cert.tmp"
	if err := util.WriteFileWithContent(keystoreDir, tmpFile, data); err != nil {
		return err
	}
	defer os.Remove(path.Join(keystoreDir, tmpFile))

	args := strings.Fields(fmt.Sprintf("-importcert -noprompt -alias maven -file %s -keystore %s", tmpFile, keystoreName))
	cmd := exec.CommandContext(ctx, "keytool", args...)
	cmd.Dir = keystoreDir
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	if err != nil {
		return err
	}

	// Try to locale root CA certificates truststore, in order to import them
	// into the newly created truststore. It avoids tempering the system-wide
	// JVM truststore.
	javaHome, ok := os.LookupEnv("JAVA_HOME")
	if ok {
		caCertsPath := path.Join(javaHome, "lib/security/cacerts")
		args := strings.Fields(fmt.Sprintf("-importkeystore -noprompt -srckeystore %s -srcstorepass %s -destkeystore %s", caCertsPath, "changeit", keystoreName))
		cmd := exec.CommandContext(ctx, "keytool", args...)
		cmd.Dir = keystoreDir
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout

		err := cmd.Run()
		if err != nil {
			return err
		}
	}

	return nil
}
