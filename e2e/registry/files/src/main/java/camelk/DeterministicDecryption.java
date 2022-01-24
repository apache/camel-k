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
package camelk;

import java.io.IOException;
import java.security.GeneralSecurityException;
import com.google.crypto.tink.CleartextKeysetHandle;
import com.google.crypto.tink.DeterministicAead;
import com.google.crypto.tink.JsonKeysetReader;
import com.google.crypto.tink.KeysetHandle;
import com.google.crypto.tink.aead.AeadConfig;
import com.google.crypto.tink.config.TinkConfig;

public class DeterministicDecryption {

    private static final String KEYSET = "{\"primaryKeyId\":1757621741,\"key\":[{\"keyData\":{\"typeUrl\":\"type.googleapis.com/google.crypto.tink.AesSivKey\",\"value\":\"EkC4wjyYD7TPwkpxWFwkCrMmkOkpS2wdEwAchBW9INoJvmZHxBysCT0y6tfcW0RXeVWqMYqpuHfV/Np387MQcvme\",\"keyMaterialType\":\"SYMMETRIC\"},\"status\":\"ENABLED\",\"keyId\":1757621741,\"outputPrefixType\":\"TINK\"}]}";


    public static String decrypt(byte[] encrypted) throws GeneralSecurityException, IOException {
        AeadConfig.register();
        TinkConfig.register();

        KeysetHandle keysetHandle = CleartextKeysetHandle.read(
                JsonKeysetReader.withString(KEYSET));

        // Get the primitive.
        DeterministicAead daead =
                keysetHandle.getPrimitive(DeterministicAead.class);

        // deterministically decrypt a ciphertext.
        return new String(daead.decryptDeterministically(encrypted, null));

    }
}
