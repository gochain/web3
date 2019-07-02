# GOST

## Transfer Event Hash

```solidity
event TransferEvent(address indexed addr, uint amount);
bytes32 transferEventID = 0xb98a26c1d0427d0e3492e749861c7795bed8f6e7599a65143b5903942e611bb0;

function eventHash(address source, address addr, uint amount) public view returns (bytes32) {
    return keccak256(abi.encode(transferEventID, source, addr, amount));
}
```

## Running Tests

Run `truffle test` from this directory.

# TODO

- [ ] add `truffle test` to circleci?