package Data

type Message struct {
	Type       string
	Msg        string
	SenderAddr string
	Score      map[string]int64
	BlockId    int
	IsAlert    bool
}
