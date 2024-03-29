package arango

import (
	"fmt"
	"os"
	"flag"
	"log"
	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
)

type BitcoinTxNode struct {
	Key	string	`json:"_key"`
	Time 	int64	`json:"time"`
}

type Node struct {
	Key	string	`json:"_key"`
}

type Edge struct {
	Key		string	`json:"_key"`
	From		string	`json:"_from"`	
	To		string	`json:"_to"`
}

type BitcoinBlockNode struct {
	BlockHeight	uint64  `json:"blockHeight"` 
	Key		string	`json:"_key"`		
	BlockHash	string	`json:"blockHash"`
}

type BitcoinParentBlockEdge struct {
	Key		string	`json:"_key"`	
	From		string	`json:"_from"`	
	To		string	`json:"_to"`	
}

type BitcoinOutputEdge struct {
	Key		string	`json:"_key"`	// arango _key is generated by arango txId + '_' + outIndex
	From		string	`json:"_from"`	// arango id of transaction 'btcTx/{_key}'
	To		string	`json:"_to"`	// arango id of block 'btcAddress/{_key}'
	OutIndex	int	`json:"outIndex`
	SpentBtc	uint64	`json:"spentBtc"`
	Time 		int64	`json:"time"`
}

type BitcoinNextEdge struct {
	Key		string	`json:"_key"`	// arango _key is generated by arango  txId + '_' + outIndex
	From		string	`json:"_from"`	// arango id of transaction 'btcTx/{_key}'
	To		string	`json:"_to"`	// arango id of block 'btcTx/{_key}'
	Address	string	`json:"address"`
	OutIndex	int 	`json:"outIndex`
	SpentBtc	uint64	`json:"spentBtc"`
}	

//BitcoinInEdge has the same structure as BitcoinOutputEdge but _from :'btcAddress/{_key}', _to: 'btcTx/{_key}'

type ArangoConfig struct {
	Host 		string
	Port 		string
	User 		string
	Password	string
} 

/* connect to arangodb server using http */
/* open ArangoDB database with entered name name */
func (api *ArangoConfig) Connect() driver.Database {
	var err error
	var client driver.Client
	var conn driver.Connection
	
	flag.Parse()

	conn, err = http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{"http://" + api.Host + ":" + api.Port},
	})
	if err != nil {
		log.Fatalf("Failed to create HTTP connection: %v", err)
	}
	client, err = driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: driver.BasicAuthentication(api.User, api.Password),
	})

	fmt.Println("Enter database name: ")
	var db_name string
	var db driver.Database
	fmt.Scanf("%s", &db_name)
	db_exists, err := client.DatabaseExists(nil, db_name)
	
	if db_exists {
		db, err = client.Database(nil, db_name)
		if err != nil {
			log.Fatalf("Failed to open existing database: %v", err)
		} 
		fmt.Println(db.Name() + " exists")
	} else {
		fmt.Println("Sorry, database with this name doesn`t exist")
		fmt.Println("Do you want to create database with this name? [y/n]: ")
		var ans string
		fmt.Scanf("%s", &ans) 
		if ans == "y" {
			db, err = client.CreateDatabase(nil, db_name, nil)
			if err != nil {
				log.Fatalf("Failed to create database: %v", err)
			}
			fmt.Println("the database is successfully created. Rerun the program")
			os.Exit(1)
		}
		if ans == "n" {
			fmt.Println("Rerun the program and enter the correct database name ")
			os.Exit(1)
		}
	}
	return db 
}
	
