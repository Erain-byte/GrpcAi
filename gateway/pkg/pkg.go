package pkg

import "fmt"

// Response 成功响应
func Success(data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"code":    200,
		"message": "success",
		"data":    data,
	}
}

// Error 错误响应
func Error(code int, message string) map[string]interface{} {
	return map[string]interface{}{
		"code":    code,
		"message": message,
	}
}

// FormatAddress 格式化地址
func FormatAddress(host string, port int) string {
	return fmt.Sprintf("%s:%d", host, port)
}
