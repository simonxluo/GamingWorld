# CLAUDE.md — GamingWorld Go Agent 框架构建指南

本文件是本项目的"宪法"与教学路线图。它同时服务于：
- **人类**（项目所有者）：理解架构、按阶段推进、把控设计方向；
- **AI 协作者（Claude）**：在本仓库工作时的首要参考，遵循其中的约定与阶段顺序。

---

## 1. 项目愿景

用 **Go** 从零构建一个 agent 框架，最终服务于 **gaming 场景**：agent 驱动的游戏世界（NPC、世界状态、多 agent 感知-行动交互）。

但起点极其朴素：**一个能跑通的最小 LLM ReAct 循环**。

### 核心方法论（务必内化）

> **从最小可运行的循环开始，每一步都可运行、可验证，逐步抽象出框架。**
> 不要一开始就搭大而全的 `runtime/`。先在单文件里手写循环，等重复出现的需求暴露出抽象点，再把它抽进 `runtime/`。
> **过早抽象是本框架最大的敌人。**

---

## 2. 技术栈与约定

| 项 | 约定 |
|---|---|
| 语言 | Go 1.22+（用 `range-over-int`、`slices`、`maps` 标准库） |
| 模块路径 | `github.com/simonxluo/GamingWorld` |
| 配置来源 | `.env`（`LLM_BASE_URL` / `LLM_API_KEY` / `LLM_MODEL`） |
| LLM 端点 | `http://api.simonluo.site`（**Anthropic Messages API 兼容**），模型 `glm-5.2`；备用 DeepSeek `https://api.deepseek.com/anthropic` |
| 包命名 | **全小写**（`runtime/agent`，不是 `runtime/Agent`） |
| 错误处理 | 显式 `error` 返回，**不 panic**，不滥用 `panic/recover` |
| 日志 | 标准库 `log/slog` |
| 依赖原则 | **能不引第三方就不引**。`.env` 解析、HTTP 调用都先手写，教学价值优先 |

### .env 的加载

阶段 0 手写一个 ~20 行的 `LoadEnv()`（逐行 `key=value` 解析 + `os.Setenv`），避免引入 `godotenv`。理解 `os.Getenv` 比引入依赖更重要。后期若需要更健壮的解析再换。

---

## 3. 架构全景（最终形态，先看清要去哪）

两层分离：

```
┌─────────────────────────────────────────────────────────┐
│  agents/          业务层：具体 agent 实现（NPC、玩家代理）  │
│      npc.go  player.go  ...                              │
└────────────────────────┬────────────────────────────────┘
                         │ 装配 + 配置
┌────────────────────────▼────────────────────────────────┐
│  runtime/         框架内核：与业务无关的执行引擎           │
│                                                          │
│   llm/        LLM 客户端封装（Messages API）              │
│   tool/       Tool 接口 + Registry                        │
│   agent/      ReAct 循环引擎（核心）                      │
│   state/      一次运行的轨迹 / 世界状态                    │
│   memory/     短期(进程内) + 长期(文件/kv) 记忆            │
│   event/      可观测性 / 流式输出                          │
│   checkpoint/ 状态序列化、中断恢复                         │
│   scheduler/  多 agent 调度、并发、事件驱动                │
└─────────────────────────────────────────────────────────┘
```

**关键认知**：`runtime/` 不是一开始就有的。它是阶段 3 之后，从 `main.go` 里**一步步抽出来的**。下面的路线图会演示这个抽取过程。

---

## 4. 核心概念：ReAct 循环的本质

ReAct = **Reason + Act**。抛开一切框架，它就是一个 while 循环：

```
历史 = [系统提示, 用户问题]
循环 (最多 N 步):
    1. 把 [系统提示 + 历史 + 工具说明] 拼成 prompt，发给 LLM
    2. LLM 返回文本，解析出：
         Thought:      我下一步该想/做什么（推理）
         Action:       选哪个工具
         Action Input: 给工具的参数
       （若解析出 Final Answer，则跳出循环返回）
    3. 执行 Action(Action Input) → 得到 Observation
    4. 把 [LLM 输出 + Observation] 追加进历史
    5. 回到 1
```

**本项目主线采用"纯 prompt ReAct"**：不依赖 API 的 function calling，而是用提示词约定模型输出 `Thought / Action / Action Input` 文本，我们自己在 Go 里解析并执行。理由：
1. **最透明**——循环的每一步都暴露在 prompt 和代码里，最适合学习；
2. **端点无关**——不依赖 `api.simonluo.site` 是否透传 tool_use，任何能"文本进文本出"的模型都能跑；
3. **打下抽象地基**——之后切到原生 tool use（阶段 7）只是换一种"解析器"，循环骨架不变。

### 提示词协议（LLM 约定输出的格式）

```
Thought: <一句话推理>
Action: <工具名>
Action Input: <参数，单行或 JSON>
```
执行后，我们的代码追加：
```
Observation: <工具执行结果>
```
当模型想结束，它输出：
```
Thought: <推理>
Final Answer: <最终回答>
```

---

## 5. 渐进式实现路线图（核心）

每个阶段都产出**一个可运行的 `cmd/<name>/main.go`**。完成一个阶段、跑通验收，再进下一个。

### ▶ 阶段 0：项目骨架
**目标**：能编译、能读配置、能打印。
- `go mod init github.com/simonxluo/GamingWorld`
- `internal/env/env.go`：`LoadEnv(path string) error`（手写 `.env` 解析）
- `cmd/hello/main.go`：加载 `.env`，打印 `LLM_BASE_URL` / `LLM_MODEL`
- **验收**：`go run ./cmd/hello` 正确打印配置；不读到真实 key 到日志明文（脱敏打印）。

### ▶ 阶段 1：最小 LLM 调用（打通端点）
**目标**：发一条消息，拿到模型文本回复。
- `runtime/llm/client.go`：
  ```go
  type Client struct { BaseURL, APIKey, Model string; HTTP *http.Client }
  func (c *Client) Complete(ctx context.Context, system, user string) (string, error)
  ```
- 内部 POST `{BaseURL}/v1/messages`（Anthropic Messages 格式），解析 `content[0].text`。
- `cmd/chat/main.go`：从 stdin 读一行，打印模型回复。
- **验收**：`go run ./cmd/chat`，输入"你好"，拿到 glm-5.2 的中文回复。
- **踩坑预警**：若端点是 OpenAI 兼容而非 Anthropic，改打 `/v1/chat/completions` 并换 schema；先 `curl` 确认端点形态。

### ▶ 阶段 2：最小 ReAct 循环 ⭐（里程碑）
**目标**：**单文件**手写一个完整的 ReAct 循环，能调用工具回答问题。
- `cmd/react/main.go`：在一个文件里完成「拼 prompt → 调 LLM → 解析 → 执行工具 → 回填 Observation → 终止判断」。
- 硬编码 1~2 个工具，例如：
  - `calculate`：用 Go 算简单算式（`eval` 自实现或限定四则运算）
  - `echo`：原样返回（用于调试）
- prompt 里写明工具清单与输出协议（见第 4 节）。
- **验收**：问 `23 乘以 17 是多少`，终端能看到 `Thought → Action → Observation → Final Answer: 391` 的完整轨迹。
- **这一步先丑着**：解析用 `strings.HasPrefix`、工具用 `switch` 都行。重点是跑通循环，不是优雅。

### ▶ 阶段 3：抽象出 Tool 系统 → `runtime/tool`
**暴露的抽象点**：阶段 2 里"加工具要改 switch"很烦 → 把工具变成接口。
- `runtime/tool/tool.go`：
  ```go
  type Tool interface {
      Name() string
      Description() string
      Run(ctx context.Context, input string) (string, error)
  }
  type Registry struct{ ... }
  func (r *Registry) Register(t Tool)
  func (r *Registry) Get(name string) (Tool, bool)
  func (r *Registry) Spec() string  // 自动拼成 prompt 里的工具说明
  ```
- 把 `calculate`、`echo` 改造成实现 `Tool` 的结构体。
- **验收**：新增一个工具 `now`（返回当前时间），**不改动循环代码**，只 `Register` 即可用。

### ▶ 阶段 4：抽象出循环 → `runtime/agent`
**暴露的抽象点**：阶段 2/3 的循环逻辑夹在 `main` 里 → 抽成可复用引擎。
- `runtime/agent/agent.go`：
  ```go
  type Agent struct {
      LLM       *llm.Client
      Tools     *tool.Registry
      System    string
      MaxSteps  int
  }
  func (a *Agent) Run(ctx context.Context, input string) (answer string, err error)
  ```
- `cmd/react/main.go` 缩成 ~10 行：装配 → `agent.Run`。
- **验收**：换一个完全不同的问题域（如"查时间+算数"），`main` 不动，只换 `System` 和注册的工具。

### ▶ 阶段 5：State + Memory → `runtime/state`、`runtime/memory`
- `runtime/state`：把"历史/轨迹"显式建模为 `State`（steps 列表），不再藏在循环局部变量里。
- `runtime/memory`：抽象存储接口 `Memory { Load / Save }`；先实现进程内 map（短期），再加文件实现（长期）。
- **验收**：agent 处理完一个输入后，`State` 可序列化；下次 `Load` 能恢复上下文继续对话。

### ▶ 阶段 6：Event 系统 → `runtime/event`
**暴露的抽象点**：循环里到处 `slog.Info` 打日志，想加流式输出就得改循环 → 用事件解耦。
- `runtime/event`：定义 `Event`（Thought/Action/Observation/Final/Error），`Agent` 持有 `[]Handler`，每步 `emit`。
- **验收**：写两个 handler——一个打日志、一个流式打印到 stdout；循环代码只负责 `emit`，不知道谁在听。

### ▶ 阶段 7：原生 Tool Use（可选对比）
- 在 `runtime/llm` 增加基于 Anthropic `tools` 参数的调用路径，对比 prompt ReAct。
- **目标**：理解两种范式取舍，确认我们的抽象（`Tool` 接口）对两种解析器都成立。

### ▶ 阶段 8：Checkpoint → `runtime/checkpoint`
- 基于 `state`：序列化、存盘、`Restore`。
- **验收**：循环跑到一半被中断（模拟 crash），重启后从上次 `State` 续跑。

### ▶ 阶段 9：Scheduler → `runtime/scheduler`
- 多 agent、并发调度、事件驱动 tick。为 gaming 多 NPC 世界铺路。
- **验收**：两个 agent 在一个共享 `state`（世界）里轮流行动。

### ▶ 阶段 10：Gaming 特化 → `agents/`
- 世界状态、感知-行动循环、NPC persona、玩家代理。
- **里程碑**：一个 NPC 能根据世界状态 + 玩家输入，用 ReAct 决策并调用"移动/说话/拾取"等工具。

---

## 6. 设计原则（贯穿所有阶段）

1. **先能跑，再抽象** —— 阶段 2 必须先丑着跑通，才有资格在阶段 3+ 抽象。
2. **每阶段一个可运行入口** —— `cmd/<name>/main.go` 是阶段的"验收证明"。
3. **接口小而稳，实现可替换** —— `Tool`、`Memory`、`Memory Handler` 都是窄接口。
4. **测试用 fake LLM** —— 单测不依赖真实 API：实现一个返回预设文本的 `FakeLLM`，让 ReAct 解析/循环逻辑可确定性测试。
5. **真实 key 永不入库** —— `.env` 已被 `.gitignore`；代码里 key 只从环境读，不硬编码、不全量打日志。

---

## 7. 目录结构（目标）

```
GamingWorld/
├── cmd/
│   ├── hello/      # 阶段 0
│   ├── chat/       # 阶段 1
│   └── react/      # 阶段 2+（逐步瘦身）
├── runtime/
│   ├── llm/        # 阶段 1
│   ├── tool/       # 阶段 3
│   ├── agent/      # 阶段 4
│   ├── state/      # 阶段 5
│   ├── memory/     # 阶段 5
│   ├── event/      # 阶段 6
│   ├── checkpoint/ # 阶段 8
│   └── scheduler/  # 阶段 9
├── agents/         # 阶段 10：业务 agent
├── internal/
│   └── env/        # 阶段 0：.env 加载
├── .env            # 本地真实配置（不入库）
├── .env.example    # 配置模板（入库）
└── CLAUDE.md       # 本文件
```

> 注：现有 `runtime/` 下的大写目录（`Agent/` 等）将统一改为小写，以符合 Go 包命名惯例。

---

## 8. 常用命令

```bash
go mod init github.com/simonxluo/GamingWorld   # 阶段 0 执行一次
go run ./cmd/hello                              # 各阶段验收
go run ./cmd/chat
go run ./cmd/react
go test ./...                                   # 全量测试
go build ./...                                  # 全量编译检查
```

读取 `.env`：进程启动时调用 `internal/env.LoadEnv(".env")`，之后用 `os.Getenv("LLM_BASE_URL")` 等。

---

## 9. 当前进度

- [x] 配置基础设施：`.env` / `.env.example` / `.gitignore`（已提交，真实 key 仅在本地）
- [x] 本构建指南 `CLAUDE.md`
- [ ] **阶段 0：项目骨架** ← 下一步
- [ ] 阶段 1 ~ 10：见路线图

**下一步行动**：执行阶段 0——`go mod init`、写 `internal/env`、写 `cmd/hello`，跑通配置加载。
