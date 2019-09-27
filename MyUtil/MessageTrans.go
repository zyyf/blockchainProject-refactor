package MyUtil

import (
	"blockchainProject-refactor/Data"
	"encoding/json"
	"github.com/davecgh/go-spew/spew"
	"log"
	"net"
	"sync"
)

var connections = make(map[string]net.Conn)
var connMutex sync.RWMutex

// 发送消息给指定地址
func SendMessage(toAddr string, msg Data.Message) {
	// time.Sleep(time.Millisecond)
	msgjson, err := json.Marshal(msg)
	if err != nil {
		log.Fatal(err)
	}
	connMutex.Lock()
	if conn, isExist := connections[toAddr]; isExist {
		Sender(conn, msgjson)
	} else {
		// tcpAddr, err := net.ResolveTCPAddr("tcp4", toAddr)
		// if err != nil {
		// 	log.Fatal(err)
		// }
		conn, err := net.Dial("tcp", toAddr)
		// conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			log.Panic("panice:::" + toAddr)
			spew.Dump(msg)
			log.Fatal(err)
		}
		connections[toAddr] = conn
		Sender(conn, msgjson)
	}
	connMutex.Unlock()
}

// 关闭所有连接
func CloseConnections() {
	connMutex.Lock()
	for toAddr, conn := range connections {
		if toAddr != "127.0.0.1:30000" {
			delete(connections, toAddr)
			err := conn.Close()
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	connMutex.Unlock()
}
