package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/simonxluo/GamingWorld/runtime/state"
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

func (a Agent) Run(ctx context.Context, input string, s *state.State) (string, error) {
	s.Input = input

	for step := 1; step < a.MaxSteps; step++ {
		output, err := a.LLM.Complete(ctx, a.System, s.GetHistory())

		if err != nil {
			return "", err
		}

		if ans, ok := parseFinal(output); ok {
			s.Final = ans
			return ans, nil
		}

		action, actionInput, _ := parseAction(output)
		observation := ""

		t, ok := a.Tools.Get(action)
		if !ok {
			observation = fmt.Sprintf("未知工具:%s", action)
		}
		result, err := t.Run(ctx, actionInput)
		if err != nil {
			observation = fmt.Sprintf("工具:%s 调用失败", action)
		}

		observation = fmt.Sprintf("Action:%s, ActionInput:%s, Result:%s", action, actionInput, result)

		s.Append(state.Step{
			LLMOutput:   output,
			Action:      action,
			ActionInput: actionInput,
			Observation: observation,
		})

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
