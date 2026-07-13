package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/simonxluo/GamingWorld/internal/env"
	"github.com/simonxluo/GamingWorld/runtime/agent"
	"github.com/simonxluo/GamingWorld/runtime/llm"
	"github.com/simonxluo/GamingWorld/runtime/tool"
)

const systemPrompt = `你是一个用 ReAct 范式解题的助手。 每一步按以下格式输出:
Thought: <一句话推理,下一步做什么>
Action:<工具名,必须是下列可用工具之一>
Action Input:<工具参数,强制单行,多参数用key=value 空格分隔,例如a=32 op=* b=13>

拿到 Observation 后继续循环。当你得出最终答案是,输出:
Thought:<推理>
Final Answer:<最终回答>

可用工具:
%s

注意: Action Input 必须单行, 不要换行, 不要使用Json
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

	tools := tool.NewRegistry()
	tools.Register(CalculateTool{})
	tools.Register(EchoTool{})
	tools.Register(NowTool{})

	system := fmt.Sprintf(systemPrompt, tools.Spec())

	agent := &agent.Agent{
		LLM:      c,
		Tools:    tools,
		System:   system,
		MaxSteps: 8,
	}

	fmt.Print("> ")
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return
	}

	answer, err := agent.Run(context.Background(), sc.Text())
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Println("\n===>", answer)
}
