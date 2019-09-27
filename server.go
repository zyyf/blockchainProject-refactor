package main

import (
	"blockchainProject-refactor/Data"
	"blockchainProject-refactor/MyUtil"
	"encoding/json"
	"fmt"
	"gopkg.in/dedis/crypto.v0/ints"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

// 输出实验数据相关变量
var file *os.File
var expId string
var testFileByTxn string
var testFileByTime string
var testFileByShard string
var testFileByScore string

// 服务器的地址和监听端口
var serverIP string
var serverPort string

// 连入的节点列表
var nodeList []Data.Node
var nodeListMutex sync.Mutex

// 分片回执计数
var shardConfirmCount int
var shardConfirmCountMutex sync.Mutex
var shardConfirmChan = make(chan int) // 阻塞分片后的第一笔交易的发送

// 区块链状态变量
var shard map[Data.Node]int   // 分片情况:节点-分片编号
var shardLeader []Data.Node   // 分片领导者：分片编号-领导者节点
var currentEpoch = 0          // 当前时期数
var confirmedTxnNum = 0       // 本时期已收到交易数量
var confirmedTxnNumForTps = 0 // 用于计算tps的交易计数器
var ctMutex sync.Mutex
var toBeProcTxnNum = 0 // 待确认交易数量
// var tnMutex sync.Mutex // 交易数量锁

// 超时相关变量
var realConfirmTxnNum int
var isTimeOut = make(map[int]bool)
var blockId = 0

// 在服务器发出N笔交易后，阻塞通道，直到收回N笔交易，开始分片
var startShardChan = make(chan int)

// 让分片轮流处理交易的计数器
var toShardId = 0

var shardSize int

// 计算吞吐量的计时器
var isStartTest = false
var startTime time.Time
var startTime2 time.Time
var startTime3 time.Time
var startTime4 time.Time

// 节点评分记录
var score = make([]map[Data.Node]int64, 200, 200)
var recentScore = make(map[Data.Node]int64)
var scoreMutex sync.Mutex
var nodeAddrList = make(map[string]Data.Node) // 节点addr-节点

// 处理传入连接
func handleConnInServer(conn net.Conn) {
	// 声明一个临时缓冲区，用来存储被截断的数据
	tmpBuffer := make([]byte, 0)

	// 声明一个管道用于接收解包的数据
	readerChannel := make(chan []byte, 64)

	go func(readerChannel chan []byte) {
		for {
			select {
			case msgIn := <-readerChannel:
				var msg Data.Message
				err := json.Unmarshal(msgIn, &msg)
				if err != nil {
					log.Fatal(err)
				}

				// 根据Message的Type类别处理数据
				switch msg.Type {
				case "NewNode": // 有新节点连入
					nodeListMutex.Lock()
					nodeList = append(nodeList, Data.Node{
						Addr: msg.Msg,
					})
					nodeAddrList[msg.Msg] = nodeList[len(nodeList)-1] // 将 Node 对象绑定到 addr-node map 中
					log.Println("[INFO] 节点加入：(" + strconv.Itoa(len(nodeList)) + ") " + msg.Msg)

					// 判断节点数量是否已经达到预期数量
					if nodeNum, _ := strconv.Atoi(os.Getenv("NODENUM")); len(nodeList) == nodeNum {
						startSharding() // 预期数量节点已加入，开始分片
						startTime4 = time.Now()
						go sendTxn()
					}
					nodeListMutex.Unlock()
				case "TxnConfirm": // 交易确认
					ctMutex.Lock()
					confirmedTxnNum++
					if !msg.IsAlert {
						realConfirmTxnNum++
					}
					if confirmedTxnNum == 1 {
						log.Println("一切正常")
					}
					confirmedTxnNumForTps++
					// 检查测试是否结束
					// testTxnNum, _ := strconv.Atoi(os.Getenv("TESTTXNNUM"))
					// if confirmedTxnNum == testTxnNum {
					// waitTestStop.Wait()
					// endTime = time.Now()
					duringTime := time.Since(startTime2).Seconds()
					if duringTime > 5 { // TODO 输出吞吐量的时间间隔
						tps := float64(confirmedTxnNumForTps) / duringTime
						// fmt.Println("测试结束：")
						// fmt.Println("测试开始时间：" + startTime.Format("2006-01-02 15:04:05.0000"))
						// fmt.Println("测试结束时间：" + endTime.Format("2006-01-02 15:04:05.0000"))
						duringTime = time.Since(startTime).Seconds()
						testTime := strconv.FormatFloat(duringTime, 'f', -1, 64)
						txnNum := strconv.Itoa(confirmedTxnNum)
						avgTps := strconv.FormatFloat(tps, 'f', -1, 64)
						fmt.Println("测试时长：" + testTime + " 秒")
						fmt.Println("总交易数量：" + txnNum)
						fmt.Println("吞吐量为：" + avgTps + " tps")
						// 文件输出
						file, err = os.OpenFile(testFileByTxn, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 777)
						if err != nil {
							log.Fatal(err)
						}
						_, _ = fmt.Fprintf(file, "%s,%s\n", txnNum, avgTps)
						_ = file.Close()
						confirmedTxnNumForTps = 0
						startTime2 = time.Now()
					}
					duringTime = time.Since(startTime).Seconds()
					if duringTime > 5000 { // TODO 实验时间
						_ = file.Close()
						log.Fatal()
					}
					epochTxnNum, _ := strconv.Atoi(os.Getenv("EPOCHTXNNUM")) // 每个时期的交易数量
					if confirmedTxnNum%epochTxnNum == 0 && isTimeOut[msg.BlockId] {
						isTimeOut[msg.BlockId] = false
						startShardChan <- 1
					}

					ctMutex.Unlock()
				case "ShardConfirm": // 分片确认回执
					shardConfirmCountMutex.Lock()
					shardConfirmCount++
					// log.Println("收到分片回执 (" + strconv.Itoa(shardConfirmCount) + ") " + msg.SenderAddr)
					if shardConfirmCount == len(nodeList) && currentEpoch != 0 {
						shardConfirmChan <- 1
					}
					shardConfirmCountMutex.Unlock()
				case "Score": // 收到评分消息
					scoreMutex.Lock()

					for addr, s := range msg.Score {
						score[currentEpoch][nodeAddrList[addr]] += s
					}
					scoreMutex.Unlock()
				case "Alert": // 收到警告消息，清空领导者评分
					scoreMutex.Lock()
					for _, epochScore := range score {
						for node := range epochScore {
							if node.Addr == msg.SenderAddr {
								epochScore[node] = 0
								break
							}
						}
					}
					scoreMutex.Unlock()
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

func startSharding() {
	// TODO 分片耗时实验：实验内容为分片阶段消耗的时间
	// POW耗时
	// rnd := rand.Intn(3) + 10
	// time.Sleep(time.Duration(rnd) * time.Millisecond)
	// 分片耗时实验结束

	// TODO 行为评价代码 初始化每个时期的 score
	scoreMutex.Lock()
	score[currentEpoch] = make(map[Data.Node]int64)
	scoreMutex.Unlock()
	// 行为评价代码结束

	shardConfirmCountMutex.Lock()
	shardConfirmCount = 0 // 将分片回复计数器置零
	shardConfirmCountMutex.Unlock()

	log.Println("时期 " + strconv.Itoa(currentEpoch) + " 分片开始")
	shard = make(map[Data.Node]int)
	shardLeader = make([]Data.Node, len(nodeList)/shardSize, len(nodeList)/shardSize)
	// 通过环境变量获取分片方式：0为随机分片，1为按ip分片（同一ip在一片）
	shardMethod := os.Getenv("SHARDMETHOD")
	switch shardMethod {
	case "0":
		// -------------随机分片----------------
		rand.Seed(time.Now().UnixNano())
		randResults := make(map[int]Data.Node)
		// 给每个节点生成一个随机不重复的值
		for i := 0; i < len(nodeList); i++ {
			randNum := rand.Intn(100000)
			_, ok := randResults[randNum]
			for ok {
				randNum = rand.Intn(100000)
				_, ok = randResults[randNum]
			}
			randResults[randNum] = nodeList[i]
		}
		var keys []int
		for k := range randResults {
			keys = append(keys, k)
		}
		sort.Ints(keys)
		shardId, index := 0, 0
		for _, k := range keys {
			shard[randResults[k]] = shardId
			index++
			if index%shardSize == 0 {
				shardLeader[shardId] = randResults[k]
				// shardLeader = append(shardLeader, randResults[k])
				shardId++
			}
		}
	case "1":
		num := len(nodeList) / shardSize
		// shardNum:=make([]int,num,num)
		for i := 0; i < num; i++ {
			for {
				rnd := rand.Intn(len(nodeList))
				if _, ok := shard[nodeList[rnd]]; !ok {
					shard[nodeList[rnd]] = i
					shardLeader[i] = nodeList[rnd]
					for j := 0; j < shardSize-1; j++ {
						min := math.MaxInt32
						var minIndex Data.Node
						for _, node := range nodeList {
							_, ok := shard[node]
							if !ok && (dist1(node, nodeList[rnd]) < min) {
								min = dist1(node, nodeList[rnd])
								minIndex = node
							}
						}
						shard[minIndex] = i
					}
					break
				}
			}
		}
	case "2":
		// 实验二分片策略
		num := len(nodeList) / shardSize // 分片数量
		shardIndex := 0
		shardNum := make([]int, num, num)
		nPerfList1 := make([]Data.Node, 0, 25)
		nPerfList2 := make([]Data.Node, 0, 25)
		nPerfList3 := make([]Data.Node, 0, 25)
		nPerfList4 := make([]Data.Node, 0, 25)
		for _, node := range nodeList {
			switch strings.Split(node.Addr, ":")[1][1:2] {
			case "1":
				nPerfList1 = append(nPerfList1, node)
			case "2":
				nPerfList2 = append(nPerfList2, node)
			case "4":
				nPerfList3 = append(nPerfList3, node)
			case "8":
				nPerfList4 = append(nPerfList4, node)
			}
		}
		for _, node := range nPerfList1 {
			shard[node] = shardIndex
			if shardNum[shardIndex] == 0 {
				shardLeader[shardIndex] = node
			}
			shardNum[shardIndex]++
			if shardIndex == num-1 {
				shardIndex = 0
			} else {
				shardIndex++
			}
		}
		for _, node := range nPerfList2 {
			shard[node] = shardIndex
			if shardNum[shardIndex] == 0 {
				shardLeader[shardIndex] = node
			}
			shardNum[shardIndex]++
			if shardIndex == num-1 {
				shardIndex = 0
			} else {
				shardIndex++
			}
		}
		for _, node := range nPerfList3 {
			shard[node] = shardIndex
			if shardNum[shardIndex] == 0 {
				shardLeader[shardIndex] = node
			}
			shardNum[shardIndex]++
			if shardIndex == num-1 {
				shardIndex = 0
			} else {
				shardIndex++
			}
		}
		for _, node := range nPerfList4 {
			shard[node] = shardIndex
			if shardNum[shardIndex] == 0 {
				shardLeader[shardIndex] = node
			}
			shardNum[shardIndex]++
			if shardIndex == num-1 {
				shardIndex = 0
			} else {
				shardIndex++
			}
		}
	case "3":
		// 实验3分片策略
		shardIndex := 0
		shardExtraIndex := 200/shardSize + 200/shardSize + 400/shardSize
		num := len(nodeList) / shardSize
		fullIndex := shardExtraIndex - 1
		shardNum := make([]int, num, num)
		for i := 0; i < 400; i++ {
			shard[nodeList[i]] = shardIndex
			if shardNum[shardIndex] == 0 {
				shardLeader[shardIndex] = nodeList[i]
			}
			shardNum[shardIndex]++
			if shardIndex == fullIndex {
				shardIndex = 0
				if (399 - i) < num {
					for i++; i < 400; i++ {
						shard[nodeList[i]] = shardExtraIndex
						if shardNum[shardExtraIndex] == 0 {
							shardLeader[shardExtraIndex] = nodeList[i]
						}
						shardNum[shardExtraIndex]++
						if shardExtraIndex == num-1 {
							shardExtraIndex = fullIndex + 1
						} else {
							shardExtraIndex++
						}
					}
					break
				}
			} else {
				shardIndex++
			}
		}
		for i := 400; i < 600; i++ {
			shard[nodeList[i]] = shardIndex
			if shardNum[shardIndex] == 0 {
				shardLeader[shardIndex] = nodeList[i]
			}
			shardNum[shardIndex]++
			if shardIndex == fullIndex {
				shardIndex = 0
				if (599 - i) < num {
					for i++; i < 600; i++ {
						shard[nodeList[i]] = shardExtraIndex
						if shardNum[shardExtraIndex] == 0 {
							shardLeader[shardExtraIndex] = nodeList[i]
						}
						shardNum[shardExtraIndex]++
						if shardExtraIndex == num-1 {
							shardExtraIndex = fullIndex + 1
						} else {
							shardExtraIndex++
						}
					}
					break
				}
			} else {
				shardIndex++
			}
		}
		for i := 600; i < 800; i++ {
			shard[nodeList[i]] = shardIndex
			if shardNum[shardIndex] == 0 {
				shardLeader[shardIndex] = nodeList[i]
			}
			shardNum[shardIndex]++
			if shardIndex == fullIndex {
				shardIndex = 0
				if (799 - i) < num {
					for i++; i < 800; i++ {
						shard[nodeList[i]] = shardExtraIndex
						if shardNum[shardExtraIndex] == 0 {
							shardLeader[shardExtraIndex] = nodeList[i]
						}
						shardNum[shardExtraIndex]++
						if shardExtraIndex == num-1 {
							shardExtraIndex = fullIndex + 1
						} else {
							shardExtraIndex++
						}
					}
					break
				}
			} else {
				shardIndex++
			}
		}
	case "5": // 行为评价分片
		if currentEpoch == 0 { // TODO 随机分片轮数
			log.Println("本次为随机分片")
			rand.Seed(time.Now().UnixNano())
			randResults := make(map[int]Data.Node)
			// 给每个节点生成一个随机不重复的值
			for i := 0; i < len(nodeList); i++ {
				randNum := rand.Intn(100000)
				_, ok := randResults[randNum]
				for ok {
					randNum = rand.Intn(100000)
					_, ok = randResults[randNum]
				}
				randResults[randNum] = nodeList[i]
			}
			var keys []int
			for k := range randResults {
				keys = append(keys, k)
			}
			sort.Ints(keys)
			shardId, index := 0, 0
			for _, k := range keys {
				shard[randResults[k]] = shardId
				index++
				if index%shardSize == 0 {
					shardLeader[shardId] = randResults[k]
					// shardLeader = append(shardLeader, randResults[k])
					shardId++
				}
			}
			for _, node := range nodeList {
				score[currentEpoch][node] = 0
			}
		} else {
			// 对评分进行排序，生成一个分数降序的节点切片
			recentScore = make(map[Data.Node]int64)
			for i := currentEpoch - 1; i > currentEpoch-10 && i >= 0; i-- { // TODO 分数的滑动窗口
				for node, nodeScore := range score[i] {
					recentScore[node] += nodeScore
				}
			}
			sortedScore := make([]Data.Node, 0, len(nodeList))
			for node, nodeScore := range recentScore {
				if len(sortedScore) == 0 {
					sortedScore = append(sortedScore, node)
				} else {
					isInsert := false
					for index, v := range sortedScore {
						if recentScore[v] <= nodeScore {
							s := []Data.Node{node}
							s1 := MyUtil.MergeNodeSlice(sortedScore[0:index], s)
							sortedScore = MyUtil.MergeNodeSlice(s1, sortedScore[index:])
							isInsert = true
							break
						}
					}
					if !isInsert {
						sortedScore = append(sortedScore, node)
					}
				}
			}

			// TODO 开始按评分分片
			num := len(nodeList) / shardSize
			// 复现rep chain
			nodeIndex := 0
			shardList := make([][]Data.Node, num, num) // 分片id-节点列表
			for i := 0; i < num; i++ {
				shardList[i] = make([]Data.Node, 0, shardSize)
			}
			rand.Seed(time.Now().UnixNano())
			for i := 0; i < shardSize; i++ {
				// 生成一个0到num-1的随机序列  洗牌算法
				list := make([]int, num, num)
				for j := 0; j < num; j++ {
					list[j] = j
				}
				for j := num - 1; j >= 0; j-- {
					rnd := rand.Intn(j + 1)
					temp := list[rnd]
					list[rnd] = list[j]
					list[j] = temp
				}
				for _, id := range list {
					n := sortedScore[nodeIndex]
					shard[n] = id
					shardList[id] = append(shardList[id], sortedScore[nodeIndex])
					nodeIndex++
				}
			}
			// log.Println("分片完毕，开始选择领导者")
			// 开始选择领导者
			for i := 0; i < num; i++ {
				for j := 0; j < len(shardList); j++ {
					max := recentScore[shardList[i][j]]
					maxIndex := j
					for k := j + 1; k < len(shardList[i]); k++ {
						if recentScore[shardList[i][k]] > max {
							temp := shardList[i][k]
							shardList[i][k] = shardList[i][maxIndex]
							shardList[i][maxIndex] = temp
							maxIndex = k
							max = recentScore[shardList[i][k]]
						}
					}
				}
				// 随机选评分大于中位数的一个节点
				for {
					rnd := rand.Intn(shardSize / 2)
					if recentScore[shardList[i][rnd]] > recentScore[shardList[i][shardSize/2]] {
						shardLeader[i] = shardList[i][rnd]
						break
					} else if recentScore[shardList[i][0]] == recentScore[shardList[i][shardSize/2]] {
						shardLeader[i] = shardList[i][0]
						break
					}
				}
			}
			for _, node := range nodeList {
				score[currentEpoch][node] = 0
			}
			// 下面是按照大小分片
			// shardScore:=make([]int64,num,num)
			// nodeIndex:=0
			// shardNum:=make([]int,num,num)
			// // 分数最高的为领导者
			// for i:=0;i<num;i++{
			// 	shard[sortedScore[nodeIndex]]=i
			// 	shardNum[i]++
			// 	shardLeader[i]=sortedScore[nodeIndex]
			// 	shardScore[i]+=recentScore[sortedScore[nodeIndex]]
			// 	nodeIndex++
			// }
			// for index,node:=range sortedScore{
			// 	if index<nodeIndex{
			// 		continue
			// 	}
			// 	var minShard int64 =math.MaxInt64
			// 	minIndex:=0
			// 	for i:=0;i<num;i++{
			// 		if shardScore[i]<minShard && shardNum[i]<shardSizeClient{
			// 			minIndex=i
			// 			minShard=shardScore[i]
			// 		}
			// 	}
			// 	shard[node]=minIndex
			// 	shardScore[minIndex]+=recentScore[node]
			// 	shardNum[minIndex]++
			// }
		}
	}

	// 向每个节点发送分片结果
	// printShardResult()
	for _, n := range nodeList {
		shardId := shard[n]
		var isLeader string
		shardNodeList := ""
		if shardLeader[shardId] == n {
			isLeader = "1" // 该节点为分片领导者
			for node, shardid := range shard {
				if shardid == shardId && node.Addr != n.Addr {
					shardNodeList = shardNodeList + node.Addr + ","
				}
			}
		} else {
			isLeader = "0"
			shardNodeList = shardNodeList + shardLeader[shardId].Addr
		}
		shardSize, _ := strconv.Atoi(os.Getenv("SHARDSIZE"))
		msg := Data.Message{
			Type: "Shard",
			Msg:  strconv.Itoa(shardSize) + "," + strconv.Itoa(shardId) + "," + isLeader + "," + shardNodeList,
		}
		// log.Println("发送分片结果给 " + n.Addr + " (" + strconv.Itoa(index) + ")")
		MyUtil.SendMessage(n.Addr, msg)
	}

	log.Println("时期 " + strconv.Itoa(currentEpoch) + " 分片结束")
}

func main() {
	// 实验参数设置
	_ = godotenv.Load("const.env")
	shardSize, _ = strconv.Atoi(os.Getenv("SHARDSIZE"))
	shardMethod := os.Getenv("SHARDMETHOD")
	serverPort = os.Getenv("SERVERPORT")
	isLocal := os.Getenv("ISLOCALTEST")
	if isLocal == "1" {
		serverIP = "127.0.0.1"
	} else {
		serverIP = MyUtil.GetOutboundIP().String()
	}

	// 实验数据文件名设置
	expId = "exp(throughput)" // TODO 数据输出文件名
	testFileByScore = "result/output-" + expId + "-score" + ".csv"
	testFileByTxn = "result/output-" + expId + "-byTxn-" + shardMethod + ".csv"
	testFileByTime = "result/output-" + expId + "-byTime-" + shardMethod + ".csv"
	testFileByShard = "result/output-" + expId + "-byShard-" + shardMethod + ".csv"
	MyUtil.DeleteExistFile(testFileByShard)
	MyUtil.DeleteExistFile(testFileByTxn)
	MyUtil.DeleteExistFile(testFileByScore)

	// 日志设置
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("[INFO] 服务器启动")

	serverAddr := serverIP + ":" + serverPort
	listener, _ := net.Listen("tcp", serverAddr)
	log.Println("开始监听: " + serverAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleConnInServer(conn) // 处理传入的连接
	}

}

func sendTxn() {
	log.Println("测试开始")
	if !isStartTest {
		startTime = time.Now()
		startTime2 = time.Now()
		startTime3 = time.Now()
		isStartTest = true
		go func() { // 初始化记录文件
			MyUtil.DeleteExistFile(testFileByTime)
			file, err := os.OpenFile(testFileByTime, os.O_WRONLY|os.O_CREATE, 0666)
			if err != nil {
				log.Fatal(err)
			}
			dtime := 0
			for {
				ctMutex.Lock()
				// txnNum := strconv.Itoa(confirmedTxnNum)
				tps := float64(confirmedTxnNum) / float64(dtime)
				ctMutex.Unlock()
				tpsStr := strconv.FormatFloat(tps, 'f', -1, 64)
				_, _ = fmt.Fprintf(file, "%s,%s\n", strconv.Itoa(dtime), tpsStr)
				dtime += 5
				time.Sleep(5 * time.Second)
			}
		}()
	}
	epochTxnNum, _ := strconv.Atoi(os.Getenv("EPOCHTXNNUM")) // 每个时期的交易数量
	var randStrSeed = "0123456789qwertyuiopasdfghjklzxcvbnm"
	id := ""
	toShardId = 0
	rand.Seed(time.Now().UnixNano())
	for {
		for i := 0; i < 20; i++ {
			id = id + string(randStrSeed[rand.Intn(len(randStrSeed))])
		}
		txnMsg := Data.Message{
			Type:    "Txn",
			Msg:     id,
			BlockId: blockId,
		}
		id = ""

		go MyUtil.SendMessage(shardLeader[toShardId].Addr, txnMsg)
		if toShardId < len(shardLeader)-1 {
			toShardId++
		} else {
			toShardId = 0
		}
		toBeProcTxnNum++
		if toBeProcTxnNum%epochTxnNum == 0 && toBeProcTxnNum != 0 {
			// log.Println("等待交易确认完成")
			// 等待剩下的交易确认完成
			ctMutex.Lock()
			isTimeOut[blockId] = true
			ctMutex.Unlock()
			// go func(blockId int) { // TODO 安全实验超时内容代码
			// 	time.Sleep(200 * time.Millisecond) // TODO TIMEOUT
			// 	ctMutex.Lock()
			// 	if isTimeOut[blockId] {
			// 		// log.Println("已超时，继续发送交易")
			// 		confirmedTxnNum += epochTxnNum - (confirmedTxnNum % epochTxnNum)
			// 		isTimeOut[blockId] = false
			// 		startShardChan <- 1
			// 	}
			// 	ctMutex.Unlock()
			// }(blockId)
			<-startShardChan
			// 交易已经全部确认完成
			blockId++

			if time.Since(startTime3).Seconds() > 20 { // TODO 分片间隔
				if currentEpoch%1 == 0 {
					printScore(currentEpoch)
				}
				if currentEpoch == 100 { // TODO 按分片数停止实验
					log.Fatal()
				}
				file, err := os.OpenFile(testFileByShard, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 777)
				if err != nil {
					log.Fatal(err)
				}
				ctMutex.Lock()
				tps := float64(realConfirmTxnNum) / time.Since(startTime4).Seconds()
				realConfirmTxnNum = 0
				ctMutex.Unlock()
				_, _ = fmt.Fprintf(file, "%s,%f\n", strconv.Itoa(currentEpoch), tps)
				_ = file.Close()
				log.Println("时期 " + strconv.Itoa(currentEpoch) + " 结束")
				currentEpoch++
				startSharding()
				// printShardResult()
				startTime4 = time.Now()
				<-shardConfirmChan // 等待收到全部的分片回执
				log.Println("时期 " + strconv.Itoa(currentEpoch) + " 开始")
				startTime3 = time.Now()
				rand.Seed(time.Now().UnixNano())
			}
		}
	}
}

// printScore
func printScore(epoch int) {
	file, err := os.OpenFile(testFileByScore, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 777)
	if err != nil {
		log.Fatal(err)
	}
	// for node,s:=range score{
	// 	_,_=fmt.Fprintf(file, "%s,%s\n", node.Addr,strconv.Itoa(int(s)))
	// }
	for _, node := range nodeList {
		_, _ = fmt.Fprintf(file, "%s,%s,%s\n", strconv.Itoa(epoch), strings.Split(node.Addr, ":")[1], strconv.Itoa(int(score[epoch][node])))
	}
	_ = file.Close()
}

// printShardResult
func printShardResult() {
	shardSize, _ := strconv.Atoi(os.Getenv("SHARDSIZE"))
	num := len(nodeList) / shardSize
	for i := 0; i < num; i++ {
		log.Println("分片 " + strconv.Itoa(i) + " 领导者 " + shardLeader[i].Addr)
		for _, node := range nodeList {
			if shard[node] == i {
				log.Println(node.Addr)
			}
		}
	}
}

func dist1(node1 Data.Node, node2 Data.Node) int {
	type1, _ := strconv.Atoi(strings.Split(node1.Addr, ":")[1][1:2])
	type2, _ := strconv.Atoi(strings.Split(node2.Addr, ":")[1][1:2])
	return ints.Abs(type1 - type2)
}
