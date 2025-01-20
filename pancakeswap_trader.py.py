import time
from web3 import Web3
from web3.middleware import geth_poa_middleware
from decimal import Decimal

# Custom BSC RPC URL
BSC_RPC_URL = 'https://services.tokenview.io/vipapi/nodeservice/bsc?apikey=gVFJX5OyPdc2kHH7youg'

# Connect to BSC using Web3
w3 = Web3(Web3.HTTPProvider(BSC_RPC_URL))

# Middleware for BSC (if you're using a chain with Proof of Authority)
w3.middleware_stack.inject(geth_poa_middleware, layer=0)

# Your wallet private key and address (never share the private key!)
private_key = '0x21fa1bf8dc9793971382c89776e623f9177e4e30b24537d1b2f9383dc46a00c6'
address= 0x97293ceab815896883e8200aef5a4581a70504b2        
w3.eth.account.privateKeyToAccount(private_key).address

# PancakeSwap Router and Token Addresses
PANCAKE_ROUTER_ADDRESS = '0x05fF0d2460D4d8ddD1F75808c55B0c94b41F63b8'  # PancakeSwap Router contract address
WBNB_ADDRESS = '0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c'  # WBNB Token Address
BOOM_ADDRESS = '0xcd6a51559254030ca30c2fb2cbdf5c492e8caf9c'  # BOOM Token Address

# PancakeSwap Router ABI (simplified version for this example)
router_abi = [
    {
        "constant": False,
        "inputs": [
            {
                "name": "amountOutMin",
                "type": "uint256"
            },
            {
                "name": "path",
                "type": "address[]"
            },
            {
                "name": "to",
                "type": "address"
            },
            {
                "name": "deadline",
                "type": "uint256"
            }
        ],
        "name": "swapExactETHForTokens",
        "outputs": [
            {
                "name": "",
                "type": "uint256[]"
            }
        ],
        "payable": True,
        "stateMutability": "payable",
        "type": "function"
    },
    # Add other necessary methods to interact with the router here.
]

# Create contract instances
router_contract = w3.eth.contract(address=PANCAKE_ROUTER_ADDRESS, abi=router_abi)

# Function to buy BOOM token using WBNB
def buy_boom(amount_in_wbnb):
    # Calculate amount out (minimum expected BOOM token amount)
    amount_out_min = 0  # Use a slippage percentage or use an API to estimate slippage
    
    path = [WBNB_ADDRESS, BOOM_ADDRESS]
    deadline = int(time.time()) + 60 * 10  # 10 minutes from now

    # Build the transaction to swap WBNB for BOOM
    transaction = router_contract.functions.swapExactETHForTokens(
        amount_out_min,
        path,
        address,
        deadline
    ).buildTransaction({
        'from': address,
        'value': w3.toWei(amount_in_wbnb, 'ether'),  # Amount of WBNB to send
        'gas': 2000000,
        'gasPrice': w3.toWei('5', 'gwei'),
        'nonce': w3.eth.getTransactionCount(address)
    })

    # Sign the transaction with your private key
    signed_transaction = w3.eth.account.signTransaction(transaction, private_key)

    # Send the transaction to the network
    tx_hash = w3.eth.sendRawTransaction(signed_transaction.rawTransaction)
    print(f"Transaction sent: {w3.toHex(tx_hash)}")

    # Wait for transaction receipt
    receipt = w3.eth.waitForTransactionReceipt(tx_hash)
    if receipt['status'] == 1:
        print(f"Successfully bought BOOM tokens: {amount_in_wbnb} WBNB")
    else:
        print("Transaction failed")

# Function to add liquidity to the BOOM/WBNB pair
def add_liquidity(amount_in_bnb, amount_in_boom):
    # You will need the contract for adding liquidity here
    pass  # Placeholder function, you can extend this to interact with the PancakeSwap Router

# Main script to interact with the liquidity management
if __name__ == "__main__":
    # Example: Buy 0.1 WBNB worth of BOOM token
    buy_boom(0.1)

    # Add liquidity (Example: Add 0.5 WBNB and corresponding BOOM tokens)
    # add_liquidity(0.5, 1000)  # This is just an example; replace with actual values