package MyUtil

import (
	"log"
	"os"
)

func IsFileExists(path string) bool {
	_, err := os.Stat(path) // os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func DeleteExistFile(path string) {
	if IsFileExists(path) {
		err := os.Remove(path)
		if err != nil {
			log.Panic(err)
		}
	}
}
