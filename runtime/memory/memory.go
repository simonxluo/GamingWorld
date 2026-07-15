package memory

import (
	"context"

	"github.com/simonxluo/GamingWorld/runtime/state"
)

type Memory interface {
	// Load 不存在时返回空 State(不是error )
	Load(ctx context.Context, agentID string) (*state.State, error)
	Save(ctx context.Context, agentID string, s *state.State) error
}

// InMemory:进程内 map,sync.RwMutex 保护并发，重启及丢失
type InMemory struct {
	/* mu + map[string]*state.State*/
}

// File:每个 agent 一个<agentID>.json，放在指定目录，跨进程持久
type File struct {
	Dir string
}
