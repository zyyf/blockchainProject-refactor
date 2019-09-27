package Data

import (
	"encoding/json"
	"log"
)

type Transaction struct {
	//FromAddr string
	//ToAddr string
	Index  int
	Amount float64
}

func GetTransaction(msg Message) Transaction {
	var transaction Transaction
	err := json.Unmarshal([]byte(msg.Msg), &transaction)
	if err != nil {
		log.Fatal(err)
	}
	return transaction
}
