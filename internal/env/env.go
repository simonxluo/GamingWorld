// Package env 提供 .env 文件加载：逐行解析 key=value 写入环境变量。
package env

import (
	"bufio"
	"os"
	"strings"
)

// LoadEnv 逐行读取 path（典型为 ".env"），把 key=value 写入进程环境变量。
//
// 解析规则：
//   - 跳过空行与以 '#' 开头的注释行；
//   - 按第一个 '=' 切分（值里允许再出现 '='，如 URL）；
//   - 去掉 key/value 两端空白，去掉值两端成对的单/双引号；
//   - 若环境变量已存在，则不覆盖——shell 里导出的值优先级更高。
//
// 文件不存在时返回错误，由调用方决定是否降级（如改用系统环境变量）。
func LoadEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = unquote(value)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value) // key 非空且来自合法行，此处不会失败
		}
	}
	return scanner.Err()
}

// unquote 去掉值两端成对的引号（"..." 或 '...'），裸值原样返回。
func unquote(v string) string {
	if len(v) >= 2 {
		first, last := v[0], v[len(v)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return v[1 : len(v)-1]
		}
	}
	return v
}
