## This project connects to an Ethereum smartcontract deployed on the Sepolia testnet, listens for `TaskCreated` and `TaskCompleted` events using Go, and stores them in a MySQL database. It uses Alchemy for WebSocket RPC and `.env` for environment variable management.

---


### 🔧 Setup Instructions

- First download GETH (go-ethereum) from https://geth.ethereum.org/downloads

```
cd todolist-go-client
```
### 🚀 Running the Project

- Initialize Go Modules

```
go mod init todolist-go-client
go mod tidy
```

### 🚀 Install dependencies
```
go get github.com/ethereum/go-ethereum
go get github.com/joho/godotenv
go get -u github.com/go-sql-driver/mysql
```

### 🚀 Generate Go bindings from ABI
```
abigen --abi ./Todolist.json --pkg todolistv2 --out ./todolistv2.go
```

### 🚀 Run the Go project
```
go run .\main.go    
```

### 🚀 Expected output
```
🔁 Listening for TaskCreated events...
✅ TaskCreated: ID=5, Content='Task 2', Status=0, Assignee=0x...
```

### 📡 Triggering Events (from Hardhat Project)
- In your separate Hardhat project, run:

```
npx hardhat run scripts/interact.js --network sepolia
```




