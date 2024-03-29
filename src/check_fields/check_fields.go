// check all field of docs in collections: btcBlock, btcTx, btcIn, btcOut, btcParentBlock
// btcAddress was checked during checking _key fields
package check_fields

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"os"
	"sync"
	driver "github.com/arangodb/go-driver"
	ar "arango"
	btc "bitcoin_rpc"
)

var file *os.File

func Check(db driver.Database) {
	var err error
	/* connect to bitcoin-core */
	bcApi := btc.BitcoinConfig{ Host: "localhost", Port: 10001, User: "btcuser", Password: "1234", UseSSL: false}
	bc := bcApi.ConnectBitcoin()
		
	/* make file for nodes and edges with incorrect field values */
	file, err = os.Create("../txt/incorrect_fields.txt")
    	if err != nil {
        	fmt.Println("Unable to create file:", err) 
        	os.Exit(1) 
    	}
    	defer file.Close() 
			
	/* for saving _key fields of docs for ImportDocuments method */
	arr_block := make([]ar.BitcoinBlockNode, 0, 100)	
	arr_tx := make([]ar.BitcoinTxNode, 0, 1000)	
	arr_in := make([]ar.BitcoinOutputEdge, 0, 1000)	
	arr_out := make([]ar.BitcoinOutputEdge, 0, 1000)	
	arr_next := make([]ar.BitcoinNextEdge, 0, 1000)	
	arr_parent := make([]ar.BitcoinParentBlockEdge, 0, 1000)		
	
	/* make chans for each of collection */
	/* later we`ll call goroutines for each collection*/
	blocks := make(chan []ar.BitcoinBlockNode, 1)			
	txs := make(chan []ar.BitcoinTxNode, 1)
	in := make(chan []ar.BitcoinOutputEdge, 1)
	out := make(chan []ar.BitcoinOutputEdge, 1)
	next := make(chan []ar.BitcoinNextEdge, 1)
	parents := make(chan []ar.BitcoinParentBlockEdge, 1)
	
	var start, end uint64
    	fmt.Println("Enter the starting block index: ")
    	fmt.Scanf("%d", &start)
    	fmt.Println("Enter the ending block index: ")
    	fmt.Scanf("%d", &end)
    	
    	var wg1 sync.WaitGroup
    	wg1.Add(int(end-start+1))
    	
	var n uint64
	for n = start; n <= end ; n ++ {
		
		hash := bcApi.GetBlockHash(n, bc)
		log.Printf("block %d has blockHash: %s\n", n, hash)
		block := bcApi.GetBlock(hash, bc)
		str := strconv.FormatInt(int64(block.Height), 10)
		//log.Printf("fileds for btcBlock: height: %d, key: %s, hash: %s\n", block.Height, str, hash)
		arr_block = append(arr_block, ar.BitcoinBlockNode{ BlockHeight: block.Height, Key: str, BlockHash: hash, })
		blocks <- arr_block
		arr_block = arr_block[:0]
		go CheckFieldsofBlockNode(db, "btcBlock", blocks, file, &wg1)		
		/* get all txid from msg_block - block.Tx */
		/* for each txid get the raw transaction */
		for _, t := range block.Tx {
			msg_tx := bcApi.GetRawTransaction(t, true, bc)
			log.Printf("fileds for btcTx: key: %s\ntime: %d\n", msg_tx.Txid, msg_tx.Time)
			arr_tx = append(arr_tx, ar.BitcoinTxNode{ Key: msg_tx.Txid, Time: msg_tx.Time})
			arr_tx = arr_tx[:0]
			wg1.Add(1)
			go CheckFieldsofTxNode(db, "btcTx", txs, file, &wg1)
			parentBlockKey := str + "_" + msg_tx.Txid
			//log.Printf("_key in btcParentBlock: %s\n", parentBlockKey)
			arr_parent = append(arr_parent, ar.BitcoinParentBlockEdge{ Key: parentBlockKey,
										     From: "btcTx/" + msg_tx.Txid,
										     To: "btcBlock/" + strconv.Itoa(int(n)), 
										   })
			parents <- arr_parent
			arr_parent = arr_parent[:0]
			wg1.Add(1)
			go CheckFieldsofParentEdge(db, "btcParentBlock", parents, file, &wg1)				   
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
				
				/* searching for fields spentBtc(val) and time*/
				var time int64
				var val float64 
				time = msg_tx.Time
				for _, v := range msg_tx.Vout {
					if v.N == vout {
						val = v.Value
					}
				} 
				if edgesKey != "" {
					arr_in = append(arr_in, ar.BitcoinOutputEdge{ 
											Key: edgesKey,
											From: "btcAddress/t",	
											To: "btcTx/t",		
											OutIndex: vout,
											SpentBtc: uint64(val * math.Pow10(8)),
											Time: time, })
					in <- arr_in
					arr_in = arr_in[:0]
					wg1.Add(1)
					go CheckFieldsofInOutEdge(db, "btcIn", in, file, &wg1)
					arr_next = append(arr_next, ar.BitcoinNextEdge{ 
											Key: edgesKey,
											From: "btcTx/t",	
											To: "btcTx/t",		
											Address: "",		
											OutIndex: vout,
											SpentBtc: uint64(val * math.Pow10(8)), })	
					next <- arr_next
					arr_next = arr_next[:0]
					wg1.Add(1)
					go CheckFieldsofNextEdge(db, "btcNext", next, file, &wg1)
				}
				if edgeOutKey != "" {
					arr_out = append(arr_out, ar.BitcoinOutputEdge{ 
											Key: edgeOutKey,
											From: "btcTx/" + msg_tx.Txid,
											To: "btcAddress/t",	
											OutIndex: vout,
											SpentBtc: uint64(val * math.Pow10(8)),
											Time: time, })
					out <- arr_out
					arr_out = arr_out[:0]
					wg1.Add(1)
					go CheckFieldsofInOutEdge(db, "btcOut", out, file, &wg1)
				}
			}
		}
	}
	log.Println("end of checking fields")
}

func CheckFieldsofInOutEdge(db driver.Database, coll string, ch <-chan []ar.BitcoinOutputEdge, file *os.File, wg *sync.WaitGroup) {
	defer wg.Done()
	arr := <-ch
	col, err := db.Collection(nil, coll)
	if err != nil {
		log.Fatalf("Failed openning the collection: %v", err)
	}
	var doc ar.BitcoinOutputEdge 
	for _, a := range arr{
		_, err := col.ReadDocument(nil, a.Key, &doc)
		if err != nil {
    			log.Fatalf("Failed reading doc: %v", err)
		}
		fmt.Printf("doc: %#v\n", doc)	
		var f, t bool
		if coll == "btcIn" {
			f, t = strings.Contains(doc.From, "btcAddress"), strings.Contains(doc.To, "btcTx")
		}
		if coll == "btcOut" {
			f, t = strings.Contains(doc.From, "btcTx"), strings.Contains(doc.To, "btcAddress")
		}
		ind, spent, time := doc.OutIndex == a.OutIndex, doc.SpentBtc == a.SpentBtc, doc.Time == a.Time
		if !(f && t && ind && spent && time) {
			file.WriteString(coll + ",  _key: " + a.Key + "\n")
		}
	}
}

func CheckFieldsofNextEdge(db driver.Database, coll string, ch <-chan []ar.BitcoinNextEdge, file *os.File, wg *sync.WaitGroup) {
	defer wg.Done()
	arr := <-ch
	col, err := db.Collection(nil, coll)
	if err != nil {
		log.Fatalf("Failed openning the collection: %v", err)
	}
	var doc ar.BitcoinNextEdge 
	for _, a := range arr{
		_, err := col.ReadDocument(nil, a.Key, &doc)
		if err != nil {
    			log.Fatalf("Failed reading doc: %v", err)
		}
		fmt.Printf("doc: %#v\n", doc)	
		f, t, ad := strings.Contains(doc.From, "btcTx"), strings.Contains(doc.To, "btcTx"), strings.Contains(doc.Address, "")
		ind, spent := doc.OutIndex == a.OutIndex, doc.SpentBtc == a.SpentBtc
		if !(f && t && ad && ind && spent) {
			file.WriteString(coll + ", _key: " + a.Key + "\n")
		}
	}
}

func CheckFieldsofParentEdge(db driver.Database, coll string, ch <-chan []ar.BitcoinParentBlockEdge, file *os.File, wg *sync.WaitGroup) {
	defer wg.Done()
	arr := <-ch
	col, err := db.Collection(nil, coll)
	if err != nil {
		log.Fatalf("Failed openning the collection: %v", err)
	}
	var doc ar.BitcoinParentBlockEdge 
	for _, a := range arr{
		_, err := col.ReadDocument(nil, a.Key, &doc)
		if err != nil {
    			log.Fatalf("Failed reading doc: %v", err)
		}
		fmt.Printf("doc: %#v\n", doc)	
		f, t := doc.From == a.From, doc.To == a.To
		if !(f && t) {
			file.WriteString(coll + ", _key: " + a.Key + "\n")
		}
	}	
}

func CheckFieldsofTxNode(db driver.Database, coll string, ch <-chan []ar.BitcoinTxNode, file *os.File, wg *sync.WaitGroup) {
	defer wg.Done()
	arr := <-ch
	col, err := db.Collection(nil, coll)
	if err != nil {
		log.Fatalf("Failed openning the collection: %v", err)
	}
	var doc ar.BitcoinTxNode 
	for _, a := range arr{
		_, err := col.ReadDocument(nil, a.Key, &doc)
		if err != nil {
    			log.Fatalf("Failed reading doc: %v", err)
		}
		fmt.Printf("doc: %#v\n", doc)	
		t := doc.Time == a.Time
		if !t {
			file.WriteString(coll + ", _key: " + a.Key + "\n")
		}
	}
}

func CheckFieldsofBlockNode(db driver.Database, coll string, ch <-chan []ar.BitcoinBlockNode, file *os.File, wg *sync.WaitGroup) {
	defer wg.Done()
	arr := <-ch
	col, err := db.Collection(nil, coll)
	if err != nil {
		log.Fatalf("Failed openning the collection: %v", err)
	}
	var doc ar.BitcoinBlockNode 
	for _, a := range arr{
		_, err := col.ReadDocument(nil, a.Key, &doc)
		if err != nil {
    			log.Fatalf("Failed reading doc: %v", err)
		}
		fmt.Printf("doc: %#v\n", doc)	
		hash, h := doc.BlockHash == a.BlockHash, doc.BlockHeight == a.BlockHeight
		if !(hash && h) {
			file.WriteString(coll + ", _key: " + a.Key + "\n")
		}
	}	
}

