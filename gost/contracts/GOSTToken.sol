pragma solidity ^0.5.8;

import "../../lib/oz/contracts/token/ERC20/ERC20Detailed.sol";
import "../../lib/oz/contracts/token/ERC20/ERC20Burnable.sol";
import "../../lib/oz/contracts/token/ERC20/ERC20Mintable.sol";
import "./token.sol";

contract GOSTToken is ERC20Burnable, ERC20Mintable, ERC20Detailed, ProxyToken {
    constructor() ERC20Detailed("GOST", "GOST", 18) public {}
}
