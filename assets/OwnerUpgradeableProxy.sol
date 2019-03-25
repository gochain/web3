pragma solidity ^0.4.24;

import "./UpgradeableProxy.sol";

/*
 * OwnerUpgradeableProxy is an upgradeable proxy that only allows the contract
 * owner to upgrade and pause the proxy.
 */
contract OwnerUpgradeableProxy is UpgradeableProxy {
  bytes32 private constant ownerPosition = keccak256("gochain.proxy.owner");

  /*
   * Initializes the proxy and sets the owner.
   */
  constructor() public {
    _setOwner(msg.sender);
  }

  /*
   * Restricts a function to only allow execution by the proxy owner.
   */
  modifier ownerOnly() {
    require(msg.sender == owner());
    _;
  }

  /*
   * Returns the owner of the proxy contract.
   */
  function owner() public view returns (address addr) {
    bytes32 pos = ownerPosition;
    assembly {
      addr := sload(pos)
    }
  }

  /*
   * Sets the owner of the contract.
   */
  function _setOwner(address addr) internal {
    bytes32 pos = ownerPosition;
    assembly {
      sstore(pos, addr)
    }
  }

  /*
   * Upgrades the contract to a new target address. Only allowed by the owner.
   */
  function upgrade(address target) public ownerOnly {
    _upgrade(target);
  }

  /*
   * Pauses the contract and does not allow functions to be executed besides
   * declared functions directly on the proxy (e.g. upgrade(), resume()).
   */
  function pause() public ownerOnly {
    _pause();
  }

  /*
   * Resumes a previously paused contract.
   */
  function resume() public ownerOnly {
    _resume();
  }
}
