package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type CalculateTool struct{}
type EchoTool struct{}
type NowTool struct{}

// -- CalculateTool
func (CalculateTool) Name() string { return "calculate" }
func (CalculateTool) Description() string {
	return "整数四则运算- Action Input 形如: a=23 op=* b=17"
}
func (CalculateTool) Run(_ context.Context, input string) (string, error) {
	return calculate(input)
}

func calculate(actionInput string) (string, error) {
	kv := parseKV(actionInput)
	a, err1 := strconv.Atoi(kv["a"])
	b, err2 := strconv.Atoi(kv["b"])
	if err1 != nil || err2 != nil {
		return "", fmt.Errorf("a/b 必须是整数,收到 a=%q b=%q", kv["a"], kv["b"])
	}
	switch kv["op"] {
	case "+":
		return strconv.Itoa(a + b), nil
	case "-":
		return strconv.Itoa(a - b), nil
	case "*":
		return strconv.Itoa(a * b), nil
	case "/":
		if b == 0 {
			return "", fmt.Errorf("除以零")
		}
		return strconv.Itoa(a / b), nil
	default:
		return "", fmt.Errorf("未知运算符 %q（仅支持 + - * /）", kv["op"])
	}
}

// --

// -- EchoTool
func (EchoTool) Name() string { return "echo" }
func (EchoTool) Description() string {
	return "原样返回输入,用于调试,Action Input: 任意文本"
}
func (EchoTool) Run(_ context.Context, input string) (string, error) {
	return input, nil
}

// --

func (NowTool) Name() string { return "now" }
func (NowTool) Description() string {
	return "获取当前时间，取决于执行器的环境所使用的时区信息"
}
func (NowTool) Run(_ context.Context, input string) (string, error) {
	return time.Now().String(), nil
}

func parseKV(s string) map[string]string {
	m := map[string]string{}
	for _, f := range strings.Fields(s) {
		if k, v, ok := strings.Cut(f, "="); ok {
			m[k] = v
		}
	}

	return m
}
