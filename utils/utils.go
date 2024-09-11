package utils

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// GetToday 获取今天时间戳
func GetToday() int64 {
	now := time.Now()
	// 获取当天零点的时间
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	// 将时间转换为 Unix 时间戳
	timestamp := todayStart.UnixMilli()
	return timestamp
}

// 获取现在时间戳
func GetNowUnixMilli() *int64 {
	now := time.Now().UnixMilli()
	// 将时间转换为 Unix 时间戳
	return &now
}

// GenerateIDList 生成指定范围内的ID列表
func GenerateIDList(start, end int64) []int64 {
	var ids []int64
	for i := start + 1; i <= end; i++ { // 从 start+1 开始生成
		ids = append(ids, i)
	}
	return ids
}

// 判断文件是否存在

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

// 辅助函数: 生成 SQL 查询占位符
func BuildPlaceholders(n int) string {
	placeholders := make([]string, n)
	for i := range placeholders {
		placeholders[i] = "?"
	}
	return strings.Join(placeholders, ", ")
}

// 将字符串转换为时间戳
func TimeStrToUnixMilli(timeStr string) (int64, error) {
	// 定义时间字符串
	// timeStr := "2019-04-08 00:11:49"

	// 解析时间字符串为 time.Time 对象
	layout := "2006-01-02 15:04:05" // Go 的时间格式基于这个特殊的日期
	t, err := time.Parse(layout, timeStr)
	if err != nil {
		fmt.Println("Error parsing time:", err)
		return 0, err
	}
	// 转换为 Unix 时间戳
	return t.UnixMilli(), nil
}
