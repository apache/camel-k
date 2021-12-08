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
