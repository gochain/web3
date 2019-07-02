package gost

//go:generate abigen --lang go --abi contracts/Transfers.abi --bin contracts/Transfers.bin --pkg gost --type transfers --out transfers.go

import (
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/gochain-io/gochain/v3/accounts/abi"
	"github.com/gochain-io/gochain/v3/common"
	"github.com/gochain-io/gochain/v3/crypto"
)

var (
	transfersABI    abi.ABI
	transferEvent   abi.Event
	transferEventID common.Hash
	hashTy          abi.Type
	addressTy       abi.Type
	uintTy          abi.Type
)

func init() {
	var err error
	transfersABI, err = abi.JSON(strings.NewReader(TransfersABI))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	transferEvent = transfersABI.Events["TransferEvent"]
	transferEventID = transferEvent.Id()
	hashTy, _ = abi.NewType("bytes32")
	addressTy, _ = abi.NewType("address")
	uintTy, _ = abi.NewType("uint256")
}

//TODO doc
func TransferEventHash(sourceContract common.Address, addr common.Address, amount *big.Int) (common.Hash, error) {
	//TODO cross.EventHasher(transferEvent)(source, addr, amount)
	args := abi.Arguments{
		abi.Argument{
			Type: hashTy,
		},
		abi.Argument{
			Type: addressTy,
		},
		abi.Argument{
			Type: addressTy,
		},
		abi.Argument{
			Type: uintTy,
		},
	}
	b, err := args.Pack(transferEventID, sourceContract, addr, amount)
	if err != nil {
		return common.Hash{}, err
	}
	return crypto.Keccak256Hash(b), nil
}
