pragma solidity ^0.5.11;

contract Types {

    string name;
    bytes32 public tbytes32;
    bytes public tbytes;
    string public tstring;
    uint256 public tuint256;
    uint8 public tuint8;

    /* This runs when the contract is executed */
    constructor() public {
        name = "World";
        tbytes32 = "i'm bytes";
        tbytes = "this is a long string with variable length, so what are we going to do about it?";
        tstring = "this is another long string with variable length that doesn't fit into bytes32";
    }

    function hello() public view returns (string memory, string memory) {
        return ("Hello", name);
    }

    function setName(string memory _name) public {
        name = _name;
    }

    function setBytes32(bytes32 x) public {
        tbytes32 = x;
    }
    
    function setBytes(bytes memory x) public {
        tbytes = x;
    }

    function setUint256(uint256 x) public {
        tuint256 = x;
    }

    function setUint8(uint8 x) public {
        tuint8 = x;
    }
}