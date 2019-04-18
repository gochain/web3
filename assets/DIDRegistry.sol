pragma solidity ^0.4.24;

contract DIDRegistry {
	mapping(bytes32 => address) public owners;
	mapping(bytes32 => string) public hashes;

	/*
	 * Registers associates an identifier with an IPFS hash.
	 *
	 * The identifier must have been previously unregistered or the
	 * registration must belong to the message sender.
	 */
	function register(bytes32 identifier, string hash) public {
		address owner = owners[identifier];
		require(owner == 0x0 || owner == msg.sender);
		owners[identifier] = msg.sender;
		hashes[identifier] = hash;
	}

	/*
	 * Returns the owner address of the given identifier.
	 * Returns 0x0 if no owner has claimed the identifier.
	 */
	function owner(bytes32 identifier) public view returns (address) {
		 return owners[identifier];
	}

	/*
	 * Returns the IPFS hash for the given identifier.
	 * Returns an empty string if the identifier has not been registered.
	 */
	function hash(bytes32 identifier) public view returns (string) {
		 return hashes[identifier];
	}
}