package tool

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Tool 是一个可被 ReAct 循环发现的工具。三个方法分别提供：名字、说明、执行。
type Tool interface {
	Name() string
	Description() string
	Run(ctx context.Context, input string) (string, error)
}

// Registry 是工具的注册表，按名字索引。
type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register 注册一个工具；重名直接覆盖。
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// Get 按名字取工具，ok=false 表示不存在。
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok

}

// Spec 把所有工具的「名字 + 说明」拼成 prompt 里的工具清单（按名字排序，输出稳定）。
func (r *Registry) Spec() string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)

	var sb strings.Builder
	for _, name := range names {
		fmt.Fprintf(&sb, "- %s: %s\n", name, r.tools[name].Description())
	}
	return sb.String()
}
