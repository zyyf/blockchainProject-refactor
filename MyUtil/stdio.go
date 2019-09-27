package MyUtil

import (
	"bufio"
	"log"
	"os"
	"strings"
)

// 从控制台输入一行字符串
func ReadFromStdin() string {
	stdread := bufio.NewReader(os.Stdin)

	input, err := stdread.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	input = strings.Replace(input, "\r", "", -1)
	return input
}
