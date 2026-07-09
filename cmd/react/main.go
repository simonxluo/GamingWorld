package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/simonxluo/GamingWorld/internal/env"
	"github.com/simonxluo/GamingWorld/runtime/llm"
	"github.com/simonxluo/GamingWorld/runtime/tool"
)

const systemPrompt = `你是一个用 ReAct 范式解题的助手。
可用工具:
%s

规则：...
`

type CalculateTool struct{}

func (CalculateTool) Name() string { return "calculate" }
func (CalculateTool) Description() string {
	return "整数四则运- Action Input 形如: a= 23 op=* b=17"
}
func (CalculateTool) Run(_ context.Context, input string) (string, error) {
	return calculate(input)
}

type EchoTool struct{}

func (EchoTool) Name() string { return "calculate" }
func (EchoTool) Description() string {
	return "原样返回输入，用于调试，Action Input: 任意文本"
}
func (EchoTool) Run(_ context.Context, input string) (string, error) {
	return input, nil
}

func main() {
	if err := env.LoadEnv(".env"); err != nil {
		fmt.Fprintln(os.Stderr, "warn: 未加载 .env:", err)
	}

	c := &llm.Client{
		BaseUrl: os.Getenv("LLM_BASE_URL"),
		ApiKey:  os.Getenv("LLM_API_KEY"),
		Model:   os.Getenv("LLM_MODEL"),
	}

	tools := tool.NewRegistry()
	tools.Register(CalculateTool{})
	tools.Register(EchoTool{})

	system := fmt.Sprintf(systemPrompt, tools.Spec())

	fmt.Print("> ")
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return
	}

	answer, err := react(context.Background(), c, system, sc.Text(), 8, tools)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Println("\n===>", answer)
}

// react 执行ReAct 循环: 调 LLM -> 解析 -> 执行工具 -> 回填 Observation，直到拿到 Final Answer
func react(ctx context.Context, c *llm.Client, system, question string, maxSteps int, tools *tool.Registry) (string, error) {
	history := "用户问题： " + question + "\n"
	for step := 1; step <= maxSteps; step++ {
		fmt.Printf("\n---- step %d ----\n", step)

		output, err := c.Complete(ctx, system, history)
		if err != nil {
			return "", fmt.Errorf("react step %d 调用失败: %w", step, err)
		}
		fmt.Println(output)

		if ans, ok := parseFinal(output); ok {
			return ans, nil
		}
		action, actionInput, ok := parseAction(output)

		if !ok {
			history += output + "\nObservation: 格式错误：未找到 Action 或 Final Answer, 请严格按照格式输 \n\n"
			continue
		}
		t, ok := tools.Get(action)
		observation := ""
		if !ok {
			observation = "未知工具: " + action
		} else {
			result, err := t.Run(ctx, actionInput)
			if err != nil {
				observation = "工具执行失败: " + err.Error()
			} else {
				observation = result
			}
		}

		history += output + "\nObservation: " + observation + "\n\n"
	}
	return "", fmt.Errorf("react: 达到最大步数 %d 仍未结束", maxSteps)
}

func field(output, prefix string) (string, bool) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix)), true
		}
	}
	return "", false
}

func parseFinal(output string) (string, bool) {
	return field(output, "Final Answer:")
}

func parseAction(output string) (action, actionInput string, ok bool) {
	action, ok = field(output, "Action:")
	if !ok {
		return "", "", false
	}

	actionInput, _ = field(output, "Action Input:")
	return action, actionInput, true
}

func calculate(actionInput string) (string, error) {
	kv := parseKV(actionInput)
	a, err1 := strconv.Atoi(kv["a"])
	b, err2 := strconv.Atoi(kv["b"])
	if err1 != nil || err2 != nil {
		return "", fmt.Errorf("a/b 必须是整数，收到 a=%q b=%q", kv["a"], kv["b"])
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

func parseKV(s string) map[string]string {
	m := map[string]string{}
	for _, f := range strings.Fields(s) {
		if k, v, ok := strings.Cut(f, "="); ok {
			m[k] = v
		}
	}

	return m
}
