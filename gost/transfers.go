// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package gost

import (
	"math/big"
	"strings"

	"github.com/gochain-io/gochain/v3"
	"github.com/gochain-io/gochain/v3/accounts/abi"
	"github.com/gochain-io/gochain/v3/accounts/abi/bind"
	"github.com/gochain-io/gochain/v3/common"
	"github.com/gochain-io/gochain/v3/core/types"
	"github.com/gochain-io/gochain/v3/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = big.NewInt
	_ = strings.NewReader
	_ = gochain.NotFound
	_ = abi.U256
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// TransfersABI is the input ABI used to generate the binding from.
const TransfersABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"},{\"name\":\"\",\"type\":\"uint256\"},{\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"claimed\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"addr\",\"type\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"emitEvent\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"source\",\"type\":\"address\"},{\"name\":\"addr\",\"type\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"eventHash\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"blockNum\",\"type\":\"uint256\"},{\"name\":\"logIndex\",\"type\":\"uint256\"},{\"name\":\"addr\",\"type\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"requestConfirmation\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"blockNum\",\"type\":\"uint256\"},{\"name\":\"logIndex\",\"type\":\"uint256\"},{\"name\":\"addr\",\"type\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"status\",\"outputs\":[{\"name\":\"\",\"type\":\"uint8\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"blockNum\",\"type\":\"uint256\"},{\"name\":\"logIndex\",\"type\":\"uint256\"},{\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"claim\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"confirmationsContract\",\"type\":\"address\"},{\"name\":\"_transferContract\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"addr\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"TransferEvent\",\"type\":\"event\"}]"

// TransfersBin is the compiled bytecode used for deploying new contracts.
const TransfersBin = `0x60806040527fb98a26c1d0427d0e3492e749861c7795bed8f6e7599a65143b5903942e611bb060025534801561003457600080fd5b506040516040806105a78339810180604052604081101561005457600080fd5b508051602090910151600080546001600160a01b039384166001600160a01b0319918216179091556001805493909216921691909117905561050c8061009b6000396000f3fe608060405234801561001057600080fd5b50600436106100625760003560e01c80630929277c14610067578063143a4d7d146100a457806345cbee0b146100d2578063999521d31461011a578063d6a2129214610152578063efa9a9be146101ae575b600080fd5b6100906004803603606081101561007d57600080fd5b50803590602081013590604001356101d7565b604080519115158252519081900360200190f35b6100d0600480360360408110156100ba57600080fd5b506001600160a01b0381351690602001356101fd565b005b610108600480360360608110156100e857600080fd5b506001600160a01b03813581169160208101359091169060400135610240565b60408051918252519081900360200190f35b6100d06004803603608081101561013057600080fd5b508035906020810135906001600160a01b03604082013516906060013561028d565b61018a6004803603608081101561016857600080fd5b508035906020810135906001600160a01b036040820135169060600135610317565b6040518082600381111561019a57fe5b60ff16815260200191505060405180910390f35b610090600480360360608110156101c457600080fd5b50803590602081013590604001356103b7565b600360209081526000938452604080852082529284528284209052825290205460ff1681565b6040805182815290516001600160a01b038416917fb98a26c1d0427d0e3492e749861c7795bed8f6e7599a65143b5903942e611bb0919081900360200190a25050565b600254604080516020808201939093526001600160a01b03808716828401528516606082015260808082018590528251808303909101815260a090910190915280519101205b9392505050565b6000546001546001600160a01b0391821691631662869f91879187916102b591168787610240565b6040518463ffffffff1660e01b8152600401808481526020018381526020018281526020019350505050600060405180830381600087803b1580156102f957600080fd5b505af115801561030d573d6000803e3d6000fd5b5050505050505050565b600080546001546001600160a01b039182169163fad3ffd6918891889161034091168888610240565b6040518463ffffffff1660e01b815260040180848152602001838152602001828152602001935050505060206040518083038186803b15801561038257600080fd5b505afa158015610396573d6000803e3d6000fd5b505050506040513d60208110156103ac57600080fd5b505195945050505050565b60015460009081906103d3906001600160a01b03163385610240565b6000868152600360209081526040808320888452825280832084845290915290205490915060ff161561040a576000915050610286565b60005460408051600160e11b637d69ffeb02815260048101889052602481018790526044810184905290516003926001600160a01b03169163fad3ffd6916064808301926020929190829003018186803b15801561046757600080fd5b505afa15801561047b573d6000803e3d6000fd5b505050506040513d602081101561049157600080fd5b5051600381111561049e57fe5b146104ad576000915050610286565b60009485526003602090815260408087209587529481528486209186525250509020805460ff191660019081179091559056fea165627a7a723058203c56a4e0c63030e5a991d806e0e939cee47f9b78eb25426c410abc3c2d88ab660029`

// DeployTransfers deploys a new GoChain contract, binding an instance of Transfers to it.
func DeployTransfers(auth *bind.TransactOpts, backend bind.ContractBackend, confirmationsContract common.Address, _transferContract common.Address) (common.Address, *types.Transaction, *Transfers, error) {
	parsed, err := abi.JSON(strings.NewReader(TransfersABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(TransfersBin), backend, confirmationsContract, _transferContract)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Transfers{TransfersCaller: TransfersCaller{contract: contract}, TransfersTransactor: TransfersTransactor{contract: contract}, TransfersFilterer: TransfersFilterer{contract: contract}}, nil
}

// Transfers is an auto generated Go binding around an GoChain contract.
type Transfers struct {
	TransfersCaller     // Read-only binding to the contract
	TransfersTransactor // Write-only binding to the contract
	TransfersFilterer   // Log filterer for contract events
}

// TransfersCaller is an auto generated read-only Go binding around an GoChain contract.
type TransfersCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TransfersTransactor is an auto generated write-only Go binding around an GoChain contract.
type TransfersTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TransfersFilterer is an auto generated log filtering Go binding around an GoChain contract events.
type TransfersFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TransfersSession is an auto generated Go binding around an GoChain contract,
// with pre-set call and transact options.
type TransfersSession struct {
	Contract     *Transfers        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TransfersCallerSession is an auto generated read-only Go binding around an GoChain contract,
// with pre-set call options.
type TransfersCallerSession struct {
	Contract *TransfersCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// TransfersTransactorSession is an auto generated write-only Go binding around an GoChain contract,
// with pre-set transact options.
type TransfersTransactorSession struct {
	Contract     *TransfersTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// TransfersRaw is an auto generated low-level Go binding around an GoChain contract.
type TransfersRaw struct {
	Contract *Transfers // Generic contract binding to access the raw methods on
}

// TransfersCallerRaw is an auto generated low-level read-only Go binding around an GoChain contract.
type TransfersCallerRaw struct {
	Contract *TransfersCaller // Generic read-only contract binding to access the raw methods on
}

// TransfersTransactorRaw is an auto generated low-level write-only Go binding around an GoChain contract.
type TransfersTransactorRaw struct {
	Contract *TransfersTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTransfers creates a new instance of Transfers, bound to a specific deployed contract.
func NewTransfers(address common.Address, backend bind.ContractBackend) (*Transfers, error) {
	contract, err := bindTransfers(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Transfers{TransfersCaller: TransfersCaller{contract: contract}, TransfersTransactor: TransfersTransactor{contract: contract}, TransfersFilterer: TransfersFilterer{contract: contract}}, nil
}

// NewTransfersCaller creates a new read-only instance of Transfers, bound to a specific deployed contract.
func NewTransfersCaller(address common.Address, caller bind.ContractCaller) (*TransfersCaller, error) {
	contract, err := bindTransfers(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TransfersCaller{contract: contract}, nil
}

// NewTransfersTransactor creates a new write-only instance of Transfers, bound to a specific deployed contract.
func NewTransfersTransactor(address common.Address, transactor bind.ContractTransactor) (*TransfersTransactor, error) {
	contract, err := bindTransfers(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TransfersTransactor{contract: contract}, nil
}

// NewTransfersFilterer creates a new log filterer instance of Transfers, bound to a specific deployed contract.
func NewTransfersFilterer(address common.Address, filterer bind.ContractFilterer) (*TransfersFilterer, error) {
	contract, err := bindTransfers(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TransfersFilterer{contract: contract}, nil
}

// bindTransfers binds a generic wrapper to an already deployed contract.
func bindTransfers(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(TransfersABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Transfers *TransfersRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Transfers.Contract.TransfersCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Transfers *TransfersRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Transfers.Contract.TransfersTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Transfers *TransfersRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Transfers.Contract.TransfersTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Transfers *TransfersCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Transfers.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Transfers *TransfersTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Transfers.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Transfers *TransfersTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Transfers.Contract.contract.Transact(opts, method, params...)
}

// Claimed is a free data retrieval call binding the contract method 0x0929277c.
//
// Solidity: function claimed(uint256 , uint256 , bytes32 ) constant returns(bool)
func (_Transfers *TransfersCaller) Claimed(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int, arg2 [32]byte) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _Transfers.contract.Call(opts, out, "claimed", arg0, arg1, arg2)
	return *ret0, err
}

// Claimed is a free data retrieval call binding the contract method 0x0929277c.
//
// Solidity: function claimed(uint256 , uint256 , bytes32 ) constant returns(bool)
func (_Transfers *TransfersSession) Claimed(arg0 *big.Int, arg1 *big.Int, arg2 [32]byte) (bool, error) {
	return _Transfers.Contract.Claimed(&_Transfers.CallOpts, arg0, arg1, arg2)
}

// Claimed is a free data retrieval call binding the contract method 0x0929277c.
//
// Solidity: function claimed(uint256 , uint256 , bytes32 ) constant returns(bool)
func (_Transfers *TransfersCallerSession) Claimed(arg0 *big.Int, arg1 *big.Int, arg2 [32]byte) (bool, error) {
	return _Transfers.Contract.Claimed(&_Transfers.CallOpts, arg0, arg1, arg2)
}

// EventHash is a free data retrieval call binding the contract method 0x45cbee0b.
//
// Solidity: function eventHash(address source, address addr, uint256 amount) constant returns(bytes32)
func (_Transfers *TransfersCaller) EventHash(opts *bind.CallOpts, source common.Address, addr common.Address, amount *big.Int) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _Transfers.contract.Call(opts, out, "eventHash", source, addr, amount)
	return *ret0, err
}

// EventHash is a free data retrieval call binding the contract method 0x45cbee0b.
//
// Solidity: function eventHash(address source, address addr, uint256 amount) constant returns(bytes32)
func (_Transfers *TransfersSession) EventHash(source common.Address, addr common.Address, amount *big.Int) ([32]byte, error) {
	return _Transfers.Contract.EventHash(&_Transfers.CallOpts, source, addr, amount)
}

// EventHash is a free data retrieval call binding the contract method 0x45cbee0b.
//
// Solidity: function eventHash(address source, address addr, uint256 amount) constant returns(bytes32)
func (_Transfers *TransfersCallerSession) EventHash(source common.Address, addr common.Address, amount *big.Int) ([32]byte, error) {
	return _Transfers.Contract.EventHash(&_Transfers.CallOpts, source, addr, amount)
}

// Claim is a paid mutator transaction binding the contract method 0xefa9a9be.
//
// Solidity: function claim(uint256 blockNum, uint256 logIndex, uint256 amount) returns(bool)
func (_Transfers *TransfersTransactor) Claim(opts *bind.TransactOpts, blockNum *big.Int, logIndex *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _Transfers.contract.Transact(opts, "claim", blockNum, logIndex, amount)
}

// Claim is a paid mutator transaction binding the contract method 0xefa9a9be.
//
// Solidity: function claim(uint256 blockNum, uint256 logIndex, uint256 amount) returns(bool)
func (_Transfers *TransfersSession) Claim(blockNum *big.Int, logIndex *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _Transfers.Contract.Claim(&_Transfers.TransactOpts, blockNum, logIndex, amount)
}

// Claim is a paid mutator transaction binding the contract method 0xefa9a9be.
//
// Solidity: function claim(uint256 blockNum, uint256 logIndex, uint256 amount) returns(bool)
func (_Transfers *TransfersTransactorSession) Claim(blockNum *big.Int, logIndex *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _Transfers.Contract.Claim(&_Transfers.TransactOpts, blockNum, logIndex, amount)
}

// EmitEvent is a paid mutator transaction binding the contract method 0x143a4d7d.
//
// Solidity: function emitEvent(address addr, uint256 amount) returns()
func (_Transfers *TransfersTransactor) EmitEvent(opts *bind.TransactOpts, addr common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Transfers.contract.Transact(opts, "emitEvent", addr, amount)
}

// EmitEvent is a paid mutator transaction binding the contract method 0x143a4d7d.
//
// Solidity: function emitEvent(address addr, uint256 amount) returns()
func (_Transfers *TransfersSession) EmitEvent(addr common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Transfers.Contract.EmitEvent(&_Transfers.TransactOpts, addr, amount)
}

// EmitEvent is a paid mutator transaction binding the contract method 0x143a4d7d.
//
// Solidity: function emitEvent(address addr, uint256 amount) returns()
func (_Transfers *TransfersTransactorSession) EmitEvent(addr common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Transfers.Contract.EmitEvent(&_Transfers.TransactOpts, addr, amount)
}

// RequestConfirmation is a paid mutator transaction binding the contract method 0x999521d3.
//
// Solidity: function requestConfirmation(uint256 blockNum, uint256 logIndex, address addr, uint256 amount) returns()
func (_Transfers *TransfersTransactor) RequestConfirmation(opts *bind.TransactOpts, blockNum *big.Int, logIndex *big.Int, addr common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Transfers.contract.Transact(opts, "requestConfirmation", blockNum, logIndex, addr, amount)
}

// RequestConfirmation is a paid mutator transaction binding the contract method 0x999521d3.
//
// Solidity: function requestConfirmation(uint256 blockNum, uint256 logIndex, address addr, uint256 amount) returns()
func (_Transfers *TransfersSession) RequestConfirmation(blockNum *big.Int, logIndex *big.Int, addr common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Transfers.Contract.RequestConfirmation(&_Transfers.TransactOpts, blockNum, logIndex, addr, amount)
}

// RequestConfirmation is a paid mutator transaction binding the contract method 0x999521d3.
//
// Solidity: function requestConfirmation(uint256 blockNum, uint256 logIndex, address addr, uint256 amount) returns()
func (_Transfers *TransfersTransactorSession) RequestConfirmation(blockNum *big.Int, logIndex *big.Int, addr common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Transfers.Contract.RequestConfirmation(&_Transfers.TransactOpts, blockNum, logIndex, addr, amount)
}

// Status is a paid mutator transaction binding the contract method 0xd6a21292.
//
// Solidity: function status(uint256 blockNum, uint256 logIndex, address addr, uint256 amount) returns(uint8)
func (_Transfers *TransfersTransactor) Status(opts *bind.TransactOpts, blockNum *big.Int, logIndex *big.Int, addr common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Transfers.contract.Transact(opts, "status", blockNum, logIndex, addr, amount)
}

// Status is a paid mutator transaction binding the contract method 0xd6a21292.
//
// Solidity: function status(uint256 blockNum, uint256 logIndex, address addr, uint256 amount) returns(uint8)
func (_Transfers *TransfersSession) Status(blockNum *big.Int, logIndex *big.Int, addr common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Transfers.Contract.Status(&_Transfers.TransactOpts, blockNum, logIndex, addr, amount)
}

// Status is a paid mutator transaction binding the contract method 0xd6a21292.
//
// Solidity: function status(uint256 blockNum, uint256 logIndex, address addr, uint256 amount) returns(uint8)
func (_Transfers *TransfersTransactorSession) Status(blockNum *big.Int, logIndex *big.Int, addr common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Transfers.Contract.Status(&_Transfers.TransactOpts, blockNum, logIndex, addr, amount)
}

// TransfersTransferEventIterator is returned from FilterTransferEvent and is used to iterate over the raw logs and unpacked data for TransferEvent events raised by the Transfers contract.
type TransfersTransferEventIterator struct {
	Event *TransfersTransferEvent // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log       // Log channel receiving the found contract events
	sub  gochain.Subscription // Subscription for errors, completion and termination
	done bool                 // Whether the subscription completed delivering logs
	fail error                // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TransfersTransferEventIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TransfersTransferEvent)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TransfersTransferEvent)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TransfersTransferEventIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TransfersTransferEventIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TransfersTransferEvent represents a TransferEvent event raised by the Transfers contract.
type TransfersTransferEvent struct {
	Addr   common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterTransferEvent is a free log retrieval operation binding the contract event 0xb98a26c1d0427d0e3492e749861c7795bed8f6e7599a65143b5903942e611bb0.
//
// Solidity: event TransferEvent(address indexed addr, uint256 amount)
func (_Transfers *TransfersFilterer) FilterTransferEvent(opts *bind.FilterOpts, addr []common.Address) (*TransfersTransferEventIterator, error) {

	var addrRule []interface{}
	for _, addrItem := range addr {
		addrRule = append(addrRule, addrItem)
	}

	logs, sub, err := _Transfers.contract.FilterLogs(opts, "TransferEvent", addrRule)
	if err != nil {
		return nil, err
	}
	return &TransfersTransferEventIterator{contract: _Transfers.contract, event: "TransferEvent", logs: logs, sub: sub}, nil
}

// WatchTransferEvent is a free log subscription operation binding the contract event 0xb98a26c1d0427d0e3492e749861c7795bed8f6e7599a65143b5903942e611bb0.
//
// Solidity: event TransferEvent(address indexed addr, uint256 amount)
func (_Transfers *TransfersFilterer) WatchTransferEvent(opts *bind.WatchOpts, sink chan<- *TransfersTransferEvent, addr []common.Address) (event.Subscription, error) {

	var addrRule []interface{}
	for _, addrItem := range addr {
		addrRule = append(addrRule, addrItem)
	}

	logs, sub, err := _Transfers.contract.WatchLogs(opts, "TransferEvent", addrRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TransfersTransferEvent)
				if err := _Transfers.contract.UnpackLog(event, "TransferEvent", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}
