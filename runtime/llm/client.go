package llm

import (
	"context"
	"net/http"
)

// Client 配置，包括 LLM API 地址、模型、HTTP 客户端。
type Client struct {
	ApiURL     string
	ApiKey     string
	Model      string
	HttpClient *http.Client
}

func (c *Client) Complete(ctx context.Context, systen, user string) (string, error) {
	// TODO 1: 组请求体结构（定义 request 结构体 + json tag）
	//   model, max_tokens, system, messages: [{role:"user", content: user}]
	// TODO 2: json.Marshal → bytes.NewReader(body)
	// TODO 3: http.NewRequestWithContext(ctx, POST, BaseURL+"/v1/messages", body)
	//         设置 Header: Content-Type / anthropic-version: 2023-06-01 / Authorization: Bearer APIKey
	// TODO 4: c.HTTP.Do(req)（注意 c.HTTP 为 nil 时用 http.DefaultClient）
	// TODO 5: 检查 resp.StatusCode != 200 → 读 body 返回错误
	// TODO 6: 解析响应结构（content 是数组，取 [0].text），返回 text

	return "", nil
}
