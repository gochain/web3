package web3

import (
	"encoding/json"
	"math/big"
	"reflect"
	"testing"
)

func TestConvertArgument(t *testing.T) {
	const addr = "0xa25b5e2d2d63dad7fa940e239925f29320f5103d"
	const hash = "0x0123456789012345678901234567890101234567890123456789012345678901"
	bytes4 := hash[:10]
	var bytes4Array [4]byte
	copy(bytes4Array[:], FromHex(bytes4))

	tests := []struct {
		name    string
		t       byte
		s       int
		param   interface{}
		want    interface{}
		wantErr bool
	}{
		{"int256<-int", IntTy, 256, 1, big.NewInt(1), false},
		{"int256<-big.Int", IntTy, 256, big.NewInt(1), big.NewInt(1), false},
		{"uint256<-int", UintTy, 256, 1, big.NewInt(1), false},
		{"uint256<-big.Int", UintTy, 256, big.NewInt(1), big.NewInt(1), false},
		{"int8<-int", IntTy, 8, 1, int8(1), false},
		{"int8<-min", IntTy, 8, -128, int8(-128), false},
		{"int8<-big.Int", IntTy, 8, big.NewInt(1), int8(1), false},
		{"uint8<-int", UintTy, 8, 1, uint8(1), false},
		{"uint8<-big.Int", UintTy, 8, big.NewInt(1), uint8(1), false},
		{"int256<-hex", IntTy, 256, "0x1", big.NewInt(1), false},
		{"int256<-string", IntTy, 256, "1", big.NewInt(1), false},
		{"uint256<-zero", UintTy, 256, "0", big.NewInt(0), false},
		{"uint64<-json", UintTy, 64, json.Number("10000000000000001"), uint64(10000000000000001), false},
		{"address<-address", AddressTy, 20, HexToAddress(addr), HexToAddress(addr), false},
		{"address<-hex", AddressTy, 20, addr, HexToAddress(addr), false},
		{"hash<-hash", FixedBytesTy, 32, HexToHash(hash), HexToHash(hash), false},
		{"hash<-hex", FixedBytesTy, 32, hash, HexToHash(hash), false},
		{"[4]byte<-[4]bytes", FixedBytesTy, 4, bytes4Array, bytes4Array, false},
		{"[4]byte<-hex", FixedBytesTy, 4, bytes4, bytes4Array, false},
		{"bytes<-bytes", BytesTy, 0, FromHex("0x1234"), FromHex("0x1234"), false},
		{"bytes<-hex", BytesTy, 0, "0x1234", FromHex("0x1234"), false},

		// Error cases (including signed underflow!)
		{"uint256<-negative", UintTy, 256, -1, nil, true},
		{"uint8<-negative", UintTy, 8, -1, nil, true},
		{"uint8<-overflow", UintTy, 8, 256, nil, true},
		{"int8<-underflow", IntTy, 8, -129, nil, true},
		{"int8<-overflow", IntTy, 8, 128, nil, true},
		{"int16<-underflow", IntTy, 16, -40000, nil, true},
		{"int32<-underflow", IntTy, 32, int64(-3000000000), nil, true},
		{"int256<-float64", IntTy, 256, float64(1), nil, true},
		{"uint8<-float", UintTy, 8, 1.1, nil, true},
		{"uint8<-negative-float", UintTy, 8, -1.1, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertArgument(ABIType{T: tt.t, Size: tt.s}, tt.param)
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got (%T) %v; want (%T) %v", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestParseGweiAndUnits(t *testing.T) {
	for _, tt := range []struct {
		val    string
		exp    *big.Int
		expErr bool
	}{
		{val: "1", exp: weiPerGwei},
		{val: "10", exp: Gwei(10)},
		{val: "1.1", exp: new(big.Int).Add(Gwei(1), big.NewInt(100000000))},
		{val: "100000", exp: Gwei(100000)},
		{val: "1.000000001", exp: new(big.Int).Add(Gwei(1), big.NewInt(1))},
		{val: "1.0000000001", expErr: true},
		{val: "-1", expErr: true},
	} {
		t.Run(tt.val, func(t *testing.T) {
			got, err := ParseGwei(tt.val)
			if (err != nil) != tt.expErr {
				t.Fatalf("ParseGwei(%q) err=%v, wantErr=%v", tt.val, err, tt.expErr)
			}
			if !tt.expErr && got.Cmp(tt.exp) != 0 {
				t.Fatalf("got %s, want %s", got, tt.exp)
			}
		})
	}

	ethWei, err := ParseAmount("1.5eth")
	if err != nil || ethWei.String() != "1500000000000000000" {
		t.Fatalf("ParseAmount(1.5eth) = %v, %v", ethWei, err)
	}
}

func TestViemReferenceVectors(t *testing.T) {
	const (
		pkHex             = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
		wantChecksumAddr  = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
		viemPersonalSig   = "0x6c1e2f48b1608b104bca23917bc69d498e4a8f62a6c16578515f48f564004ab15ad050dcb7741bd3a6bd353fd7b677aa7552a3a6244f83e42bffc52da185987d1c"
		viemLegacyRawTx   = "0xf86b0784773594008252089470997970c51812dc3a010c7d01b50e0d17dc79c88814d1120d7b16000080819ba085c717fa08b32916426027f645d411fb0f78af6fc66889a16fd340ccf2e5d3f99f3d5c3360c8b1352420aa335b3cf6df61f6b5976da1a7c17e8146092bb0c567"
		viemLegacyHash    = "0x266e7d14ad6855173150d7f025ff7846ac8a3c575ba6c8f02c48a7c9a7fc8288"
		viemEIP1559RawTx  = "0x02f8b9012a8459682f008506fc23ac00830186a09470997970c51812dc3a010c7d01b50e0d17dc79c888016345785d8a0000b844a9059cbb00000000000000000000000070997970c51812dc3a010c7d01b50e0d17dc79c80000000000000000000000000000000000000000000000000de0b6b3a7640000c080a0545b37e14f040a594a747c4eace78a20ebb8fe267de1455cdb23f0b3246e18b0a0653ea2a49a48d0b91c24f518e0335b7fda1333f2e6ce249daf45811821fcd7a9"
		viemEIP1559Hash   = "0xee88d9c68544b894de13979924026b65b90989f680e9f0db72f4b50153070a2e"
		viemBatchCalldata = "0x509c5cbc000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000120000000000000000000000000000000000000000000000000000000000000000200000000000000000000000070997970c51812dc3a010c7d01b50e0d17dc79c80000000000000000000000003c44cdddb6a900fa2b585dd299e03d12fa4293bc0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000006400000000000000000000000000000000000000000000000000000000000000c80000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000006553f1000000000000000000000000000000000000000000000000000000000000000007706179726f6c6c00000000000000000000000000000000000000000000000000"
		viemCustomErr     = "0xcf47918100000000000000000000000000000000000000000000000000000000000000320000000000000000000000000000000000000000000000000000000000000064"
	)

	acc, err := ParsePrivateKey(pkHex)
	if err != nil {
		t.Fatalf("ParsePrivateKey failed: %v", err)
	}
	if acc.PublicKey() != wantChecksumAddr {
		t.Fatalf("EIP-55 address mismatch: got %s, want %s", acc.PublicKey(), wantChecksumAddr)
	}

	// EIP-191 personal_sign & recovery against viem
	msg := []byte("Hello Web3!")
	recoveredFromViem, err := PersonalRecover(msg, FromHex(viemPersonalSig))
	if err != nil {
		t.Fatalf("PersonalRecover on viem sig failed: %v", err)
	}
	if recoveredFromViem.Hex() != wantChecksumAddr {
		t.Fatalf("PersonalRecover got %s, want %s", recoveredFromViem.Hex(), wantChecksumAddr)
	}
	ourSig, err := PersonalSign(msg, acc.Key())
	if err != nil {
		t.Fatalf("PersonalSign failed: %v", err)
	}
	recoveredOurs, err := PersonalRecover(msg, ourSig)
	if err != nil || recoveredOurs.Hex() != wantChecksumAddr {
		t.Fatalf("PersonalRecover on our sig got %s, err=%v", recoveredOurs.Hex(), err)
	}

	// Legacy EIP-155 transaction (chainId 60)
	toAddr := HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
	legacyVal, _ := new(big.Int).SetString("1500000000000000000", 10)
	legacyTx := &Transaction{
		Type:     TxTypeLegacy,
		Nonce:    7,
		To:       &toAddr,
		Value:    legacyVal,
		GasLimit: 21000,
		GasPrice: big.NewInt(2000000000),
	}
	rawLegacy, err := SignTransaction(legacyTx, big.NewInt(60), acc.Key())
	if err != nil {
		t.Fatalf("SignTransaction legacy failed: %v", err)
	}
	if EncodeHex(rawLegacy) != viemLegacyRawTx {
		t.Fatalf("legacy raw tx mismatch:\ngot:  %s\nwant: %s", EncodeHex(rawLegacy), viemLegacyRawTx)
	}
	if legacyTx.Hash.Hex() != viemLegacyHash {
		t.Fatalf("legacy tx hash mismatch: got %s, want %s", legacyTx.Hash.Hex(), viemLegacyHash)
	}
	sender, err := RecoverTransactionSender(legacyTx)
	if err != nil || sender != acc.Address() {
		t.Fatalf("RecoverTransactionSender legacy = %s, err=%v", sender.Hex(), err)
	}

	// EIP-1559 dynamic fee transaction (chainId 1)
	eip1559Val, _ := new(big.Int).SetString("100000000000000000", 10)
	eip1559Data := FromHex("0xa9059cbb00000000000000000000000070997970c51812dc3a010c7d01b50e0d17dc79c80000000000000000000000000000000000000000000000000de0b6b3a7640000")
	dynTx := &Transaction{
		Type:                 TxTypeDynamicFee,
		Nonce:                42,
		To:                   &toAddr,
		Value:                eip1559Val,
		GasLimit:             100000,
		MaxPriorityFeePerGas: big.NewInt(1500000000),
		MaxFeePerGas:         big.NewInt(30000000000),
		Input:                eip1559Data,
	}
	rawDyn, err := SignTransaction(dynTx, big.NewInt(1), acc.Key())
	if err != nil {
		t.Fatalf("SignTransaction eip1559 failed: %v", err)
	}
	if EncodeHex(rawDyn) != viemEIP1559RawTx {
		t.Fatalf("eip1559 raw tx mismatch:\ngot:  %s\nwant: %s", EncodeHex(rawDyn), viemEIP1559RawTx)
	}
	if dynTx.Hash.Hex() != viemEIP1559Hash {
		t.Fatalf("eip1559 tx hash mismatch: got %s, want %s", dynTx.Hash.Hex(), viemEIP1559Hash)
	}
	dynSender, err := RecoverTransactionSender(dynTx)
	if err != nil || dynSender != acc.Address() {
		t.Fatalf("RecoverTransactionSender eip1559 = %s, err=%v", dynSender.Hex(), err)
	}

	// ABI packing with slices + dynamic tuple vs viem
	fn, err := ParseHumanSignature("batchSend(address[] recipients, uint256[] amounts, (string memo, uint64 timestamp) meta)")
	if err != nil {
		t.Fatalf("ParseHumanSignature failed: %v", err)
	}
	packedBody, err := fn.Inputs.Pack(
		"[0x70997970C51812dc3A010C7d01b50e0d17dc79C8,0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC]",
		"[100,200]",
		`{"memo":"payroll","timestamp":1700000000}`,
	)
	if err != nil {
		t.Fatalf("fn.Inputs.Pack failed: %v", err)
	}
	fullCalldata := EncodeHex(append(append([]byte{}, fn.ID...), packedBody...))
	if fullCalldata != viemBatchCalldata {
		t.Fatalf("ABI batchCalldata mismatch:\ngot:  %s\nwant: %s", fullCalldata, viemBatchCalldata)
	}

	// Unpack round-trip on batchCalldata
	unpacked, err := fn.Inputs.UnpackValues(packedBody)
	if err != nil {
		t.Fatalf("UnpackValues failed: %v", err)
	}
	metaMap, ok := unpacked[2].(map[string]interface{})
	if !ok || metaMap["memo"] != "payroll" || metaMap["timestamp"] != uint64(1700000000) {
		t.Fatalf("unexpected unpacked tuple: %#v", unpacked[2])
	}

	// Custom Solidity error decoding vs viem
	errABI, err := ParseABI(`[{"type":"error","name":"InsufficientBalance","inputs":[{"name":"available","type":"uint256"},{"name":"required","type":"uint256"}]}]`)
	if err != nil {
		t.Fatalf("ParseABI failed: %v", err)
	}
	revMsg, ok := DecodeRevert(FromHex(viemCustomErr), &errABI)
	if !ok || revMsg != "execution reverted: InsufficientBalance(available=50, required=100)" {
		t.Fatalf("unexpected DecodeRevert: ok=%v msg=%q", ok, revMsg)
	}
}

func TestKeystoreRoundTrip(t *testing.T) {
	acc, err := CreateAccount()
	if err != nil {
		t.Fatal(err)
	}
	ksJSON, err := EncryptKeystore(acc, "correct-horse-battery-staple", 4096, 1)
	if err != nil {
		t.Fatalf("EncryptKeystore failed: %v", err)
	}
	decAcc, err := DecryptKeystore(ksJSON, "correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("DecryptKeystore failed: %v", err)
	}
	if decAcc.PrivateKey() != acc.PrivateKey() || decAcc.Address() != acc.Address() {
		t.Fatalf("decrypted account mismatch")
	}
	if _, err := DecryptKeystore(ksJSON, "wrong-password"); err == nil {
		t.Fatalf("expected error on wrong keystore password")
	}

	// Malformed keystore JSON checks (short dklen / short IV must return error, never panic)
	if _, err := DecryptKeystore([]byte(`{"version":3,"crypto":{"cipher":"aes-128-ctr","ciphertext":"00","cipherparams":{"iv":"0102"},"kdf":"scrypt","kdfparams":{"dklen":32,"n":4096,"r":8,"p":1,"salt":"00"},"mac":"00"}}`), "pw"); err == nil {
		t.Fatalf("expected error on short keystore IV")
	}
	if _, err := DecryptKeystore([]byte(`{"version":3,"crypto":{"cipher":"aes-128-ctr","ciphertext":"00","cipherparams":{"iv":"000102030405060708090a0b0c0d0e0f"},"kdf":"scrypt","kdfparams":{"dklen":16,"n":4096,"r":8,"p":1,"salt":"00"},"mac":"00"}}`), "pw"); err == nil {
		t.Fatalf("expected error on dklen < 32")
	}
}

func TestParseHumanSignatureMutability(t *testing.T) {
	nonPay, err := ParseHumanSignature("function submit() nonpayable returns (uint256 payableAmount)")
	if err != nil {
		t.Fatal(err)
	}
	if nonPay.Payable || nonPay.IsConstant() {
		t.Fatalf("expected nonpayable non-constant, got Payable=%v Constant=%v", nonPay.Payable, nonPay.IsConstant())
	}

	shorthandView, err := ParseHumanSignature("hello():(string,string)")
	if err != nil {
		t.Fatal(err)
	}
	if !shorthandView.IsConstant() {
		t.Fatalf("expected colon shorthand with outputs to default to constant/view")
	}

	// Verify GetABI handles inline colon signatures on Windows (no ERROR_INVALID_NAME failure)
	inlineABI, err := GetABI("hello():(string,string)")
	if err != nil || inlineABI == nil {
		t.Fatalf("GetABI inline colon signature failed: %v", err)
	}

	// Verify CSV tuple input ("123,456") does not get partially consumed by JSON decoder
	tupleFn, err := ParseHumanSignature("setPair((uint256 a, uint256 b) pair)")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tupleFn.Inputs.Pack("123,456"); err != nil {
		t.Fatalf("CSV tuple Pack failed: %v", err)
	}

	// Verify huge dynamic bytes length does not integer-overflow bounds check and panic
	strArg := ABIArguments{{Type: ABIType{T: StringTy}}}
	malicious := append(padLeft32(big.NewInt(0x20).Bytes()), FromHex("0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")...)
	if _, err := strArg.UnpackValues(malicious); err == nil {
		t.Fatalf("expected error on maliciously large dynamic bytes length")
	}
}

func TestGenerateGoBinding(t *testing.T) {
	customABI := `[
		{"type":"function","name":"getData","stateMutability":"view","inputs":[{"name":"v","type":"uint8"},{"name":"res","type":"bytes"}],"outputs":[{"name":"","type":"bytes"}]}
	]`
	code, err := GenerateGoBinding("token", "MyToken", customABI, "0x6080604052")
	if err != nil {
		t.Fatalf("GenerateGoBinding failed: %v", err)
	}
	codeStr := string(code)
	if len(code) == 0 {
		t.Fatal("empty generated code")
	}
	if !reflect.DeepEqual(true, len(codeStr) > 0) {
		t.Fatal("unexpected code")
	}
}
