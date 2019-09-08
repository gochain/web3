package web3

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gochain/gochain/v3/accounts/abi"
	"github.com/gochain/gochain/v3/common"
	"github.com/gochain/gochain/v3/common/hexutil"
	"github.com/gochain/gochain/v3/core/types"
	"github.com/gochain/gochain/v3/crypto"
	"github.com/gochain/gochain/v3/rlp"
)

var NotFoundErr = errors.New("not found")

const (
	testnetURL = "https://testnet-rpc.gochain.io"
	mainnetURL = "https://rpc.gochain.io"
)

var Networks = map[string]Network{
	"testnet": {
		URL:  testnetURL,
		Unit: "GO",
	},
	"gochain": {
		URL:  mainnetURL,
		Unit: "GO",
	},
	"localhost": {
		URL:  "http://localhost:8545",
		Unit: "GO",
	},
	"ethereum": {
		URL:  "https://main-rpc.linkpool.io",
		Unit: "ETH",
	},
	"ropsten": {
		URL:  "https://ropsten-rpc.linkpool.io",
		Unit: "ETH",
	},
}

type Network struct {
	URL  string
	Unit string
	//TODO net_id, chain_id
}

var (
	weiPerGO   = big.NewInt(1e18)
	weiPerGwei = big.NewInt(1e9)
)

// WeiAsBase converts w wei in to the base unit, and formats it as a decimal fraction with full precision (up to 18 decimals).
func WeiAsBase(w *big.Int) string {
	return new(big.Rat).SetFrac(w, weiPerGO).FloatString(18)
}

// WeiAsGwei converts w wei in to gwei, and formats it as a decimal fraction with full precision (up to 9 decimals).
func WeiAsGwei(w *big.Int) string {
	return new(big.Rat).SetFrac(w, weiPerGwei).FloatString(9)
}

// IntAsFloat converts a *big.Int (ie: wei), to *big.Float (ie: ETH)
func IntAsFloat(i *big.Int, decimals int) *big.Float {
	f := new(big.Float)
	f.SetPrec(100)
	f.SetInt(i)
	f.Quo(f, big.NewFloat(math.Pow10(decimals)))
	return f
}

// FloatAsInt converts a float to a *big.Int based on the decimals passed in
func FloatAsInt(amountF *big.Float, decimals int) *big.Int {
	bigval := new(big.Float)
	bigval.SetPrec(100)
	bigval.SetString(amountF.String()) // have to do this to not lose precision

	coinDecimals := new(big.Float)
	coinDecimals.SetFloat64(math.Pow10(decimals))
	bigval.Mul(bigval, coinDecimals)

	amountI := new(big.Int)
	// todo: could sanity check the accuracy here
	bigval.Int(amountI) // big.NewInt(int64(amountInWeiF)) // amountInGo.Mul(amountInGo, big.NewInt(int64(math.Pow10(18))))
	return amountI
}

// CallConstantFunction executes a contract function call without submitting a transaction.
func CallConstantFunction(ctx context.Context, client Client, myabi abi.ABI, address, functionName string, parameters ...interface{}) (interface{}, error) {
	if parameters == nil {
		parameters = []interface{}{}
	}
	if len(myabi.Methods[functionName].Inputs) != len(parameters) {
		return nil, errors.New("Wrong number of arguments expected:" + strconv.Itoa(len(myabi.Methods[functionName].Inputs)) + " given:" + strconv.Itoa(len(parameters)))
	}
	if address == "" {
		return nil, errors.New("no contract address specified")
	}
	var out []interface{}
	for _, t := range myabi.Methods[functionName].Outputs {
		x, err := convertOutputParameter(t)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	args2, err := ConvertArguments(myabi.Methods[functionName], parameters)
	if err != nil {
		return nil, err
	}
	input, err := myabi.Pack(functionName, args2...)
	if err != nil {
		return nil, err
	}

	toAddress := common.HexToAddress(address)

	res, err := client.Call(ctx, CallMsg{Data: input, To: &toAddress})
	if err != nil {
		return nil, err

	}
	if len(myabi.Methods[functionName].Outputs) > 1 {
		err = myabi.Unpack(&out, functionName, res)
		if err != nil {
			return nil, err
		}
		for i, o := range out {
			out[i] = reflect.ValueOf(o).Elem().Interface()
		}
		return out, nil
	}
	v := out[0]
	err = myabi.Unpack(v, functionName, res)
	if err != nil {
		return nil, err
	}
	ret := reflect.ValueOf(v).Elem().Interface()
	return ret, nil
}

// CallTransactFunction submits a transaction to execute a smart contract function call.
func CallTransactFunction(ctx context.Context, client Client, myabi abi.ABI, address, privateKeyHex, functionName string, amount int, parameters ...interface{}) (*Transaction, error) {
	if address == "" {
		return nil, errors.New("no contract address specified")
	}

	if len(myabi.Methods[functionName].Inputs) != len(parameters) {
		return nil, errors.New("Wrong number of arguments expected:" + strconv.Itoa(len(myabi.Methods[functionName].Inputs)) + " given:" + strconv.Itoa(len(parameters)))
	}
	args2, err := ConvertArguments(myabi.Methods[functionName], parameters)
	if err != nil {
		return nil, err
	}
	input, err := myabi.Pack(functionName, args2...)
	if err != nil {
		return nil, err
	}
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}
	gasPrice, err := client.GetGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get gas price: %v", err)
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("error casting public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.GetPendingTransactionCount(ctx, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("cannot get nonce: %v", err)
	}
	toAddress := common.HexToAddress(address)
	tx := types.NewTransaction(nonce, toAddress, big.NewInt(int64(amount)), 20000000, gasPrice, input)
	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, privateKey)
	if err != nil {
		return nil, fmt.Errorf("cannot sign transaction: %v", err)
	}
	raw, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		return nil, err
	}
	err = client.SendRawTransaction(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("cannot send transaction: %v", err)
	}
	return convertTx(signedTx, fromAddress), nil
}

// DeployContract submits a contract creation transaction.
// abiJSON is only required when including params for the constructor.
func DeployContract(ctx context.Context, client Client, privateKeyHex string, binHex, abiJSON string, params ...interface{}) (*Transaction, error) {
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}

	gasPrice, err := client.GetGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get gas price: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.GetPendingTransactionCount(ctx, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("cannot get nonce: %v", err)
	}
	binData, err := hexutil.Decode(binHex)
	if err != nil {
		return nil, fmt.Errorf("cannot decode contract data: %v", err)
	}
	if len(params) > 0 {
		abiData, err := abi.JSON(strings.NewReader(abiJSON))
		if err != nil {
			return nil, fmt.Errorf("failed to parse ABI: %v", err)
		}
		args2, err := ConvertArguments(abiData.Constructor, params)
		if err != nil {
			return nil, err
		}
		input, err := abiData.Pack("", args2...)
		if err != nil {
			return nil, fmt.Errorf("cannot pack parameters: %v", err)
		}
		binData = append(binData, input...)
	}
	//TODO try to use web3.Transaction only; can't sign currently
	tx := types.NewContractCreation(nonce, big.NewInt(0), 2000000, gasPrice, binData)
	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, privateKey)
	if err != nil {
		return nil, fmt.Errorf("cannot sign transaction: %v", err)
	}
	raw, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		return nil, err
	}
	err = client.SendRawTransaction(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("cannot send transaction: %v", err)
	}

	return convertTx(signedTx, fromAddress), nil
}

func Send(ctx context.Context, client Client, privateKeyHex string, address common.Address, amount *big.Int) (*Transaction, error) {
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}
	gasPrice, err := client.GetGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get gas price: %v", err)
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("error casting public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.GetPendingTransactionCount(ctx, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("cannot get nonce: %v", err)
	}
	tx := types.NewTransaction(nonce, address, amount, 100000, gasPrice, nil)
	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, privateKey)
	if err != nil {
		return nil, fmt.Errorf("cannot sign transaction: %v", err)
	}
	raw, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		return nil, err
	}
	err = client.SendRawTransaction(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("cannot send transaction: %v", err)
	}
	return convertTx(signedTx, fromAddress), nil
}

func convertTx(tx *types.Transaction, from common.Address) *Transaction {
	rtx := &Transaction{}
	rtx.Nonce = tx.Nonce()
	rtx.GasPrice = tx.GasPrice()
	rtx.GasLimit = tx.Gas()
	rtx.To = tx.To()
	rtx.Value = tx.Value()
	rtx.Input = tx.Data()
	rtx.Hash = tx.Hash()
	rtx.From = from
	rtx.V, rtx.R, rtx.S = tx.RawSignatureValues()
	return rtx
}

// ConvertArguments takes the abi Method along with a set of matching arguments and attempts
// to convert the arguments to the appropriate EVM type.
func ConvertArguments(method abi.Method, inputParams []interface{}) ([]interface{}, error) {
	var convertedParams []interface{}
	for i, input := range method.Inputs {
		// fmt.Println("INPUT TYPE:", input.Type.T, "SIZE:", input.Type.Size)
		p := inputParams[i]
		switch input.Type.T {
		case abi.BoolTy:
			val, _ := strconv.ParseBool(p.(string))
			convertedParams = append(convertedParams, val)
		case abi.UintTy:
			val := new(big.Int)
			switch p.(type) {
			case *big.Int:
				val = p.(*big.Int)
				convertedParams = append(convertedParams, convertInt(input.Type, val))
			case float64: // convenient for taking args directly from JSON values
				// if not converted to proper sizes, you get errors like: abi: cannot use ptr as type uint16 as argument
				f := p.(float64)
				f2 := big.NewFloat(f)
				i2, a := f2.Int(nil)
				if a != big.Exact {
					return nil, fmt.Errorf("floating point number %v used which is not valid in web3. Please convert to big.Int.", f)
				}
				convertedParams = append(convertedParams, convertInt(input.Type, i2))
			default:
				fmt.Sscan(p.(string), val)
				convertedParams = append(convertedParams, convertInt(input.Type, val))
			}
			// TODO: case abi.IntTy, just like above
		case abi.AddressTy:
			val := common.HexToAddress(p.(string))
			convertedParams = append(convertedParams, val)
		case abi.StringTy:
			convertedParams = append(convertedParams, p)
		case abi.BytesTy:
			val := p.(string)
			val2 := []byte(val)
			convertedParams = append(convertedParams, val2)
		case abi.FixedBytesTy:
			// slice didn't work, seems to want a fixed array...
			// a := make([]byte, t.Type.Size)
			// return &a, nil
			switch size := input.Type.Size; {
			case size == 32:
				val := p.(string)
				arr := [32]byte{}
				copy(arr[:], val)
				convertedParams = append(convertedParams, arr)
			default:
				return nil, fmt.Errorf("unsupported input byte array size %v", size)
			}
		default:
			return nil, fmt.Errorf("unsupported input type %v", input.Type.T)
		}
	}
	return convertedParams, nil
}

func convertOutputParameter(t abi.Argument) (interface{}, error) {
	switch t.Type.T {
	case abi.BoolTy:
		return new(bool), nil
	case abi.UintTy:
		switch size := t.Type.Size; {
		case size > 64:
			i := new(big.Int)
			return &i, nil
		case size > 32:
			return new(uint64), nil
		case size > 16:
			return new(uint32), nil
		case size > 8:
			return new(uint16), nil
		default:
			return new(uint8), nil
		}
	case abi.IntTy:
		switch size := t.Type.Size; {
		case size > 64:
			i := new(big.Int)
			return &i, nil
		case size > 32:
			return new(int64), nil
		case size > 16:
			return new(int32), nil
		case size > 8:
			return new(int16), nil
		default:
			return new(int8), nil
		}
	case abi.StringTy:
		return new(string), nil
	case abi.AddressTy:
		return new(common.Address), nil
	case abi.BytesTy:
		return new([]byte), nil
	case abi.FixedBytesTy:
		// slice didn't work, seems to want a fixed array...
		// a := make([]byte, t.Type.Size)
		// return &a, nil
		switch size := t.Type.Size; {
		case size == 32:
			return new([32]byte), nil
		default:
			return nil, fmt.Errorf("unsupported output byte array size %v", size)
		}
	default:
		return nil, fmt.Errorf("unsupported output type %v", t.Type.T)
	}
}

func convertInt(t abi.Type, i *big.Int) interface{} {
	switch size := t.Size; {
	case size > 64:
		return i
	case size > 32:
		return i.Uint64()
	case size > 16:
		return uint32(i.Uint64())
	case size > 8:
		return uint16(i.Uint64())
	default:
		return uint8(i.Uint64())
	}
}

// WaitForReceipt polls for a transaction receipt until it is available, or ctx is cancelled.
func WaitForReceipt(ctx context.Context, client Client, hash common.Hash) (*Receipt, error) {
	for {
		receipt, err := client.GetTransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}
		if err != NotFoundErr {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func FindEventById(abi abi.ABI, id common.Hash) *abi.Event {
	for _, event := range abi.Events {
		if event.Id() == id {
			return &event
		}
	}
	return nil
}
func getInputs(args abi.Arguments, indexed bool) []abi.Argument {
	var out []abi.Argument
	for _, arg := range args {
		if arg.Indexed == indexed {
			out = append(out, arg)
		}
	}
	return out
}

// func ParseReceipt(myabi abi.ABI, receipt *Receipt) (map[string]map[string]interface{}, error) {
func ParseLogs(myabi abi.ABI, logs []*types.Log) ([]Event, error) {
	var output []Event
	// output := make(map[string]map[string]interface{})
	for _, log := range logs {
		var out []interface{}
		//event id is always in the first topic
		event := FindEventById(myabi, log.Topics[0])
		fields := make(map[string]interface{})
		nonIndexed := getInputs(event.Inputs, false)
		for _, t := range nonIndexed {
			x, err := convertOutputParameter(t)
			if err != nil {
				return nil, err
			}
			out = append(out, x)
		}
		if len(nonIndexed) > 1 {
			err := myabi.Unpack(&out, event.Name, log.Data)
			if err != nil {
				return nil, err
			}
			for i, o := range out {
				fields[nonIndexed[i].Name] = reflect.ValueOf(o).Elem().Interface()
			}
		} else {
			err := myabi.Unpack(&out[0], event.Name, log.Data)
			if err != nil {
				return nil, err
			}
			fields[nonIndexed[0].Name] = out[0]
		}
		for i, input := range getInputs(event.Inputs, true) {
			fields[input.Name] = log.Topics[i+1].String()
		}
		output = append(output, Event{Name: event.Name, Fields: fields})
	}
	return output, nil
}

// ParseAmount parses a string (human readable amount with units ie 1go, 1nanogo...) and returns big.Int value of this string in wei/atto
func ParseAmount(amount string) (*big.Int, error) {
	var ret = new(big.Int)
	var mul = big.NewInt(1)
	amount = strings.ToLower(amount)
	switch {
	case strings.HasSuffix(amount, "nanogo"):
		amount = strings.TrimSuffix(amount, "nanogo")
		mul = weiPerGwei
	case strings.HasSuffix(amount, "gwei"):
		amount = strings.TrimSuffix(amount, "gwei")
		mul = weiPerGwei
	case strings.HasSuffix(amount, "attogo"):
		amount = strings.TrimSuffix(amount, "attogo")
	case strings.HasSuffix(amount, "wei"):
		amount = strings.TrimSuffix(amount, "wei")
	case strings.HasSuffix(amount, "eth"):
		amount = strings.TrimSuffix(amount, "eth")
		mul = weiPerGO
	default:
		amount = strings.TrimSuffix(amount, "go")
		mul = weiPerGO
	}
	val, err := ParseBigInt(amount)
	if err != nil {
		return nil, err
	}
	return ret.Mul(val, mul), nil
}

// ParseBigInt parses a string (base 10 only) and returns big.Int value of this string in wei/atto
func ParseBigInt(value string) (*big.Int, error) {
	if value == "" {
		return nil, errors.New("Cannot parse empty string")
	}
	i, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, fmt.Errorf("Failed to parse integer %q", value)
	}
	return i, nil
}
