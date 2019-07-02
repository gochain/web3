pragma solidity ^0.5.8;

// The Confirmations interface provides methods for confirming events from other networks.
interface Confirmations {
    // The status of an event is None before confirmation is requested, and Pending after.
    // Once a majority consensus is reached, it will transition to Invalid or Confirmed, both of which are final.
    enum Status { None, Pending, Invalid, Confirmed }

    // Get the status for an event.
    function status(uint blockNum, uint logIndex, bytes32 eventHash) external view returns (Status);

    // Request confirmation of an event. Status must be None.
    // Transitions status to Pending and emits a ConfirmationRequested event.
    function requestConfirmation(uint blockNum, uint logIndex, bytes32 eventHash) external;

    // Emitted when confirmation of an event is requested.
    event ConfirmationRequested(uint indexed blockNum, uint indexed logIndex, bytes32 eventHash);

    // Emitted when an event is confirmed by a majority of signers to be either valid or invalid.
    event Confirmed(uint indexed blockNum, uint indexed logIndex, bytes32 eventHash, bool valid);
}

// Transfers manages confirmations and claims for TransferEvents.
contract Transfers {
    // Confirmations from another chain.
    Confirmations confirmations;
    // The contract which emits the TransferEvents.
    address transferContract;
    event TransferEvent(address indexed addr, uint amount);
    bytes32 transferEventID = 0xb98a26c1d0427d0e3492e749861c7795bed8f6e7599a65143b5903942e611bb0;
    // TransferEvents which have been claimed.
    // blockNum=>logIndex=>eventHash
    mapping(uint=>mapping(uint=>mapping(bytes32=>bool))) public claimed;

    constructor(address confirmationsContract, address _transferContract) public {
        confirmations = Confirmations(confirmationsContract);
        transferContract = _transferContract; //TODO must be able to init after deploy
    }

    function emitEvent(address addr, uint amount) public {
        emit TransferEvent(addr, amount);
    }

    function requestConfirmation(uint blockNum, uint logIndex, address addr, uint amount) public {
        confirmations.requestConfirmation(blockNum, logIndex, eventHash(transferContract, addr, amount));
    }

    function status(uint blockNum, uint logIndex, address addr, uint amount) public returns (Confirmations.Status) {
        return confirmations.status(blockNum, logIndex, eventHash(transferContract, addr, amount));
    }

    // Claim a confirmed TransferEvent. Each event may only be claimed once.
    // Only callable by the address in the event, and after the event is confirmed.
    function claim(uint blockNum, uint logIndex, uint amount) public returns (bool) {
        bytes32 eh = eventHash(transferContract, msg.sender, amount);
        if (claimed[blockNum][logIndex][eh]) {
            return false;
        }
        if (confirmations.status(blockNum, logIndex, eh) != Confirmations.Status.Confirmed) {
            return false;
        }
        claimed[blockNum][logIndex][eh] = true;
        return true;
    }

    // Compute the hash for a TransferEvent.
    function eventHash(address source, address addr, uint amount) public view returns (bytes32) {
        return keccak256(abi.encode(transferEventID, source, addr, amount));
    }
}