package Data

type Block struct {
	Index     int
	TimeStamp string
	Hash      string
	PrevHash  string
	Txns      []Transaction
}
