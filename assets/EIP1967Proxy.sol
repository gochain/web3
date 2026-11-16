// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/**
 * @title EIP1967UpgradeableProxy
 * @notice Standard EIP-1967 upgradeable proxy with optional initialization calldata and emergency pause support.
 */
contract EIP1967UpgradeableProxy {
    // bytes32(uint256(keccak256("eip1967.proxy.implementation")) - 1)
    bytes32 internal constant IMPLEMENTATION_SLOT =
        0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc;

    // bytes32(uint256(keccak256("eip1967.proxy.admin")) - 1)
    bytes32 internal constant ADMIN_SLOT =
        0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103;

    // bytes32(uint256(keccak256("eip1967.proxy.paused")) - 1)
    bytes32 internal constant PAUSED_SLOT =
        0x8dea8703c3cf94703383ce38a9c894669dccd4ca8e65ddb43267aa0248711450;

    event Upgraded(address indexed implementation);
    event AdminChanged(address previousAdmin, address newAdmin);
    event Paused(address account);
    event Unpaused(address account);

    error ProxyUnauthorized();
    error ProxyInvalidImplementation(address implementation);
    error ProxyInvalidAdmin();
    error ProxyPaused();

    constructor(address initialImplementation, address initialAdmin, bytes memory data) payable {
        address adminAddr = initialAdmin == address(0) ? msg.sender : initialAdmin;
        _setAdmin(adminAddr);
        _upgradeToAndCall(initialImplementation, data);
    }

    modifier onlyAdmin() {
        if (msg.sender != _getAdmin()) revert ProxyUnauthorized();
        _;
    }

    function implementation() external view returns (address) {
        return _getImplementation();
    }

    function target() external view returns (address) {
        return _getImplementation();
    }

    function admin() external view returns (address) {
        return _getAdmin();
    }

    function paused() external view returns (bool) {
        return _isPaused();
    }

    function upgradeTo(address newImplementation) external onlyAdmin {
        _upgradeToAndCall(newImplementation, "");
    }

    function upgrade(address newImplementation) external onlyAdmin {
        _upgradeToAndCall(newImplementation, "");
    }

    function upgradeToAndCall(address newImplementation, bytes calldata data) external payable onlyAdmin {
        _upgradeToAndCall(newImplementation, data);
    }

    function changeAdmin(address newAdmin) external onlyAdmin {
        if (newAdmin == address(0)) revert ProxyInvalidAdmin();
        _setAdmin(newAdmin);
    }

    function pause() external onlyAdmin {
        bytes32 slot = PAUSED_SLOT;
        assembly {
            sstore(slot, 1)
        }
        emit Paused(msg.sender);
    }

    function resume() external onlyAdmin {
        bytes32 slot = PAUSED_SLOT;
        assembly {
            sstore(slot, 0)
        }
        emit Unpaused(msg.sender);
    }

    function _getImplementation() internal view returns (address impl) {
        bytes32 slot = IMPLEMENTATION_SLOT;
        assembly {
            impl := sload(slot)
        }
    }

    function _getAdmin() internal view returns (address adm) {
        bytes32 slot = ADMIN_SLOT;
        assembly {
            adm := sload(slot)
        }
    }

    function _isPaused() internal view returns (bool val) {
        bytes32 slot = PAUSED_SLOT;
        uint256 raw;
        assembly {
            raw := sload(slot)
        }
        val = raw != 0;
    }

    function _setAdmin(address newAdmin) internal {
        address prev = _getAdmin();
        bytes32 slot = ADMIN_SLOT;
        assembly {
            sstore(slot, newAdmin)
        }
        emit AdminChanged(prev, newAdmin);
    }

    function _upgradeToAndCall(address newImplementation, bytes memory data) internal {
        if (newImplementation.code.length == 0) {
            revert ProxyInvalidImplementation(newImplementation);
        }
        bytes32 slot = IMPLEMENTATION_SLOT;
        assembly {
            sstore(slot, newImplementation)
        }
        emit Upgraded(newImplementation);

        if (data.length > 0) {
            (bool ok, bytes memory ret) = newImplementation.delegatecall(data);
            if (!ok) {
                assembly {
                    revert(add(ret, 0x20), mload(ret))
                }
            }
        }
    }

    function _delegate(address impl) internal {
        if (_isPaused()) revert ProxyPaused();
        if (impl == address(0)) revert ProxyInvalidImplementation(address(0));
        assembly {
            calldatacopy(0, 0, calldatasize())
            let result := delegatecall(gas(), impl, 0, calldatasize(), 0, 0)
            returndatacopy(0, 0, returndatasize())
            switch result
            case 0 {
                revert(0, returndatasize())
            }
            default {
                return(0, returndatasize())
            }
        }
    }

    fallback() external payable {
        _delegate(_getImplementation());
    }

    receive() external payable {
        _delegate(_getImplementation());
    }
}
