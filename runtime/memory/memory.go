package memory

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/simonxluo/GamingWorld/runtime/state"
)

// Memory 按 agent 维度存取跨运行的 State
type Memory interface {
	// Load 不存在时返回空 State，不是 error
	Load(ctx context.Context, agentID string) (*state.State, error)
	Save(ctx context.Context, agentID string, s *state.State) error
}

// InMemory 进程内存储，重启即丢失；多 agent 可能并发读写同一实例，需自身保证安全
type InMemory struct {
	//锁和按 agentID 索引的存储
	mu   sync.RWMutex
	data map[string]*state.State
}

func NewInMemory() *InMemory {
	return &InMemory{
		data: make(map[string]*state.State),
	}
}

// Load 在读保护下查找；命中返回该 State，未命中返回 state.New(agentID)
func (m *InMemory) Load(_ context.Context, agentID string) (*state.State, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.data[agentID]; ok {
		return s, nil
	}

	return state.New(agentID), nil
}

// Save 在写保护下写入
func (m *InMemory) Save(_ context.Context, agentID string, s *state.State) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[agentID] = s
	return nil
}

// File 跨进程持久：每个 agent 一个 <Dir>/<agentID>.json
type File struct {
	Dir string
}

// Load 读取并反序列化 <Dir>/<agentID>.json
// 文件不存在视为首次运行，返回空 State（不是 error）；其他读失败才返回错误
func (f *File) Load(_ context.Context, agentID string) (*state.State, error) {
	data, err := os.ReadFile(f.path(agentID))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return state.New(agentID), nil
		}
		return nil, err
	}
	var s state.State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	s.AgentID = agentID
	return &s, nil
}

// Save 序列化 State 后写入 <Dir>/<agentID>.json；目录不存在则创建
func (f *File) Save(_ context.Context, agentID string, s *state.State) error {
	if err := os.MkdirAll(f.Dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	return os.WriteFile(f.path(agentID), data, 0o644)
}

func (f *File) path(agentID string) string {
	return filepath.Join(f.Dir, agentID+".json")
}
