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
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/log"
)

var (
	logger = log.WithName("keytool")

	loggerInfo  = func(s string) { logger.Info(s) }
	loggerError = func(s string) { logger.Error(nil, s) }
)

func GenerateKeystore(ctx context.Context, keystoreDir, keystoreName, keystorePass string, data [][]byte) error {
	for i, data := range data {
		args := strings.Fields(fmt.Sprintf("-importcert -noprompt -alias maven-%d -storepass %s -keystore %s", i, keystorePass, keystoreName))
		cmd := exec.CommandContext(ctx, "keytool", args...)
		cmd.Dir = keystoreDir
		cmd.Stdin = bytes.NewReader(data)
		// keytool logs info messages to stderr, as stdout is used to output results,
		// otherwise it logs error messages to stdout.
		err := util.RunAndLog(ctx, cmd, loggerError, loggerInfo)
		if err != nil {
			return err
		}
	}

	// Try to locate root CA certificates truststore, in order to import them
	// into the newly created truststore. It avoids tempering the system-wide
	// JVM truststore.
	javaHome, ok := os.LookupEnv("JAVA_HOME")
	if ok {
		caCertsPath := path.Join(javaHome, "lib/security/cacerts")
		args := strings.Fields(fmt.Sprintf("-importkeystore -noprompt -srckeystore %s -srcstorepass %s -destkeystore %s -deststorepass %s", caCertsPath, "changeit", keystoreName, keystorePass))
		cmd := exec.CommandContext(ctx, "keytool", args...)
		cmd.Dir = keystoreDir
		// keytool logs info messages to stderr, as stdout is used to output results,
		// otherwise it logs error messages to stdout.
		err := util.RunAndLog(ctx, cmd, loggerError, loggerInfo)
		if err != nil {
			return err
		}
	}

	return nil
}

// NewKeystorePassword generates a random password.
// The keytool CLI mandates a password at least 6 characters long
// to access any key stores.
func NewKeystorePassword() string {
	return util.RandomString(10)
}
