package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/simonxluo/GamingWorld/runtime/state"
	"github.com/simonxluo/GamingWorld/runtime/tool"
)

// fakeLLM 实现 Completer，按预设脚本依次返回，完全不碰网络
// 它把"不可控的真实 LLM"换成"可控的脚本"，让被测的解析+循环逻辑可断言
type fakeLLM struct {
	replies []string // 预设返回序列
	calls   int      // 已调用次数
}

// Complete 返回脚本里的下一条
//   - calls >= len(replies)：脚本耗尽，返回 error
//     （防止循环拿不到回复而空转、把测试挂死）
//   - 否则：取 replies[calls]，calls++
func (f *fakeLLM) Complete(_ context.Context, _, _ string) (string, error) {
	if f.calls >= len(f.replies) {
		return "", fmt.Errorf("fakeLLM 脚本耗尽")
	}

	result := f.replies[f.calls]
	f.calls++
	return result, nil
}

// TestParseFinal：验证从 LLM 输出里抠出 Final Answer
func TestParseFinal(t *testing.T) {
	// 两个用例：
	//   1) "Thought: x\nFinal Answer: 391"  → 期望 ("391", true)
	//   2) "Thought: x\nAction: echo"        → 期望 ("", false)
	// 调 parseFinal(text)，用 t.Errorf 报告实际与期望不符

	// 测试Final是否可靠 1
	got, ok := parseFinal("Thought: x\nFinal Answer: 391")
	if got != "391" || !ok {
		t.Errorf("含 Final Answer: got(%q, %v), want(\"391\", true)", got, ok)
	}

	// 测试Final是否可靠 2
	got, ok = parseFinal("Thought: x\nAction: echo")
	if got != "" || ok {
		t.Errorf("含 Final Answer: got(%q, %v), want(\"\", false)", got, ok)
	}
}

// TestParseAction：验证抠出 Action + Action Input
func TestParseAction(t *testing.T) {
	// 构造 "Thought: x\nAction: echo\nAction Input: hello"
	// 调 parseAction → 期望 action=="echo", actionInput=="hello", ok==true
	// 再补一个"缺 Action Input"的用例，验证 ok 仍为 true 但 actionInput 为空
	ac, acInput, ok := parseAction("Thought: x\nAction: echo\nAction Input: hello")
	if ac != "echo" || acInput != "hello" || !ok {
		t.Errorf("对于Action echo调用: got(%q, %q, %v), want(\"echo\", \"hello\", true)", ac, acInput, ok)
	}

	ac, acInput, ok = parseAction("Thought: x\nAction: echo\n")
	if ac != "echo" || acInput != "" || !ok {
		t.Errorf("对于Action echo调用: got(%q, %q, %v), want(\"echo\", \"\", true)", ac, acInput, ok)
	}
}

type echoTool struct{}

func (echoTool) Name() string {
	return "echo"
}
func (echoTool) Description() string {
	return "原样返回 input"
}
func (echoTool) Run(_ context.Context, input string) (string, error) {
	return input, nil
}

// TestRunLoop：用 fakeLLM 驱动一次两步循环，断言 State 轨迹被正确记录
func TestRunLoop(t *testing.T) {
	// 装配：
	//   llm := &fakeLLM{replies: []string{
	//       "Thought: 我用工具\nAction: echo\nAction Input: 391",
	//       "Thought: 拿到结果\nFinal Answer: 391",
	//   }}
	//   ★ 需要一个 Tool：在本文件定义 echoTool（原样返回 input）
	//   tools := tool.NewRegistry(); tools.Register(echoTool{})
	//   a := Agent{LLM: llm, Tools: tools, System: "", MaxSteps: 5}
	//   s := state.New("test")
	//   answer, err := a.Run(context.Background(), "随便问", s)
	//
	// 断言（不满足用 t.Errorf）：
	//   - err == nil
	//   - answer == "391"
	//   - len(s.Steps) == 1   ← 第二步 Final 直接 return，不进 Steps（回顾 agent.go:31-34）
	//   - s.Final == "391"
	//   - s.Steps[0].Action == "echo" && s.Steps[0].Observation 含 "391"
	//
	// 提示：用到 tool / state，记得加 import
	llm := &fakeLLM{
		replies: []string{
			"Thought: 我用工具\nAction: echo\nAction Input: 391",
			"Thought: 拿到结果\nFinal Answer: 391",
		},
	}

	tools := tool.NewRegistry()
	tools.Register(echoTool{})

	a := Agent{
		LLM:      llm,
		Tools:    tools,
		System:   "",
		MaxSteps: 5,
	}

	s := state.New("test")
	answer, err := a.Run(context.Background(), "ask anything", s)

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	if answer != "391" {
		t.Errorf("Run() answer = %v", answer)
	}

	if len(s.Steps) != 1 {
		t.Errorf("Run() len(s.Steps) = %v", len(s.Steps))
	}

	if s.Final != "391" {
		t.Errorf("Run() s.Final = %v", s.Final)
	}
}
