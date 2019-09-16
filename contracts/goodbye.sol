pragma solidity ^0.5.11;

contract Goodbye {

    string name;

    /* This runs when the contract is executed */
    constructor() public {
        name = "World";
    }

    function hello() public view returns (string memory, string memory) {
        return ("Goodbye", name);
    }

    function setName(string memory _name) public {
        name = _name;
    }
}