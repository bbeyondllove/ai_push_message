package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// 加载.env文件
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("无法加载.env文件: %v", err)
	}

	// 从.env文件读取EXTERNAL_API_KEY
	apiKey := os.Getenv("EXTERNAL_API_KEY")
	if apiKey == "" {
		log.Fatalf("EXTERNAL_API_KEY未在.env文件中设置")
	}

	fmt.Printf("从.env读取的API密钥: %s\n", apiKey)

	// 时间戳后4位
	timestampLastFourDigits := "2779"
	
	// 拼接apiKey和时间戳后4位
	input := apiKey + timestampLastFourDigits
	
	// 使用标准库计算MD5
	hasher := md5.New()
	hasher.Write([]byte(input))
	result := hex.EncodeToString(hasher.Sum(nil))
	
	fmt.Printf("输入: %s\n", input)
	fmt.Printf("MD5结果: %s\n", result)
	fmt.Printf("日志中的结果: %s\n", "bf840e4b13bbb190e7b416cee0f86331")
	fmt.Printf("是否匹配: %v\n", result == "bf840e4b13bbb190e7b416cee0f86331")
	
	// 测试8510的情况
	timestampLastFourDigits2 := "8510"
	input2 := apiKey + timestampLastFourDigits2
	hasher.Reset()
	hasher.Write([]byte(input2))
	result2 := hex.EncodeToString(hasher.Sum(nil))
	
	fmt.Printf("\n输入: %s\n", input2)
	fmt.Printf("MD5结果: %s\n", result2)
	fmt.Printf("期望结果: %s\n", "8063bfb9300cdb4766d025e1a8b49209")
	fmt.Printf("是否匹配: %v\n", result2 == "8063bfb9300cdb4766d025e1a8b49209")
}