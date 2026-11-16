// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract Hello {
    string private name;

    constructor() {
        name = "World";
    }

    function hello() public view returns (string memory, string memory) {
        return ("Hello", name);
    }

    function setName(string memory _name) public {
        name = _name;
    }
}
