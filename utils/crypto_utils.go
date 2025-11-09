package utils

import (
	"crypto/md5"
	"encoding/hex"
)

// CalculateMD5 计算字符串的MD5哈希值，返回32位小写十六进制字符串
func CalculateMD5(input string) string {
	hasher := md5.New()
	hasher.Write([]byte(input))
	return hex.EncodeToString(hasher.Sum(nil))
}

// CalculateAuthorizationHeader 计算Authorization头的值：apiKey+timestamp后4位的MD5值
func CalculateAuthorizationHeader(apiKey string, timestampLastFourDigits string) string {
	authStr := apiKey + timestampLastFourDigits
	return CalculateMD5(authStr)
}
