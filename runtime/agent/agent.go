package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/simonxluo/GamingWorld/runtime/tool"
)

type Completer interface {
	Complete(ctx context.Context, system, user string) (string, error)
}
type Agent struct {
	LLM      Completer
	Tools    *tool.Registry
	System   string
	MaxSteps int
}

func (a Agent) Run(ctx context.Context, input string) (string, error) {
	history := fmt.Sprintf("用户问题: %s \n", input)

	for step := 1; step < a.MaxSteps; step++ {
		fmt.Printf("\n---- step %d ----\n", step)
		output, err := a.LLM.Complete(ctx, a.System, history)
		if err != nil {
			return "", fmt.Errorf("agent stop %d 调用失败: %w", step, err)
		}
		fmt.Println(output)

		if ans, ok := parseFinal(output); ok {
			return ans, nil
		}
		action, actionInput, ok := parseAction(output)
		if !ok {
			history += output + "\nObservation: 格式错误, 未找到Action 或 Final Answer, 请严格按照格式输出\n\n"
			continue
		}

		t, ok := a.Tools.Get(action)
		observation := ""
		if !ok {
			observation = "未知工具: " + action
		} else {
			result, err := t.Run(ctx, actionInput)
			if err != nil {
				observation = fmt.Sprintf("工具 %q 执行失败: %v", action, err)
			} else {
				observation = result
			}
		}
		history += fmt.Sprintf("Action: %s\nAction Input: %s\nObservation: %s\n\n", action, actionInput, observation)
	}
	return "", nil
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
