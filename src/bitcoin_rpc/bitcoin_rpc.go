package bitcoin_rpc

import (
	"log"
	"github.com/Toorop/go-bitcoind"
)

type BitcoinConfig struct {
	Host 		string
	Port		int
	User		string
	Password	string
	UseSSL		bool
}

/* connect to bitcoin-core */
func (api *BitcoinConfig) ConnectBitcoin() *bitcoind.Bitcoind { 
	bc, err := bitcoind.New(api.Host, api.Port, api.User, api.Password, api.UseSSL)
	if err != nil {
		log.Fatalln(err)
	}
	return bc
}

/* get blockhash */
func (api *BitcoinConfig) GetBlockHash(n uint64, bc *bitcoind.Bitcoind) string {
	hash, err := bc.GetBlockHash(n)
	if err != nil {
		log.Fatalf("Failed to get blockHash: %v", err)
	}
	return hash
}

/* get block */
func (api *BitcoinConfig) GetBlock(hash string, bc *bitcoind.Bitcoind) bitcoind.Block {
	block, err := bc.GetBlock(hash)
	if err != nil {
		log.Fatalf("Failed to get blockBlock: %v", err)
	}
	return block
}

/* get raw transaction */
func (api *BitcoinConfig) GetRawTransaction(t string, v bool, bc *bitcoind.Bitcoind) bitcoind.RawTransaction {
	msg_tx, err := bc.GetRawTransactionUPD(t, v)	
	if err != nil {
		log.Fatalf("Failed to get rawTransaction: %v", err)
	}
	return msg_tx
}

