package main

import (
	"context"
	"database/sql"
	"os"
	"sync"

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

type TaskCreated struct {
	Id       *big.Int
	Content  string
	Status   uint8
	Assignee common.Address
}

const contractAddress = "0xBAceCa0E44D15CC275D4c30b6BbF258828F9D0D4" // Replace with your proxy contract address

// func to connect to ethereum node using alchemy key
func connect(alchemyKey string) *ethclient.Client {

	client, err := ethclient.Dial("wss://eth-sepolia.g.alchemy.com/v2/" + alchemyKey)
	if err != nil {
		log.Fatalf("Failed to connect to Alchemy WebSocket: %v", err)
	}

	return client

}

// func to load contract abi it is converting abi  jason to string
func loadContractAbi() abi.ABI {
	contractAbi, err := abi.JSON(strings.NewReader(todolistv2.Todolistv2ABI))
	if err != nil {
		log.Fatalf("Failed to parse ABI: %v", err)
	}
	return contractAbi
}

// func to connect to sql
func connectToMysql(mysqlDSN string) *sql.DB {
	db, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	return db
}

// func to log events type and hanlde subscription
func processEvents(
	wg *sync.WaitGroup,
	query ethereum.FilterQuery,
	client *ethclient.Client,
	contractAbi abi.ABI,
	db *sql.DB,
	taskCreatedHash common.Hash,
) {

	defer wg.Done()
	logs := make(chan types.Log)

	sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		log.Fatalf("Initial subscription failed: %v", err)
	}

	defer sub.Unsubscribe()

	for {
		select {
		case err := <-sub.Err():
			log.Printf("Subscription error: %v", err)

			// firse retry kro ek baar
			log.Printf("again retrying to setup subscription")
			 sub,err = client.SubscribeFilterLogs(context.Background(), query, logs)
			if err != nil {
				log.Fatalf("Initial subscription failed: %v", err)
			}

		case vLog := <-logs:
			switch vLog.Topics[0] {
			case taskCreatedHash:
				handleTaskCreated(vLog, contractAbi, db)

			default:
				log.Println("Unknown event signature")
			}
		}
	}
}

// func to handle event and insert it into the db

func handleTaskCreated(vLog types.Log, contractAbi abi.ABI, db *sql.DB) {
	var event TaskCreated
	err := contractAbi.UnpackIntoInterface(&event, "TaskCreated", vLog.Data)
	if err != nil {
		log.Printf("Failed to unpack TaskCreated: %v", err)
		return
	}

	fmt.Printf("✅ TaskCreated: ID=%v, Content='%s', Status=%v, Assignee=%s\n",
		event.Id, event.Content, event.Status, event.Assignee.Hex())

	_, err = db.Exec(`INSERT INTO tasks (id, content, status, assignee) VALUES (?, ?, ?, ?)`,
		event.Id.Int64(), event.Content, event.Status, event.Assignee.Hex())

	if err != nil {
		log.Printf("MySQL insert failed: %v", err)
	}
}

func main() {
	var wg sync.WaitGroup
	godotenv.Load()
	alchemyKey := os.Getenv("ALCHEMY_KEY")
	mysqlDSN := os.Getenv("DATABASE")

	// Connect to Ethereum via Alchemy
	client := connect(alchemyKey)

	// Load contract ABI
	contractAbi := loadContractAbi()

	// Connect to MySQL
	db := connectToMysql(mysqlDSN)

	// Event Signature
	taskCreatedSig := []byte("TaskCreated(uint256,string,uint8,address)")

	// event hash
	taskCreatedHash := crypto.Keccak256Hash(taskCreatedSig)

	// Filter logs from the deployed proxy contract
	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			common.HexToAddress(contractAddress),
		},
		Topics: [][]common.Hash{
			{taskCreatedHash},
		},
	}

	fmt.Println("🔁 Listening for TaskCreated events...")

	wg.Add(1)

	processEvents(&wg,query, client, contractAbi, db, taskCreatedHash)

	wg.Wait()

}