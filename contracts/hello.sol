pragma solidity ^0.4.24;

contract HelloWorld {
    
    string name;

    /* This runs when the contract is executed */
    constructor() public {
        name = "World";
    }

    function hello() public view returns (string, string) {
        return ("Hello", name);
    }

    function setName(string _name) public {
        name = _name;
    }
}