package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"os"
	"encoding/hex"
	"encoding/json"
	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/Toorop/go-bitcoind"
)

type Node struct {
	Key	string	`json:"_key"`
}

type Edge struct {
	Key		string	`json:"_key"`
	From		string	`json:"_from"`	
	To		string	`json:"_to"`
}

const (
	SERVER_HOST        = "localhost"
	SERVER_PORT        = 10001
	USER               = "btcuser"
	PASSWD             = "1234"
	USESSL             = false
)

var keys *os.File
var db driver.Database

func main() {

	/* connect to bitcoin-core */
	bc, err := bitcoind.New(SERVER_HOST, SERVER_PORT, USER, PASSWD, USESSL)
	if err != nil {
		log.Fatalln(err)
	}
	
	/* connect to arangodb server using http */
	var client driver.Client
	var conn driver.Connection
	
	flag.Parse()

	conn, err = http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{"http://localhost:8529"},
	})
	if err != nil {
		log.Fatalf("Failed to create HTTP connection: %v", err)
	}
	client, err = driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: driver.BasicAuthentication("root", ""),
	})
	
	/* open ArangoDB database with entered name name */
	fmt.Println("Enter database name: ")
	var db_name string
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
	
	/* make file for nodes and edges didn`t exist in db before importDocument method */
	keys, err = os.Create("keys_of_imported_docs.txt")
    	if err != nil{
        	fmt.Println("Unable to create file:", err) 
        	os.Exit(1) 
    	}
    	defer keys.Close() 
	
	/* getblockcount; can use count variable insted of n variable in for cycle for blocks */
	/*count, err := bc.GetBlockCount()
	if err != nil {
		log.Fatalf("Failed to get blockCount: %v", err)
	}
	log.Printf("Block count: %d", count) */
	
	/* make chans for each of collection */
	/* later we`ll call goroutines for each collection*/
	blocks := make(chan []Node, 100)			
	txs := make(chan []Node, 1000)
	addrs := make(chan []Node, 2500)
	in := make(chan []Edge, 1000)
	out := make(chan []Edge, 1000)
	next := make(chan []Edge, 1000)
	parents := make(chan []Edge, 1000)
		
	/* for saving _key fields of docs for ImportDocuments method */
	arr_block := make([]Node, 0, 100)	
	arr_tx := make([]Node, 0, 1000)	
	arr_addr := make([]Node, 0, 2500)	
	arr_in := make([]Edge, 0, 1000)	
	arr_out := make([]Edge, 0, 1000)	
	arr_next := make([]Edge, 0, 1000)	
	arr_parent := make([]Edge, 0, 1000)		
	
	/* map for types of transactions type->seen/not (true, false)*/
	typesTx := map[string]bool { 		// has address field?
		"pubkeyhash": false,		// yes
		"nonstadard": false,		// no 
		"multisig": false,		// no 
		"pubkey": false,		// no 
		"scripthash" : false,		// yes
		"nulldata": false,		// no 
		"witness_v0_keyhash" : false,	// yes
		"witness_v0_scripthash": false,// yes
		"witness_unknown": false,	// no
	}
	
	/* counters for each type */
	pbkh := 0
	nstd := 0
	mltsg := 0
	pbk := 0
	srpth := 0
	nulld := 0
	w_kh := 0
	w_srpth := 0
	w_un := 0
	
	/* file for transactions with special types in vout of tx and their tx_hash */
	file, err := os.Create("transactions.txt")
    	if err != nil{
        	fmt.Println("Unable to create file:", err) 
        	os.Exit(1) 
    	}
    	defer file.Close() 
    	
    	var n, start, end uint64
    	fmt.Println("Enter the starting block index: ")
    	fmt.Scanf("%d", &start)
    	fmt.Println("Enter the ending block index: ")
    	fmt.Scanf("%d", &end)
    	
	for n = start; n <= end; n ++ {
		/* get blockhash */
		hash, err := bc.GetBlockHash(n)
		if err != nil {
			log.Fatalf("Failed to get blockHash: %v", err)
		}
		log.Printf("block %d has blockHash: %s\n", n, hash)
	
		/* get block */
		block, err := bc.GetBlock(hash)
		if err != nil {
			log.Fatalf("Failed to get blockBlock: %v", err)
		}
		log.Printf("_key in btcBlock: %d\n", block.Height)
		str := strconv.FormatInt(int64(block.Height), 10)
		arr_block = append(arr_block, Node{ Key: str, })
		blocks <- arr_block
		go ImportNodes(db, "btcBlock", blocks, keys)
		/* get all txid from msg_block */
		block_tx := block.Tx
		//for i, t := range block_tx {
		//	log.Printf("tx %d: %s", i, t)
		//}
		//log.Printf("tx`s of btcBlock: %s", msg_block.Tx)
	
		/* for each txid get the raw transaction */
		for _, t := range block_tx {
			msg_tx, err := bc.GetRawTransactionUPD(t, true)	// my method GetRawTransactionUPD I added to package
			if err != nil {
				log.Fatalf("Failed to get rawTransaction: %v", err)
			}
			log.Printf("_key in btcTx: %s\n", msg_tx.Txid)
			arr_tx = append(arr_tx, Node{ Key: msg_tx.Txid, })
			txs <- arr_tx
			go ImportNodes(db, "btcTx", txs, keys)
			parentBlockKey := str + "_" + msg_tx.Txid
			log.Printf("_key in btcParentBlock: %s\n", parentBlockKey)
			arr_parent = append(arr_parent, Edge{ Key: parentBlockKey, From: "t/t", To: "t/t", })
			parents <- arr_parent
			go ImportEdges(db, "btcParentBlock", parents, keys)
			for _, vin := range msg_tx.Vin {
				txid := vin.Txid
				//log.Printf("txid field: %s", txid)
				vout := vin.Vout //int
				voutstr := strconv.Itoa(vout)
				//log.Printf("vout field: %s", voutstr)
				var edgesKey, edgeOutKey string
				if txid == "" && voutstr == "" {
					edgesKey = ""
					edgeOutKey = ""
				} else if txid == "" && voutstr != "" {
					edgesKey = ""
					edgeOutKey = msg_tx.Txid + "_" + voutstr
				} else {
					edgesKey = txid + "_" + voutstr
					edgeOutKey = txid + "_" + voutstr
				}
				log.Printf("_key in btcIn: %s\n", edgesKey)
				log.Printf("_key in btcOut: %s\n", edgeOutKey)
				log.Printf("_key in btcNext: %s\n", edgesKey)
				if edgesKey != "" {
					arr_in = append(arr_in, Edge{ Key: edgesKey, From: "t/t", To: "t/t", })
					in <- arr_in
					go ImportEdges(db, "btcIn", in, keys)
					arr_next = append(arr_next, Edge{ Key: edgesKey, From: "t/t", To: "t/t", })
					next <- arr_next
					go ImportEdges(db, "btcNext", next, keys)
				}
				if edgeOutKey != "" {
					arr_out = append(arr_out, Edge{ Key: edgeOutKey, From: "t/t", To: "t/t", })
					out <- arr_out
					go ImportEdges(db, "btcOut", out, keys )
				}
				
			}
			for _, vout := range msg_tx.Vout {
				address := vout.ScriptPubKey.Address
				index_of_tx_output := strconv.Itoa(vout.N)
				type_vout := vout.ScriptPubKey.Type
				//log.Printf("index of tx output: %s", index_of_tx_output)	
				var addrKey string
				if address == ""{
					addrKey = msg_tx.Txid + "_" + index_of_tx_output 
					if type_vout == "nonstadard" && typesTx["nonstadard"] == false {
						typesTx["nonstadard"] = true
						nstd++
						file.WriteString("nonstandard: " + msg_tx.Txid + "\n")
						if nstd < 10  {		
							typesTx["nonstadard"] = false
						}
					}
					if type_vout == "multisig" && typesTx["multisig"] == false {
						typesTx["multisig"] = true
						mltsg++
						file.WriteString("multisig: " + msg_tx.Txid + "\n")
						if mltsg < 10  {	
							typesTx["multisig"] = false
						}
					}
					if type_vout == "pubkey" && typesTx["pubkey"] == false {
						typesTx["pubkey"] = true
						pbk++
						file.WriteString("pubkey: " + msg_tx.Txid + "\n")
						if pbk < 10  {	
							typesTx["pubkey"] = false
						}
					}
					if type_vout == "nulldata" && typesTx["nulldata"] == false {
						typesTx["nulldata"] = true
						nulld++
						file.WriteString("nulldata: " + msg_tx.Txid + "\n")
						if nulld < 10  {	
							typesTx["nulldata"] = false
						}
					}
					if type_vout == "witness_unknown" && typesTx["witness_unknown"] == false {
						typesTx["witness_unknown"] = true
						w_un++
						file.WriteString("witness_unknown: " + msg_tx.Txid + "\n")
						if w_un< 10  {	
							typesTx["witness_unknown"] = false
						}
					}
				} else {
					addrKey = address
					if type_vout == "pubkeyhash" && typesTx["pubkeyhash"] == false {
						typesTx["pubkeyhash"] = true
						pbkh++
						file.WriteString("pubkeyhash: " + msg_tx.Txid + "\n")
						if pbkh < 10  {		
							typesTx["pubkeyhash"] = false
						}
					}
					if type_vout == "scripthash" && typesTx["scripthash"] == false {
						typesTx["scripthash"] = true
						srpth++
						file.WriteString("scripthash: " + msg_tx.Txid + "\n")
						if srpth < 10  {		
							typesTx["scripthash"] = false
						}
					}
					if type_vout == "witness_v0_keyhash" && typesTx["witness_v0_keyhash"] == false {
						typesTx["witness_v0_keyhash"] = true
						w_kh++
						file.WriteString("witness_v0_keyhash: " + msg_tx.Txid + "\n")
						if w_kh < 10  {		
							typesTx["witness_v0_keyhash"] = false
						}
					}
					if type_vout == "witness_v0_scripthash" && typesTx["witness_v0_scripthash"] == false {
						typesTx["witness_v0_scripthash"] = true
						w_srpth++
						file.WriteString("witness_v0_scripthash: " + msg_tx.Txid + "\n")
						if w_srpth < 10  {		
							typesTx["witness_v0_scripthash"] = false
						}
					}
				}	
				arr_addr = append(arr_addr, Node{ Key: addrKey, })
				addrs <- arr_addr
				go ImportNodes(db, "btcAddress", addrs, keys)
				log.Printf("_key in btcAddress: %s\n", addrKey) //here
			}
		}
	}
	log.Println("end of process")
}


/* ====================== helpful functions ====================== */
func formatRawResponse(raw []byte) string {
	l := len(raw)
	if l < 2 {
		return hex.EncodeToString(raw)
	}
	if (raw[0] == '{' && raw[l-1] == '}') || (raw[0] == '[' && raw[l-1] == ']') {
		return string(raw)
	}
	return hex.EncodeToString(raw)
}

func describe(err error) string {
	if err == nil {
		return "nil"
	}
	cause := driver.Cause(err)
	var msg string
	if re, ok := cause.(*driver.ResponseError); ok {
		msg = re.Error()
	} else {
		c, _ := json.Marshal(cause)
		msg = string(c)
	}
	if cause.Error() != err.Error() {
		return fmt.Sprintf("%v caused by %v (%v)", err, cause, msg)
	}
	return fmt.Sprintf("%v (%v)", err, msg)
}

func ImportNodes(db driver.Database, coll string, ch chan []Node, keys *os.File) {
	//get docs from chan and open collection with "coll" name
	arr := <-ch
	log.Println("got nodes from channel " + "coll_name is " + coll)
	col, err := db.Collection(nil, coll)
	if err != nil {
		log.Fatalf("Failed openning the collection: %v", err)
	}
	var raw []byte
	var details []string
	ctx := driver.WithImportDetails(driver.WithRawResponse(nil, &raw), &details)
	var options driver.ImportDocumentOptions
	options = driver.ImportDocumentOptions{ Overwrite: false, OnDuplicate: "ImportOnDuplicateError", Complete: false,}
	stats, err := col.ImportDocuments(ctx, arr, &options)
	if err != nil {
		log.Fatalf("Failed to import documents: %s %#v", describe(err), err)
	} else {
		if stats.Created != int64(len(arr)) {
			log.Printf("Expected %d created documents, got %d (json %s)", len(arr), stats.Created, formatRawResponse(raw))
			// field Created holds the number of documents imported.
			// we expect that method ImportDocuments will import all docs from array. 
			//But this method will not import the current document because of the unique key constraint violation.

			details_str := strings.Join(details, " ")
			for _, a := range arr {
				if !strings.Contains(details_str, a.Key) {
					keys.WriteString("collection: " + coll + ", _key: " + a.Key + "\n")
				}		
			}	
		}
	}	
}

func ImportEdges(db driver.Database, coll string, ch chan []Edge, keys *os.File) {
	//get docs from chan and open collection with "coll" name
	arr := <-ch
	log.Println("got edges from channel " + "coll_name is " + coll)
	col, err := db.Collection(nil, coll)
	if err != nil {
		log.Fatalf("Failed openning the collection: %v", err)
	}
	var raw []byte
	var details []string
	ctx := driver.WithImportDetails(driver.WithRawResponse(nil, &raw), &details)
	var options driver.ImportDocumentOptions
	options = driver.ImportDocumentOptions{ Overwrite: false, OnDuplicate: "ImportOnDuplicateError", Complete: false,}
	stats, err := col.ImportDocuments(ctx, arr, &options)
	if err != nil {
		log.Fatalf("Failed to import documents: %s %#v", describe(err), err)
	} else {
		if stats.Created != int64(len(arr)) {
			log.Printf("Expected %d created documents, got %d (json %s)", len(arr), stats.Created, formatRawResponse(raw))
			// field Created holds the number of documents imported.
			// we expect that method ImportDocuments will import all docs from array. 
			//But this method will not import the current document because of the unique key constraint violation.

			details_str := strings.Join(details, " ")
			for _, a := range arr {
				if !strings.Contains(details_str, a.Key) {
					keys.WriteString("collection: " + coll + ", _key: " + a.Key + "\n")
				}		
			}
			
		}
	}	
}

