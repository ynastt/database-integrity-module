package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"os"
	"sync"
	"encoding/hex"
	"encoding/json"
	driver "github.com/arangodb/go-driver"
	check "check_fields"
	ar "arango"
	btc "bitcoin_rpc"
)

var keys *os.File
var db driver.Database

func main() {	
	var err error
	
	/* connect to bitcoin-core */
	bcApi := btc.BitcoinConfig{ Host: "localhost", Port: 10001, User: "btcuser", Password: "1234", UseSSL: false}
	bc := bcApi.ConnectBitcoin()
	
	/* connect to arangodb server using http */
	/* open ArangoDB database with entered name name */
	dbApi := ar.ArangoConfig{ Host: "localhost", Port: "8529", User: "root", Password: "",}
	db := dbApi.Connect()
		
	flag.Parse()
	
	/* make file for nodes and edges didn`t exist in db before importDocument method */
	keys, err = os.Create("../txt/keys_of_imported_docs.txt")
    	if err != nil {
        	fmt.Println("Unable to create file:", err) 
        	os.Exit(1) 
    	}
    	defer keys.Close() 
		
	/* make chans for each of collection */
	/* later we`ll call goroutines for each collection*/
	blocks := make(chan []ar.Node, 1)			
	txs := make(chan []ar.Node, 1)
	addrs := make(chan []ar.Node, 1)
	in := make(chan []ar.Edge, 1)
	out := make(chan []ar.Edge, 1)
	next := make(chan []ar.Edge, 1)
	parents := make(chan []ar.Edge, 1)
		
	/* for saving _key fields of docs for ImportDocuments method */
	arr_block := make([]ar.Node, 0, 100)	
	arr_tx := make([]ar.Node, 0, 1000)	
	arr_addr := make([]ar.Node, 0, 2500)	
	arr_in := make([]ar.Edge, 0, 1000)	
	arr_out := make([]ar.Edge, 0, 1000)	
	arr_next := make([]ar.Edge, 0, 1000)	
	arr_parent := make([]ar.Edge, 0, 1000)		
	
	/* map for types of transactions type->seen (true, false) */
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
	pbkh, nstd, mltsg, pbk, srpth, nulld, w_kh, w_srpth, w_un := 0, 0, 0, 0, 0, 0, 0, 0, 0
	
	/* file for transactions with special types in vout of tx and their tx_hash */
	tran, err := os.Create("../txt/transactions.txt")
    	if err != nil {
        	fmt.Println("Unable to create file:", err) 
        	os.Exit(1) 
    	}
    	defer tran.Close() 
    	
    	var n, start, end uint64
    	fmt.Println("Enter the starting block index: ")
    	fmt.Scanf("%d", &start)
    	fmt.Println("Enter the ending block index: ")
    	fmt.Scanf("%d", &end)
    	
    	var wg sync.WaitGroup
    	wg.Add(int(end-start+1))
    	
	for n = start; n <= end; n ++ {
		hash := bcApi.GetBlockHash(n, bc)
		//log.Printf("block %d has blockHash: %s\n", n, hash)
		block := bcApi.GetBlock(hash, bc)
		//log.Printf("_key in btcBlock: %d\n", block.Height)
		str := strconv.FormatInt(int64(block.Height), 10)
		arr_block = append(arr_block, ar.Node{ Key: str, })
		blocks <- arr_block
		arr_block = arr_block[:0]
		go ImportNodes(db, "btcBlock", blocks, keys, &wg)
		
		/* get all txid from msg_block - block.Tx */
		/* for each txid get the raw transaction */
		for _, t := range block.Tx {
			
			msg_tx := bcApi.GetRawTransaction(t, true, bc)
			//log.Printf("_key in btcTx: %s\n", msg_tx.Txid)
			arr_tx = append(arr_tx, ar.Node{ Key: msg_tx.Txid, })
			txs <- arr_tx
			arr_tx = arr_tx[:0]
			wg.Add(1)
			go ImportNodes(db, "btcTx", txs, keys, &wg)
			parentBlockKey := str + "_" + msg_tx.Txid
			//log.Printf("_key in btcParentBlock: %s\n", parentBlockKey)
			arr_parent = append(arr_parent, ar.Edge{ Key: parentBlockKey, From: "t/t", To: "t/t", })
			parents <- arr_parent
			arr_parent = arr_parent[:0]
			wg.Add(1)
			go ImportEdges(db, "btcParentBlock", parents, keys, &wg)
			
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
				//log.Printf("_key in btcIn: %s\n", edgesKey)
				//log.Printf("_key in btcOut: %s\n", edgeOutKey)
				//log.Printf("_key in btcNext: %s\n", edgesKey)
				if edgesKey != "" {
					arr_in = append(arr_in, ar.Edge{ Key: edgesKey, From: "t/t", To: "t/t", })
					in <- arr_in
					arr_in = arr_in[:0]
					wg.Add(1)
					go ImportEdges(db, "btcIn", in, keys, &wg)
					arr_next = append(arr_next, ar.Edge{ Key: edgesKey, From: "t/t", To: "t/t", })
					next <- arr_next
					arr_next= arr_next[:0]
					wg.Add(1)
					go ImportEdges(db, "btcNext", next, keys, &wg)
				}
				if edgeOutKey != "" {
					arr_out = append(arr_out, ar.Edge{ Key: edgeOutKey, From: "t/t", To: "t/t", })
					out <- arr_out
					arr_out= arr_out[:0]
					wg.Add(1)
					go ImportEdges(db, "btcOut", out, keys, &wg)
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
						tran.WriteString("nonstandard: " + msg_tx.Txid + "\n")
						if nstd < 10  {		
							typesTx["nonstadard"] = false
						}
					}
					if type_vout == "multisig" && typesTx["multisig"] == false {
						typesTx["multisig"] = true
						mltsg++
						tran.WriteString("multisig: " + msg_tx.Txid + "\n")
						if mltsg < 10  {	
							typesTx["multisig"] = false
						}
					}
					if type_vout == "pubkey" && typesTx["pubkey"] == false {
						typesTx["pubkey"] = true
						pbk++
						tran.WriteString("pubkey: " + msg_tx.Txid + "\n")
						if pbk < 10  {	
							typesTx["pubkey"] = false
						}
					}
					if type_vout == "nulldata" && typesTx["nulldata"] == false {
						typesTx["nulldata"] = true
						nulld++
						tran.WriteString("nulldata: " + msg_tx.Txid + "\n")
						if nulld < 10  {	
							typesTx["nulldata"] = false
						}
					}
					if type_vout == "witness_unknown" && typesTx["witness_unknown"] == false {
						typesTx["witness_unknown"] = true
						w_un++
						tran.WriteString("witness_unknown: " + msg_tx.Txid + "\n")
						if w_un< 10  {	
							typesTx["witness_unknown"] = false
						}
					}
					
				} else {
					addrKey = address
					if type_vout == "pubkeyhash" && typesTx["pubkeyhash"] == false {
						typesTx["pubkeyhash"] = true
						pbkh++
						tran.WriteString("pubkeyhash: " + msg_tx.Txid + "\n")
						if pbkh < 10  {		
							typesTx["pubkeyhash"] = false
						}
					}
					if type_vout == "scripthash" && typesTx["scripthash"] == false {
						typesTx["scripthash"] = true
						srpth++
						tran.WriteString("scripthash: " + msg_tx.Txid + "\n")
						if srpth < 10  {		
							typesTx["scripthash"] = false
						}
					}
					if type_vout == "witness_v0_keyhash" && typesTx["witness_v0_keyhash"] == false {
						typesTx["witness_v0_keyhash"] = true
						w_kh++
						tran.WriteString("witness_v0_keyhash: " + msg_tx.Txid + "\n")
						if w_kh < 10  {		
							typesTx["witness_v0_keyhash"] = false
						}
					}
					if type_vout == "witness_v0_scripthash" && typesTx["witness_v0_scripthash"] == false {
						typesTx["witness_v0_scripthash"] = true
						w_srpth++
						tran.WriteString("witness_v0_scripthash: " + msg_tx.Txid + "\n")
						if w_srpth < 10  {		
							typesTx["witness_v0_scripthash"] = false
						}
					}
				}	
				arr_addr = append(arr_addr, ar.Node{ Key: addrKey, })
				addrs <- arr_addr
				arr_addr= arr_addr[:0]
				wg.Add(1)
				go ImportNodes(db, "btcAddress", addrs, keys, &wg)
				//log.Printf("_key in btcAddress: %s\n", addrKey) //here
			}
		}
	}
	wg.Wait()
	close(addrs)
	close(out)
	close(in)
	close(next)
	close(txs)
	close(parents)
	close(blocks)
	/* check fields of docs in collections*/
	log.Println("\n\nstart checking fields")
	check.Check(db)
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

func ImportNodes(db driver.Database, coll string, ch <-chan []ar.Node, keys *os.File, wg *sync.WaitGroup) {
	defer wg.Done()
	arr := <-ch
	//log.Println("got nodes from channel " + "coll_name is " + coll)
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
			log.Printf("Collection: %s: Expected %d created documents, got %d (json %s)", coll, len(arr), stats.Created, 				formatRawResponse(raw))
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

func ImportEdges(db driver.Database, coll string, ch <-chan []ar.Edge, keys *os.File, wg *sync.WaitGroup) {
	defer wg.Done()
	arr := <-ch
	//log.Println("got edges from channel " + "coll_name is " + coll)
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
			log.Printf("Collection: %s: Expected %d created documents, got %d (json %s)", coll, len(arr), stats.Created, 				formatRawResponse(raw))
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

