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
)

const systemPrompt = `你是一个用 ReAct 范式解题的助手。每一步严格按以下格式输出，不要输出任何额外文字：

Thought: <一句话推理：我下一步该想/做什么>
Action: <工具名，必须从工具清单里选>
Action Input: <给工具的参数，强制单行，多参数用 key=value 空格分隔>

当你已经得到答案，用这个格式结束（不要再输出 Action）：
Thought: <推理>
Final Answer: <最终回答给用户的内容>

可用工具：
- calculate: 整数四则运算。Action Input 形如: a=23 op=* b=17
- echo: 原样返回输入，用于调试。Action Input: 任意文本

规则：Action Input 必须单行；禁止输出 JSON；未拿到结果前不要给 Final Answer。
`

func main() {
	if err := env.LoadEnv(".env"); err != nil {
		fmt.Fprintln(os.Stderr, "warn: 未加载 .env:", err)
	}

	c := &llm.Client{
		BaseUrl: os.Getenv("LLM_BASE_URL"),
		ApiKey:  os.Getenv("LLM_API_KEY"),
		Model:   os.Getenv("LLM_MODEL"),
	}

	fmt.Print("> ")
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return
	}

	answer, err := react(context.Background(), c, systemPrompt, sc.Text(), 8)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Println("\n===>", answer)
}

// react 执行ReAct 循环: 调 LLM -> 解析 -> 执行工具 -> 回填 Observation，直到拿到 Final Answer
func react(ctx context.Context, c *llm.Client, system, question string, maxSteps int) (string, error) {
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
		observation, err := execute(action, actionInput)
		if err != nil {
			observation = "工具调用执行失败：" + err.Error()
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

func execute(action, actionInput string) (string, error) {
	switch action {
	case "calculate":
		return calculate(actionInput)
	case "echo":
		return actionInput, nil
	default:
		return "", fmt.Errorf("未知工具: %s", action)
	}
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
