pragma solidity ^0.5.11;

contract Hello {

    string name;

    /* This runs when the contract is executed */
    constructor() public {
        name = "World";
    }

    function hello() public view returns (string memory, string memory) {
        return ("Hello", name);
    }

    function setName(string memory _name) public {
        name = _name;
    }
}