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
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

func GenerateKeystore(ctx context.Context, keystoreDir, keystoreName, keystorePass string, data []byte) error {
	args := strings.Fields(fmt.Sprintf("-importcert -noprompt -alias maven -storepass %s -keystore %s", keystorePass, keystoreName))
	cmd := exec.CommandContext(ctx, "keytool", args...)
	cmd.Dir = keystoreDir
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	if err != nil {
		return err
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
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout

		err := cmd.Run()
		if err != nil {
			return err
		}
	}

	return nil
}

// The keytool CLI mandates a password at least 6 characters long
// to access any key stores.
func NewKeystorePassword() string {
	return randString(10)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func randString(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			sb.WriteByte(letterBytes[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return sb.String()
}
