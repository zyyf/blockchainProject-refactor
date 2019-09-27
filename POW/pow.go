package POW

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"log"
	"math"
	"math/big"
	"time"
)

var (
	maxNonce = math.MaxInt64
)

// pow结构
type Pow struct {
	Start int
	//TargetBits int
	Target *big.Int
}

// 准备计算nonce值的数据
func (pow *Pow) prepareData(nodeRand int64, nonce int) []byte {
	data := bytes.Join(
		[][]byte{
			IntToHex(nodeRand),
			IntToHex(int64(pow.Start)),
			//IntToHex(int64(pow.TargetBits)),
			IntToHex(int64(nonce)),
		},
		[]byte{},
	)

	return data
}

// 进行pow计算
func (pow *Pow) Run(nodeRand int64) (float64, int, []byte) {
	var hashInt big.Int
	var hash [32]byte
	nonce := 0
	//target := big.NewInt(1)
	//target.Lsh(target, uint(256-pow.TargetBits))

	start := time.Now()
	for nonce < maxNonce {
		data := pow.prepareData(nodeRand, nonce)
		hash = sha256.Sum256(data)
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(pow.Target) == -1 {
			break
		} else {
			nonce++
		}
	}
	cur := time.Now()
	elapsed := cur.Sub(start).Seconds()

	return elapsed, nonce, hash[:]
}

// IntToHex将int64的数转为byte数组
func IntToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}
