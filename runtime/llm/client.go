package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client 配置，包括 LLM API 地址、模型、HTTP 客户端。
type Client struct {
	BaseUrl    string
	ApiKey     string
	Model      string
	HttpClient *http.Client
}

// 协议常量：Message API 要求显示提供版本号和路径
const (
	maxTokens        = 8192
	anthropicVersion = "2023-06-01"
	messagesPath     = "/v1/messages"
)

// 请求结构体 - anthropic格式
type messagesRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	System    string        `json:"system,omitempty"`
	Messages  []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// 构造llm api的response数据结构体
type messagesResponse struct {
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Complete 以 system + user 作为输入， 返回模型生成的文本
func (c *Client) Complete(ctx context.Context, system, user string) (string, error) {
	// TODO 1: 组请求体结构（定义 request 结构体 + json tag）
	request := messagesRequest{
		Model:     c.Model,
		MaxTokens: maxTokens,
		System:    system,
		Messages: []chatMessage{
			{
				Role:    "user",
				Content: user,
			},
		},
	}

	// TODO 2: json.Marshal → bytes.NewReader(body)
	body, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	// TODO 3: http.NewRequestWithContext(ctx, POST, BaseURL+"/v1/messages", body)
	//         设置 Header: Content-Type / anthropic-version: 2023-06-01 / Authorization: Bearer APIKey
	url := c.BaseUrl + messagesPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("content-type", "application/json")
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("x-api-key", c.ApiKey)

	// TODO 4: c.HTTP.Do(req)（注意 c.HTTP 为 nil 时用 http.DefaultClient）
	hc := c.HttpClient
	if hc == nil {
		hc = http.DefaultClient
	}

	resp, err := hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm: 调用端点失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("llm 读取响应失败: %w", err)
	}
	// TODO 5: 检查 resp.StatusCode != 200 → 读 body 返回错误
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm 端点返回 %d: %s", resp.StatusCode, string(respBody))
	}
	// TODO 6: 解析响应结构（content 是数组，取 [0].text），返回 text
	var parsed messagesResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("llm: 解析响应失败: %w", err)
	}
	var sb strings.Builder
	for _, blk := range parsed.Content {
		if blk.Type == "text" {
			sb.WriteString(blk.Text)
		}
	}

	return sb.String(), nil
}
