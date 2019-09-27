package main

import (
	"blockchainProject-refactor/Data"
	"blockchainProject-refactor/MyUtil"
	"encoding/json"
	"flag"
	"gopkg.in/dedis/crypto.v0/ints"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

// 服务器地址
var serverAddr string

// 监听的端口
var port string
var nodeIP string
var nodeAddr string

// 节点分片信息
// var shardId int                          // 节点所在分片编号
var isLeader bool                        // 节点是否为分片领导者
var shardNodeList []Data.Node            // 分片内节点列表
var leader Data.Node                     // 分片领导者
var shardSizeClient int                  // 分片大小
var txnConfirmNum = make(map[string]int) // 每个交易被确认的次数
var TCNMutex sync.Mutex

// 行为评价相关
var nodeStartTime = make(map[string]map[string]time.Time)    // 节点addr-交易ID-start time
var nodeCostTime = make(map[string]map[string]time.Duration) // 节点addr-交易id-花费时间
var nodeScore = make(map[string]int64)                       // 节点addr-评分
var nodeTxnStat = make(map[string]map[string]bool)           // 节点addr-交易-是否确认
var scMutex sync.RWMutex

var epoch = -1

// 处理传入消息
func handleConnInClient(conn net.Conn) {
	// 声明一个临时缓冲区，用来存储被截断的数据
	tmpBuffer := make([]byte, 0)

	// 声明一个管道用于接收解包的数据
	readerChannel := make(chan []byte, 16)

	go func(readerChannel chan []byte) {
		for {
			select {
			case msgin := <-readerChannel:
				var msg Data.Message
				err := json.Unmarshal(msgin, &msg)
				if err != nil {
					log.Fatal(err)
				}
				// 根据消息类型处理消息
				switch msg.Type {
				case "Shard":
					log.Println("收到分片信息")
					// shardSizeClient, _ = strconv.Atoi(strings.Split(msg.Msg, ",")[0])
					shardNodeList = make([]Data.Node, 0, shardSizeClient-1)
					// shardId, err = strconv.Atoi(strings.Split(msg.Msg, ",")[1])
					if err != nil {
						log.Fatal(err)
					}
					isLeaderStr := strings.Split(msg.Msg, ",")[2]
					if isLeaderStr == "1" { // 是领导者
						log.Println("是领导者")
						log.SetPrefix("----")
						log.Println("分片成员有：")
						isLeader = true
						leader = Data.Node{
							Addr: nodeIP + ":" + port,
						}
						for i, nodeAddr := range strings.Split(msg.Msg, ",") {
							if i != 0 && i != 1 && i != 2 && nodeAddr != "" {
								shardNodeList = append(shardNodeList, Data.Node{
									Addr: nodeAddr,
								})
								log.Println(nodeAddr)
							}
						}
						log.SetPrefix("")
					} else { // 不是领导者
						log.Println("不是领导者")
						log.SetPrefix("----")
						log.Println("领导者为：")
						isLeader = false
						leader = Data.Node{
							Addr: strings.Split(msg.Msg, ",")[3],
						}
						log.Println(strings.Split(msg.Msg, ",")[3])
						log.SetPrefix("")
					}
					log.Println("分片结束")
					// 向领导者回复-------回复服务器示意已经收到分片结果
					shardConfirmMsg := Data.Message{
						Type:       "ShardConfirm",
						SenderAddr: nodeAddr,
					}
					// time.Sleep(100 * time.Millisecond)
					MyUtil.CloseConnections()
					nodeStartTime = make(map[string]map[string]time.Time)
					nodeCostTime = make(map[string]map[string]time.Duration)
					nodeScore = make(map[string]int64)
					nodeTxnStat = make(map[string]map[string]bool)
					// log.Println(*port + " 发送分片回执")
					log.Println(serverAddr)
					MyUtil.SendMessage(serverAddr, shardConfirmMsg)

					epoch++

				case "Txn":
					if isLeader { // 领导者收到交易，直接转发给片内其他节点，等待达成共识
						log.Println("作为领导者收到交易，向成员转发 " + msg.Msg)
						for _, node := range shardNodeList {
							// TODO 延迟实验代码1
							// time.Sleep(time.Millisecond * time.Duration(20*dist2(node, Data.Node{Addr: nodeAddr})))
							// 延迟实验代码结束
							// time.Sleep(time.Millisecond*100)
							// TODO 行为评价代码1（时间计分）
							// scMutex.Lock()
							// if _, ok := nodeStartTime[node.Addr]; !ok {
							// 	nodeStartTime[node.Addr] = make(map[string]time.Time)
							// }
							// nodeStartTime[node.Addr][msg.Msg] = time.Now()
							// scMutex.Unlock()
							// 行为评价代码结束
							msg.SenderAddr = nodeAddr
							MyUtil.SendMessage(node.Addr, msg)
						}
					} else { // 非领导者收到交易，开始验证，验证完成后发送确认信息给领导者
						log.Println("收到领导者" + msg.SenderAddr + "转发来的交易 " + msg.Msg)
						msg.Type = "TxnConfirm"
						// TODO 延迟实验代码2
						// time.Sleep(time.Millisecond * time.Duration(20*dist2(leader, Data.Node{Addr: nodeAddr})))
						// 延迟实验代码结束
						// TODO 性能实验代码1
						// verifyTime, _ := strconv.Atoi((*port)[1:2]) // 性能指标
						// time.Sleep(time.Duration(verifyTime*10) * time.Millisecond)
						// 性能实验代码结束
						// TODO 活跃实验代码1
						// activeType := (*port)[1:2]
						// nowSecond := time.Now().Second()
						// switch activeType {
						// case "1":
						// 	if nowSecond > 30 {
						// 		time.Sleep(time.Duration(60-nowSecond) * time.Second)
						// 	}
						// case "2":
						// 	if nowSecond <= 30 {
						// 		time.Sleep(time.Duration(30-nowSecond) * time.Second)
						// 	}
						// }
						// 活跃实验代码结束
						// TODO 行为评价代码4(节点延迟)
						// nodeType, _ := strconv.Atoi((*port)[1:2])
						// if nodeType == 2 {
						// 	alertMsg:=Data.Message{
						// 		Type:"Alert",
						// 		SenderAddr:nodeAddr,
						// 	}
						// 	Mynet.SendMessage(leader.Addr,alertMsg)
						// 	break
						// }
						// if epoch < 999 {
						// 	time.Sleep(time.Duration((nodeType)*10) * time.Millisecond)
						// } else {
						// 	time.Sleep(time.Duration((10-nodeType)*10) * time.Millisecond)
						// }
						// 行为评价代码结束
						log.Println("交易 " + msg.Msg + " 确认完毕，回复领导者")
						msg.SenderAddr = nodeAddr
						MyUtil.SendMessage(leader.Addr, msg)
					}
				case "TxnConfirm":
					// TODO 行为评价代码2
					scMutex.Lock()
					// if _, ok := nodeCostTime[msg.SenderAddr]; !ok {
					// 	nodeCostTime[msg.SenderAddr] = make(map[string]time.Duration)
					// }
					// nodeCostTime[msg.SenderAddr][msg.Msg] = time.Since(nodeStartTime[msg.SenderAddr][msg.Msg])
					if _, ok := nodeTxnStat[msg.SenderAddr]; !ok {
						nodeTxnStat[msg.SenderAddr] = make(map[string]bool)
					}
					nodeTxnStat[msg.SenderAddr][msg.Msg] = true
					scMutex.Unlock()
					// 行为评价代码结束
					TCNMutex.Lock()
					if txnConfirmNum[msg.Msg] != -1 {
						txnConfirmNum[msg.Msg]++
					}
					// 一旦达成共识，就发送确认信息给服务器
					// if txnConfirmNum[msg.Msg] == shardSizeClient-1 {
					log.Println("收到成员 " + msg.SenderAddr + " 对交易 " + msg.Msg + " 的回复信息（" + strconv.Itoa(txnConfirmNum[msg.Msg]) + "）")
					if txnConfirmNum[msg.Msg] >= shardSizeClient*2/3 { // TODO 改成了二分之一
						msg.Type = "TxnConfirm"
						txnConfirmNum[msg.Msg] = -1
						// TODO 活跃实验代码2
						// activeType := (*port)[1:2]
						// nowSecond := time.Now().Second()
						// switch activeType {
						// case "1":
						// 	if nowSecond > 30 {
						// 		time.Sleep(time.Duration(60-nowSecond) * time.Second)
						// 	}
						// case "2":
						// 	if nowSecond <= 30 {
						// 		time.Sleep(time.Duration(30-nowSecond) * time.Second)
						// 	}
						// }
						// 活跃实验代码结束
						// TODO 性能实验代码2
						// veryfyTime, _ := strconv.Atoi((*port)[1:2]) // 性能指标
						// time.Sleep(time.Duration(veryfyTime*10) * time.Millisecond)
						// 性能实验代码结束
						// time.Sleep(time.Millisecond*100)

						// TODO 行为评价代码3
						scMutex.Lock()
						timeSum := 0.0
						var count int64 = 0
						// var longestTime int64
						// longestTime = 0
						for nAddr, node := range nodeCostTime {
							if _, ok := nodeTxnStat[nAddr]; !ok {
								nodeTxnStat[nAddr] = make(map[string]bool)
							}
							if nodeTxnStat[nAddr][msg.Msg] {
								timeSum += node[msg.Msg].Seconds()
								count += 1
							}
							// 	if node[msg.Msg].Nanoseconds()/1000000 > longestTime {
							// 		longestTime = node[msg.Msg].Nanoseconds() / 1000000
							// 	}
						}
						timeAvg := timeSum / float64(count)
						var scoreSum float64 = 0
						// nodeType, _ := strconv.Atoi((*port)[1:2])
						for _, node := range shardNodeList {
							// 得分公式
							thisScore := nodeCostTime[node.Addr][msg.Msg].Seconds() / timeAvg
							// if nodeTxnStat[node.Addr][msg.Msg] { // 确认了的才加分
							// 	thisScore := longestTime / (nodeCostTime[node.Addr][msg.Msg].Nanoseconds() / 1000000)
							// 	nodeType, _ = strconv.Atoi(strings.Split(node.Addr, ":")[1][3:4])
							// 	nodeScore[node.Addr] += 1
							// }
							// if nodeType < rndNum {
							// 	nodeScore[node.Addr] += 1
							// }
							// if epoch < 999 {
							// 	nodeScore[node.Addr] += (10 - int64(nodeType)) * 10
							// } else {
							// 	nodeScore[node.Addr] += (10 - int64(10-nodeType)) * 10
							// }
							scoreSum += thisScore
							// }
						}
						// nodeType, _ = strconv.Atoi((*port)[3:4])
						// nodeScore[nodeAddr] += 1
						// var rndNum int
						// if nodeType == 0 {
						// 	rndNum = 0
						// } else {
						// 	rndNum = rand.Intn(nodeType)
						// }
						// if nodeType < rndNum {
						// 						// 	nodeScore[nodeAddr] += 1
						// 						// }
						// if epoch < 999 {
						// 	nodeScore[nodeAddr] += (10 - int64(nodeType)) * 10
						// } else {
						// 	nodeScore[nodeAddr] += (10 - int64(10-nodeType)) * 10
						// }
						nodeScore[nodeAddr] += int64(scoreSum) / count

						// TODO 行为评价代码5（领导者延迟）

						// nodeType, _ := strconv.Atoi((*port)[1:2])
						// if nodeType == 2 {
						// scoreMsg := Data.Message{
						// 	Type:       "Score",
						// 	Msg:        "",
						// 	SenderAddr: nodeAddr,
						// 	Score:      nodeScore,
						// }
						// Mynet.SendMessage(serverAddr, scoreMsg)
						// alertMsg := Data.Message{
						// 	Type:       "Alert",
						// 	SenderAddr: nodeAddr,
						// }
						// Mynet.SendMessage(serverAddr, alertMsg)
						// msg.IsAlert = true
						// scMutex.Unlock()
						// TCNMutex.Unlock()
						// break
						// }
						// nodeType, _ = strconv.Atoi((*port)[3:4])
						// if epoch < 999 {
						// 	time.Sleep(time.Duration((nodeType)*10) * time.Millisecond)
						// } else {
						// 	time.Sleep(time.Duration((10-nodeType)*10) * time.Millisecond)
						// }
						// 行为评价代码结束
						scoreMsg := Data.Message{
							Type:       "Score",
							Msg:        "",
							SenderAddr: nodeAddr,
							Score:      nodeScore,
						}
						MyUtil.SendMessage(serverAddr, scoreMsg)
						scMutex.Unlock()
						// 行为评价代码结束
						log.Println("交易 " + msg.Msg + " 最终确认，回复服务器")
						msg.SenderAddr = nodeAddr
						MyUtil.SendMessage(serverAddr, msg)
					}
					TCNMutex.Unlock()
				}
			}
		}
	}(readerChannel)

	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			// log.Println(conn.RemoteAddr().String(), " connection error: ", err)
			return
		}

		tmpBuffer = MyUtil.Unpack(append(tmpBuffer, buffer[:n]...), readerChannel)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano()) // 设置随机种子

	// 实验参数读取
	_ = godotenv.Load("const.env")
	shardSizeClient, _ = strconv.Atoi(os.Getenv("SHARDSIZE"))

	// 运行参数读取
	port = *flag.String("p", "11001", "Listen port")
	serverAddr = *flag.String("s", "127.0.0.1:30000", "Server Address")
	isLocalTest := flag.Bool("local", true, "Set the test environment")
	flag.Parse()
	if port == "" {
		log.Panic("请用 -p 输入一个有效的监听端口")
	}
	if *isLocalTest {
		nodeIP = "127.0.0.1"
	} else {
		nodeIP = MyUtil.GetOutboundIP().String()
	}

	// 设置 log
	logPath := "log/" + port + ".txt"
	MyUtil.DeleteExistFile(logPath)
	// logFile, _ := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	// log.SetOutput(logFile) // 设置输出流
	log.SetFlags(log.Ltime | log.Lshortfile)

	// 节点初始化
	log.Println("[INFO] 节点 " + port + " 启动")
	nodeAddr = nodeIP + ":" + port
	// myMap := make(map[string]int64)
	msg := Data.Message{
		Type:       "NewNode",
		Msg:        nodeAddr,
		SenderAddr: nodeAddr,
	}
	go MyUtil.SendMessage(serverAddr, msg)

	// 开始监听消息
	listener, err := net.Listen("tcp", nodeAddr)
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Panic(err)
		}
		go handleConnInClient(conn)
	}
}

func dist2(node1 Data.Node, node2 Data.Node) int {
	type1, _ := strconv.Atoi(strings.Split(node1.Addr, ":")[1][1:2])
	type2, _ := strconv.Atoi(strings.Split(node2.Addr, ":")[1][1:2])
	return ints.Abs(type1-type2) + 1
}
