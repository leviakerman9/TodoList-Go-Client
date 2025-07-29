package main

import (
	"context"
	"database/sql"
	"os"

	// "encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"

	"todolist-go-client/todolistv2" // <-- use your correct module path

	_ "github.com/go-sql-driver/mysql"
)

const (
	contractAddress = "0xBAceCa0E44D15CC275D4c30b6BbF258828F9D0D4"               // Replace with your proxy contract address
)

func main() {

	godotenv.Load()
	   alchemyKey := os.Getenv("ALCHEMY_KEY")
	   mysqlDSN := os.Getenv("DATABASE")

	// Connect to Ethereum via Alchemy
	client, err := ethclient.Dial("wss://eth-sepolia.g.alchemy.com/v2/"+ alchemyKey)
	if err != nil {
		log.Fatalf("Failed to connect to Alchemy WebSocket: %v", err)
	}
	defer client.Close()

	// Load contract ABI
contractAbi, err := abi.JSON(strings.NewReader(todolistv2.Todolistv2ABI))
	if err != nil {
		log.Fatalf("Failed to parse ABI: %v", err)
	}

	// Connect to MySQL
	db, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	defer db.Close()

	// Filter logs from the deployed proxy contract
	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			common.HexToAddress(contractAddress),
		},
	}

	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		log.Fatalf("Failed to subscribe to logs: %v", err)
	}

	fmt.Println("🔁 Listening for TaskCreated events...")

	// Event Signature
	eventSignature := []byte("TaskCreated(uint256,string,uint8,address)")
	eventHash := crypto.Keccak256Hash(eventSignature)

	for {
		select {
		case err := <-sub.Err():
			log.Printf("Subscription error: %v", err)

		case vLog := <-logs:
			if vLog.Topics[0] == eventHash {
				var event struct {
					Id       *big.Int
					Content  string
					Status   uint8
					Assignee common.Address
				}

				err := contractAbi.UnpackIntoInterface(&event, "TaskCreated", vLog.Data)
				if err != nil {
					log.Printf("Failed to unpack log: %v", err)
					continue
				}

				fmt.Printf("✅ TaskCreated: ID=%v, Content='%s', Status=%v, Assignee=%s\n",
					event.Id, event.Content, event.Status, event.Assignee.Hex())

				_, err = db.Exec(`
					INSERT INTO tasks (id, content, status, assignee) VALUES (?, ?, ?, ?)`,
					event.Id.Int64(), event.Content, event.Status, event.Assignee.Hex())
				if err != nil {
					log.Printf("MySQL insert failed: %v", err)
				}
			}
		}
	}
}