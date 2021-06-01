package native

import (
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

var certFilename = "testdata/remote_attestation/in_teecert.pem"
var proofFilename = "testdata/remote_attestation/proof.hex"


func TestSaveRemoteAttestation(t *testing.T) {
	ds := map[string][]byte{}
	mockCtx := newTxContextMock(ds)

	proofFile, err := os.Open(proofFilename)
	if err != nil {
		t.Fatalf("open file '%s' error: %v", proofFilename, err)
	}

	proofHex, err := ioutil.ReadAll(proofFile)
	if err != nil {
		t.Fatalf("read file '%v' error: %v", proofFile, err)
	}

	proof, err := hex.DecodeString(string(proofHex))
	if err != nil {
		t.Fatalf("decode hex string error: %v", err)
	}

	contactRuntime := PrivateComputeRuntime{}
	params := map[string]string{}
	params["proof"] = string(proof)
	result, err := contactRuntime.SaveRemoteAttestation(mockCtx, params)
	assert.Error(t, err, "Save remote attestation error")

	fmt.Printf("result = %v \n", string(result));
	for key, val := range ds {
		fmt.Printf("key = %v, val = %v \n", key, val)
	}
}

func TestInTeecertPemFile(t *testing.T) {
	file, err := os.Open("/Users/caizhihong/证书测试/in_teecert.pem")
	if err != nil {
		t.Fatalf("open file error: %v", err)
	}

	crtPEM, err := ioutil.ReadAll(file)
	if err != nil {
		t.Fatalf("read file error: %v", err)
	}

	signingPubkey, cryptoPubkey, err := getPubkeyPairFromCert(crtPEM)
	if err != nil {
		t.Fatalf("get pubkey pair error: %v", err)
	}
	fmt.Printf("signing pub key ==> %v \n", signingPubkey.ToStandardKey())
	fmt.Printf("crypto pub key ==> %v \n", cryptoPubkey.ToStandardKey())
}