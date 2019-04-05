pragma solidity ^0.4.24;

/*
 * UpgradeableProxy is the base contract for all upgradeable contracts.
 * It implements proxy functionality, internal upgrade mechanisms, and internal
 * pause/resume mechanisms.
 *
 * Implementations must handle the specific upgrade & pause rules.
 */
contract UpgradeableProxy {
  event Upgraded(address indexed target);
  event Paused();
  event Resumed();

  bytes32 private constant targetPosition = keccak256("gochain.proxy.target");
  bytes32 private constant pausedPosition = keccak256("gochain.proxy.paused");

  /*
   * Initializes the starting target contract address. The placeholder 
   * address is replaced during deployment to the correct address.
   */
  constructor() public {
    address initialTarget = 0xEEfFEEffeEffeeFFeeffeeffeEfFeEffeEFfEeff;
    _upgrade(initialTarget);
  }

  /*
   * Returns the contract address that is currently being proxied to.
   */
  function target() public view returns (address addr) {
    bytes32 pos = targetPosition;
    assembly {
      addr := sload(pos)
    }
  }

  /*
   * Abstract declaration of upgrade function.
   */
  function upgrade(address addr) public;

  /*
   * Updates the target contract address.
   */
  function _upgrade(address addr) internal {
    address current = target();
    require(current != addr);
    bytes32 pos = targetPosition;
    assembly {
      sstore(pos, addr)
    }
    emit Upgraded(addr);
  }

  /*
   * Returns whether the contract is currently paused.
   */
  function paused() public view returns (bool val) {
    bytes32 pos = pausedPosition;
    bytes32 val32 = 0;
    assembly {
      val32 := sload(pos)
    }
    val = val32 != 0;
  }

  /*
   * Abstract declaration of pause function.
   */
  function pause() public;

  /*
   * Abstract declaration of resume function.
   */
  function resume() public;

  /*
   * Marks the contract as paused.
   */
  function _pause() internal {
    bytes32 pos = pausedPosition;
    bytes1 val = 1;
    assembly {
      sstore(pos, val)
    }
    emit Paused();
  }

  /*
   * Marks the contract as resumed (aka unpaused).
   */
  function _resume() internal {
    bytes32 pos = pausedPosition;
    bytes1 val = 0;
    assembly {
      sstore(pos, val)
    }
    emit Resumed();
  }

  /*
   * Passthrough function for all function calls that cannot be found.
   * Functions are delegated to the target contract but maintain the local storage.
   */
  function() payable public {
    bool _paused = paused();
    require(!_paused);

    address _target = target();
    require(_target != address(0));

    assembly {
      let ptr := mload(0x40)
      calldatacopy(ptr, 0, calldatasize)
      let result := delegatecall(gas, _target, ptr, calldatasize, 0, 0)
      let size := returndatasize
      returndatacopy(ptr, 0, size)

      switch result
      case 0 { revert(ptr, size) }
      default { return(ptr, size) }
    }
  }
}
