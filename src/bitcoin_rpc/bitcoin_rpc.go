package bitcoin_rpc

import (
	"log"
	"github.com/Toorop/go-bitcoind"
)

const (
	SERVER_HOST        = "localhost"
	SERVER_PORT        = 10001
	USER               = "btcuser"
	PASSWD             = "1234"
	USESSL             = false
)


/* connect to bitcoin-core */
func ConnectBitcoin() *bitcoind.Bitcoind { 
	bc, err := bitcoind.New(SERVER_HOST, SERVER_PORT, USER, PASSWD, USESSL)
	if err != nil {
		log.Fatalln(err)
	}
	return bc
}

/* get blockhash */
func GetBlockHash(n uint64, bc *bitcoind.Bitcoind) string {
	hash, err := bc.GetBlockHash(n)
	if err != nil {
		log.Fatalf("Failed to get blockHash: %v", err)
	}
	return hash
}

/* get block */
func GetBlock(hash string, bc *bitcoind.Bitcoind) bitcoind.Block {
	block, err := bc.GetBlock(hash)
	if err != nil {
		log.Fatalf("Failed to get blockBlock: %v", err)
	}
	return block
}

/* get raw transaction */
func GetRawTransaction(t string, v bool, bc *bitcoind.Bitcoind) bitcoind.RawTransaction {
	msg_tx, err := bc.GetRawTransactionUPD(t, v)	
	if err != nil {
		log.Fatalf("Failed to get rawTransaction: %v", err)
	}
	return msg_tx
}
		