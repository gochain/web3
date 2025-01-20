const { ethers } = require("ethers");

// Connect to the xDai network
const provider = new ethers.providers.JsonRpcProvider("https://rpc.gnosischain.com");
const wallet = new ethers.Wallet(process.env.PRIVATE_KEY, provider);

// Token contract address and ABI
const tokenAddress = "0xcd6A51559254030cA30C2FB2cbdf5c492e8Caf9c";
const recipient = "0x97293CeAB815896883e8200AEf5a4581a70504b2";
const tokenAbi = [
  "function transfer(address recipient, uint256 amount) public returns (bool)",
];

// Create token contract instance
const tokenContract = new ethers.Contract(tokenAddress, tokenAbi, wallet);

// Transfer tokens
async function sendTokens() {
  const amount = ethers.utils.parseUnits("10.0", 18); // Adjust token amount and decimals
  const tx = await tokenContract.transfer(recipient, amount);
  console.log(`Transaction hash: ${tx.hash}`);
  await tx.wait();
  console.log(`Transferred 10 tokens to ${recipient}`);
}

sendTokens();
