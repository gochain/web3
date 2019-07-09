pragma solidity ^0.5.8;

import "../../lib/oz/contracts/token/ERC20/ERC20.sol";
import "./transfers.sol";

contract ProxyToken {
    Transfers ethTransfers;

    constructor(address confirmationsContract) public {
        //TODO init transfer contract address after deploy...
        ethTransfers = new Transfers(confirmationsContract, 0x0f6cEF2b7FbB504782e35AA82a2207e816a2B7a9);
    }

    function ethTransferStatus(uint blockNum, uint logIndex, uint amount) public returns (Confirmations.Status) {
        return ethTransfers.status(blockNum, logIndex, msg.sender, amount);
    }

    function ethTransferConfirm(uint blockNum, uint logIndex, uint amount) public {
        ethTransfers.requestConfirmation(blockNum, logIndex, msg.sender, amount);
    }

    function ethTransferClaim(uint blockNum, uint logIndex, uint amount) public {
        if (ethTransfers.claim(blockNum, logIndex, amount)) {
            _mint(amount);
        }
    }
    //TODO param to send to address other than msg.sender?
    function transferToETH(uint amount) public {
        _burn(amount);
        ethTransfers.emitEvent(msg.sender, amount);
    }

    function _mint(uint) internal {}
    function _burn(uint) internal {}
}

contract TokenCustody {
    Transfers goTransfers;
    ERC20 token;

    constructor(address tokenContract, address confirmationsContract) public {
        token = ERC20(tokenContract);
        //TODO init transfer contract address after deploy...
        goTransfers = new Transfers(confirmationsContract, 0x0f6cEF2b7FbB504782e35AA82a2207e816a2B7a9);
    }

    //TODO param to send to address other than msg.sender?
    function transferToGO(uint amount) public {
        require(token.transferFrom(msg.sender, address(this), amount));
        goTransfers.emitEvent(msg.sender, amount);
    }

    function goTransferStatus(uint blockNum, uint logIndex, uint amount) public returns (Confirmations.Status) {
        return goTransfers.status(blockNum, logIndex, msg.sender, amount);
    }

    function goTransferConfirm(uint blockNum, uint logIndex, uint amount) public {
        goTransfers.requestConfirmation(blockNum, logIndex, msg.sender, amount);
    }

    function goTransferClaim(uint blockNum, uint logIndex, uint amount) public {
        if (goTransfers.claim(blockNum, logIndex, amount)) {
            token.transfer(msg.sender, amount);
        }
    }
}