pragma solidity ^0.5.8;

import "truffle/Assert.sol";
import "truffle/DeployedAddresses.sol";
import "../contracts/token.sol";

contract TestTransfers {
    //TODO test request/verify confirmation
    //TODO test claim, then reject second claim
    //TODO test emit event

    event EventHash(bytes32 got);
    function testEventhash() public {
        Transfers transfers = new Transfers(0x0000000000000000000000000000000000000000, 0x0000000000000000000000000000000000000000);

        address source = 0x0000000000000000000000000000000000001234;

        bytes32 hash = transfers.eventHash(source, 0x0000000000000000000000000000000000000001, 1);
        emit EventHash(hash);
        Assert.equal(hash, 0xb0ac7d1bcc67772d396f1ef33b61e468a6af7bba40e9ee94fa3bb0f11762e033,
            "Hashes should match");
        hash = transfers.eventHash(source, 0x000000000000000000000000000000000000000F, 10);
        emit EventHash(hash);
        Assert.equal(hash, 0xe7c9c57479a7a81885e5d86e214b54bbe5ff61229ddfc03c367163aa34a1ebe5,
            "Hashes should match");
        hash = transfers.eventHash(source, 0x000000000000000000000000000000000000000A, 1000);
        emit EventHash(hash);
        Assert.equal(hash, 0x8ae9bcae78fe3be388b42965ef220c45cec3891757bbfc4c1f1b003ddd4e0b7c,
            "Hashes should match");
        hash = transfers.eventHash(source, 0x0000000000000000000000000000000000000009, 1000000000000000000);
        emit EventHash(hash);
        Assert.equal(hash, 0x42d355b46a5cc012e09789a9bd092e436ad2b5b62851c1a548c58dcdb35ac68f,
            "Hashes should match");
    }
}

