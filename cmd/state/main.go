package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/simonxluo/GamingWorld/internal/env"
	"github.com/simonxluo/GamingWorld/runtime/agent"
	"github.com/simonxluo/GamingWorld/runtime/llm"
	"github.com/simonxluo/GamingWorld/runtime/memory"
	"github.com/simonxluo/GamingWorld/runtime/tool"
)

// systemPrompt：ReAct 协议说明，与 cmd/react 完全一致
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
	// ── (1) flag：解析 -id，决定从哪个 agent 的历史恢复
	//   id := flag.String("id", "default", "agent 标识，同一 id 共享历史")
	//   flag.Parse()
	//   ⚠ id 是 *string，传给 Load/Save 时要 *id 解引用
	id := flag.String("id", "default", "agent 标识， 同一id 共享历史")
	flag.Parse()

	// ── (2) 装配（与 cmd/react 一致，照搬即可）
	//   env.LoadEnv(".env")              // err 仅 warn，不退出
	//   &llm.Client{BaseUrl/ApiKey/Model 从 os.Getenv 读}
	//   tool.NewRegistry() + Register(CalculateTool{}, EchoTool{}, NowTool{})
	//   &agent.Agent{LLM, Tools, System: fmt.Sprintf(systemPrompt, tools.Spec()), MaxSteps: 8}
	if err := env.LoadEnv(".env"); err != nil {
		fmt.Fprintln(os.Stderr, "warn：未加载 .env:", err)
	}

	c := &llm.Client{
		BaseUrl: os.Getenv("LLM_BASE_URL"),
		ApiKey:  os.Getenv("LLM_API_KEY"),
		Model:   os.Getenv("LLM_MODEL"),
	}

	t := tool.NewRegistry()
	t.Register(CalculateTool{})
	t.Register(EchoTool{})
	t.Register(NowTool{})

	a := &agent.Agent{
		LLM:      c,
		Tools:    t,
		System:   fmt.Sprintf(systemPrompt, t.Spec()),
		MaxSteps: 8,
	}
	// ── (3) memory + ctx
	//   mem := &memory.File{Dir: ".state"}   // 用 File 才能跨进程演示恢复
	//   ctx := context.Background()
	mem := &memory.File{Dir: ".state"}
	ctx := context.Background()

	// ── (4) 循环外 Load 一次：拿到持久化的 *state.State
	//   s, err := mem.Load(ctx, *id)
	//   err != nil → 打印 stderr + os.Exit(1)
	//   打印恢复信息：*id、len(s.Steps)（首次运行为 0）
	//   ⚠ 整个会话共用这一个 s，循环里不再 Load（否则会覆盖本轮刚追加的 Steps）
	s, err := mem.Load(ctx, *id)
	if err != nil {
		fmt.Printf("err:%v", err)
	}

	// ── (5) 交互循环
	//   sc := bufio.NewScanner(os.Stdin)
	//   fmt.Print("> ")
	//   for sc.Scan() {
	//       input := strings.TrimSpace(sc.Text())
	//       空行 → continue（continue 前重打 "> "，否则光标没提示）
	//       answer, err := a.Run(ctx, input, s)
	//       err != nil → 打印 stderr + continue（别退出，让用户换问法重试）
	//       fmt.Println("\n===>", answer)
	//       ★ mem.Save(ctx, *id, s)    // 每轮落盘！Ctrl+C 直接杀进程，defer 的 Save 跑不到
	//       fmt.Print("> ")
	//   }
	//   循环结束（EOF）后可查 sc.Err()
	sc := bufio.NewScanner(os.Stdin)
	fmt.Printf("> ")
	for sc.Scan() {
		input := strings.TrimSpace(sc.Text())
		answer, err := a.Run(ctx, input, s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "stdin 读取错误: %v\n", err)
		}
		fmt.Println("\n===>", answer)
		mem.Save(ctx, *id, s)
		fmt.Print("> ")
	}
}
