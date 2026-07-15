package state

import (
	"fmt"
	"strings"
)

// Step 是 ReAct 循环的一步，一次 LLM输出 + 对应工具观测
type Step struct {
	Index       int    `json:"index"`
	LLMOutput   string `json:"llm_output"`       //模型本轮原始输出
	Action      string `json:"action,omitempty"` //解析出的工具名
	ActionInput string `json:"action_input,omitempty"`
	Observation string `json:"observation,omitempty"`
}

// State 是单个 agent 的私有轨迹，只存对话历史，不存共享信息
type State struct {
	AgentID string `json:"agent_id"`
	Input   string `json:"input,omitempty"`
	Steps   []Step `json:"steps"`
	Final   string `json:"final,omitempty"`
}

func New(agentID string) *State { return &State{AgentID: agentID} }

func (s *State) Append(step Step) {
	step.Index = len(s.Steps) + 1
	s.Steps = append(s.Steps, step)
}

func (s *State) GetHistory() string {
	var b strings.Builder
	fmt.Fprintf(&b, "用户问题: %s\n", s.Input)

	for _, st := range s.Steps {
		b.WriteString(st.LLMOutput)
		fmt.Fprintf(&b, "\nObservation: %s\n\n", st.Observation)
	}

	return b.String()
}
