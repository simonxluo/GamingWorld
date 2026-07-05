// Command hello 加载 .env 并打印当前 LLM 配置（API key 脱敏输出）。
package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/simonxluo/GamingWorld/internal/env"
)

func main() {
	// 加载本地 .env（含真实 key，已被 .gitignore 排除）。
	// 文件缺失时降级：改用 shell 已导出的环境变量，不阻断启动。
	if err := env.LoadEnv(".env"); err != nil {
		slog.Warn("未加载 .env，将使用系统环境变量", "err", err)
	}

	baseURL := os.Getenv("LLM_BASE_URL")
	model := os.Getenv("LLM_MODEL")
	apiKey := os.Getenv("LLM_API_KEY")

	fmt.Println("GamingWorld 阶段 0 — 配置加载")
	fmt.Println("LLM_BASE_URL:", orDefault(baseURL, "<未设置>"))
	fmt.Println("LLM_MODEL:    ", orDefault(model, "<未设置>"))
	fmt.Println("LLM_API_KEY:  ", maskKey(apiKey))
}

// orDefault 给空值一个可读占位。
func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// maskKey 对 API key 脱敏：仅露出首尾各 4 位，中间用 * 代替。
// 短 key 全部掩码，空值显式标注——真实 key 永不明文进日志。
func maskKey(key string) string {
	if key == "" {
		return "<empty>"
	}
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}
