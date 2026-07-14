# 性能对比数据记录规范

记录"改造前后对比"的实测数据时，只写三段，不写叙事、不写过程流水、不重复结论。正文只放汇总，原始采样数据放附录或单独文件。

## 三段结构

### 1. 资源用量

一张表列出本次实际拉起的组件与数量，并给出总 simulator 进程数 N（内存共享倍数）：

| 组件 | 数量 |
|---|---|
| AgentServer |  |
| Learner |  |
| Actor (num_actors) |  |
| MainSimulatorWorker |  |
| TeamSimulatorWorker |  |
| RunHandlerActor |  |
| DS Client |  |

- N = AgentServer 数 × (1 + team_num)，是内存共享倍数；去重收益 = (N−1) × 单对象大小
- 不写"4 agent"这类含糊说法，写清 AgentServer 数 + 每 AgentServer 的 worker 拓扑 + 推出的 N

### 2. 固定对比条件

逐条列出保持不变的条件，单独标出本轮唯一变量：

- repo（优化组 / 基线）
- skill、地图（DS_MODE_ID）
- worker 数（N）、采样窗口、稳态取值区间
- 总闸值（enable_plasma_share / enable_cpp_mmap）、子开关默认值
- 唯一变量：如关掉 `mmap_heightmap`，其余固定

### 3. 指标与收益

一张表，每行一个指标，给出总量 Δ 与单进程摊销 Δ/N：

| 指标（口径） | 改造前 | 改造后 | Δ | Δ/N |
|---|---|---|---|---|

- 口径必须注明：RSS（进程不去重）/ PSS（去重）/ docker MEM（含 page cache，接近物理节省）
- 收益同时给总量 Δ 和摊销 Δ/N，缺一不可
- 内存 ≥1GB 报 GB（2 位小数），<1GB 报 MB；耗时报 ms

## 禁止

- 叙事段落（"我们发现…""这一轮…""值得注意的是…"）
- 过程流水（采样日志、命令输出大段粘贴进正文）
- 同一结论重复多次（小结里不再复述表格已有的数字）
- 把逐点原始采样放进正文

## 原始数据去向

- 逐点采样 → `bench_results_vN/raw_<ts>/<label>_mem_samples.tsv`
- `[BENCH]` 事件 → 同目录 `<label>_bench.log`
- 正文 markdown 只引用汇总值，不内嵌原始数据；需要溯源时指明 raw 文件路径

## 正例（精简三段）

```
## HeightMap mmap 收益（N=3）

### 1. 资源用量
| 组件 | 数量 |
|---|---|
| AgentServer | 1 |
| MainSimulatorWorker | 1 |
| TeamSimulatorWorker | 2 |
| DS Client | 1 |
N = 1 × (1+2) = 3

### 2. 固定条件
repo=优化组, skill=fight, DS_MODE_ID=43, N=3, 窗口 480s 稳态 300–480s
总闸 enable_cpp_mmap=true; 唯一变量: mmap_heightmap 开 vs 关

### 3. 指标与收益
| 指标（口径） | 前(关) | 后(开) | Δ | Δ/N |
|---|---|---|---|---|
| rss_sum (RSS) | 9.07 GB | 7.02 GB | -2.05 GB | -0.68 GB |
| docker MEM | 18.4 GiB | 15.2 GiB | -3.2 GiB | -1.07 GiB |
```

## 反例（冗余）

```
这一轮我们重点验证了 HeightMap 的 mmap 收益。之前的测试因为总闸没开，
Δ 只有 3MB，属于噪声。这一轮我们打开了 enable_cpp_mmap=true，
然后跑了 00 和 05 两个 case，每个跑 8 分钟，前 90 秒等就绪……
（后续 5 段叙事 + 粘贴 20 行采样日志 + 三处重复 2.05GB 这个数字）
```
