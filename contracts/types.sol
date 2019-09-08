pragma solidity ^0.5.3;

contract Types {

    string name;
    bytes32 public tbytes32;
    bytes public tbytes;
    string public tstring;

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
}