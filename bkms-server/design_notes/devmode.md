# DevMode 开发模式组件

## Todo List
1. tafBuilder / trpcBuilder 分离，避免方法内部 if else 结构。
2. awk 这种方式在用户启动命令较长导致偏移/含空格的情况下，推荐使用/proc/pid/cmdline拿会更加准确。
3. trpc、taf 忽略 app 的中划线，下划线（app name，bin name）；init.sh匹配时打印日志，告诉用户使用的bin文件是哪个，如果报错也要明确找不到的原因。

## 概述

DevMode 是 BKMS 提供的容器内热更新调试功能，允许开发者在容器运行时动态替换二进制文件并重启服务，无需重新构建和部署镜像。

支持两种应用类型：**trpc** 和 **taf**，两者使用不同的工作目录和脚本集，但共享相同的组件架构和热更新流程。

## 环境限制

| 环境 | 是否允许 |
|------|---------|
| development | ✅ |
| test | ✅ |
| production | ❌ 严禁使用 |

## 应用类型差异

| 特性 | trpc | taf |
|------|------|-----|
| 工作根目录 | `/data/bkms/dev-mode/trpc` | `/data/bkms/dev-mode/taf` |
| 脚本挂载路径 | `/data/bkms/dev-mode/trpc/configmap-scripts` | `/data/bkms/dev-mode/taf/configmap-scripts` |
| 二进制路径 | 通过 `TrpcBinaryPath` 配置（默认 `/usr/local/trpc/bin`） | 运行时从进程信息动态确定 |
| 模板变量 | `BKMS_TRPC_BIN_PATH`、`BKMS_CUSTOM_START_SCRIPT` | 无模板变量 |
| 额外脚本 | 无 | `taf-start.sh`（wrap 方式拉起 taf 进程） |
| 进程定位方式 | 通过 bin 路径搜索列表查找 | 通过 `SERVER_NAME_ENV` 在进程列表中匹配 |

## 二进制文件命名规范

> **重要**：app 二进制文件名称必须与 BKMS 上的 app name 保持一致（不区分大小写）。

- **推荐**：使用英文字母 + 数字命名，如 `appserver`、`webserver2`
- **避免**：使用中杠（`-`）、下划线（`_`）等符号，如 ~~`app-server`~~、~~`web_server`~~

原因：脚本通过 `BKMS_APP_NAME` 环境变量在文件系统和进程列表中进行不区分大小写的模糊匹配来定位二进制文件，特殊符号可能导致匹配异常。

## 工作原理

1. 将管理脚本（init / start / stop / monitor / restart / utils）打包为 ConfigMap（`{appName}-devmode-scripts`）
2. 通过 Volume + VolumeMount 挂载到容器的对应路径（根据应用类型选择 trpc 或 taf 目录）
3. 替换容器启动命令为 `init.sh`
4. `init.sh` 检测 OS、安装依赖、将脚本复制到可写目录后启动 `monitor.sh` 守护进程
5. 用户通过 `bkms-cli` 上传新二进制到 `bin/` 目录，通过 `restart.sh` 完成热更新

## 容器内目录结构

### trpc 类型

```
/data/bkms/dev-mode/trpc/
├── bin/                    # 上传的可执行文件存放目录
├── configmap-scripts/      # ConfigMap 挂载（只读）
│   ├── init.sh             # 容器入口点：环境初始化 + 启动 monitor
│   ├── utils.sh            # 公共变量和函数（日志、进程管理、停止标志等）
│   ├── start.sh            # 启动业务进程（单例锁 + PID 记录）
│   ├── stop.sh             # 优雅停止（SIGTERM → SIGKILL）
│   ├── monitor.sh          # 守护进程：进程存活检查 + panic 检测 + 信号转发
│   └── restart.sh          # 热更新：MD5 校验 → 替换二进制 → SIGUSR2 重启
├── scripts/                # 脚本可写副本（由 init.sh 复制）
│   └── pid.conf            # 当前业务进程 PID
└── logs/                   # 各脚本运行日志
```

### taf 类型

```
/data/bkms/dev-mode/taf/
├── bin/                    # 上传的可执行文件存放目录
├── configmap-scripts/      # ConfigMap 挂载（只读）
│   ├── init.sh             # 容器入口点：环境初始化 + 启动 monitor
│   ├── utils.sh            # 公共变量和函数（日志、进程管理、停止标志等）
│   ├── start.sh            # 启动业务进程（通过 taf-start.sh wrap 拉起）
│   ├── stop.sh             # 优雅停止（SIGTERM → SIGKILL）
│   ├── monitor.sh          # 守护进程：进程存活检查 + 信号转发
│   ├── restart.sh          # 热更新：MD5 校验 → 替换二进制 → SIGUSR2 重启
│   └── taf-start.sh        # taf 专用：wrap 方式执行用户启动命令
├── scripts/                # 脚本可写副本（由 init.sh 复制）
│   └── pid.conf            # 当前业务进程 PID
└── logs/                   # 各脚本运行日志
```

## Go 代码结构

| 文件 | 职责 |
|------|------|
| `devmode.go` | 接口定义（`DevMode`）、配置结构体（`Config`）、输出结构体（`Output`）、工厂函数、`IsTrpc()` / `IsTaf()` 类型判断 |
| `builder.go` | `DevMode` 接口实现：校验、根据应用类型构建 ConfigMap / Volume / VolumeMount / Command |
| `apply.go` | `PatchGameDeployment`：通过 Strategic Merge Patch 将开发模式注入 GameDeployment |
| `constants.go` | 环境类型、资源名称、trpc / taf 路径常量、ConfigMap key 等常量 |
| `scripts.go` | 通过 `//go:embed` 嵌入 `assets/trpc/` 和 `assets/taf/` 下的 shell 脚本 |
| `errors.go` | 全局 sentinel 错误定义，外部可通过 `errors.Is` 判断具体错误类型 |

### 测试文件

| 文件 | 职责 |
|------|------|
| `devmode_trpc_test.go` | trpc 类型的 DevMode 组件单元测试 |
| `devmode_taf_test.go` | taf 类型的 DevMode 组件单元测试 |
| `apply_trpc_test.go` | trpc 类型的 PatchGameDeployment 集成测试 |
| `apply_taf_test.go` | taf 类型的 PatchGameDeployment 集成测试 |

## 核心流程

```
CreateDevModeConfig(appModel, envType, enabled)
        │
        ▼
    New(config)  →  根据 AppType 设置 WorkPath / MountPath  →  builder
        │
        ▼
  builder.Build()
   ├── Validate()        校验配置（taf 不校验 TrpcBinaryPath）
   ├── BuildConfigMap()   根据类型选择脚本集 → 渲染模板 → ConfigMap
   │                      （taf 额外生成 taf-start.sh）
   ├── BuildVolume()      ConfigMap → Volume（0755）
   ├── BuildVolumeMount() Volume → 挂载路径（trpc/taf 路径不同）
   └── BuildCommand()     [init.sh 完整路径]
        │
        ▼
PatchGameDeployment(gd, config)
   → Strategic Merge Patch 注入 Volume / VolumeMount / Command
   → 返回修改后的 GameDeployment + ConfigMap 额外资源
```

## 脚本模板变量

### trpc 类型

脚本中通过 Go `text/template` 渲染以下变量：

| 变量 | 来源 |
|------|------|
| `BKMS_TRPC_BIN_PATH` | `Config.TrpcBinaryPath` |
| `BKMS_CUSTOM_START_SCRIPT` | `Config.StartupCommand` |

### taf 类型

taf 脚本不使用模板变量。`taf-start.sh` 由 `Config.StartupCommand` 直接生成，格式为 `exec <StartupCommand>`。

### 运行时环境变量

`BKMS_APP_NAME` 由 Kubernetes 注入，用于定位二进制文件名（两种类型通用）。

## 热更新流程（restart.sh）

### trpc 类型

```
bkms-cli 上传二进制 → bin/<random_name>
        │
        ▼
restart.sh <random_name> <md5>
  1. MD5 校验上传文件
  2. 创建停止标志（暂停 monitor 自动拉起）
  3. 通过 TRPC_BIN_SEARCH_PATHS 查找当前二进制路径
  4. 删除旧二进制，移动新文件到 TRPC_BIN_PATH
  5. 发送 SIGUSR2 给当前进程触发优雅重启
  6. 等待旧进程退出 + 验证新进程启动
  7. 清除停止标志
```

### taf 类型

```
bkms-cli 上传二进制 → bin/<random_name>
        │
        ▼
restart.sh <random_name> <md5>
  1. MD5 校验上传文件
  2. 从运行中的进程动态确定 TAF_BIN_PATH 和 SERVER_NAME
  3. 若无法确定路径（进程未运行），直接失败退出
  4. 创建停止标志（暂停 monitor 自动拉起）
  5. 删除旧二进制，移动新文件到 TAF_BIN_PATH
  6. 发送 SIGUSR2 给当前进程触发优雅重启
  7. 等待旧进程退出 + 验证新进程启动
  8. 清除停止标志
```

## 进程重启（publish restart）

不替换二进制，仅「杀死业务进程 → 由 monitor 重新拉起」，过程中 Pod 不重建。入口为 `bkms-cli app publish restart`，
由 server 异步完成 Pod 内命令执行，不新增脚本，trpc / taf 共用同一流程。

### 接口

`POST /api/v1/devmode/{appID}/restart`（异步接口），请求体中 `envs` / `envTypes` / `instances` 必须且只能指定其中一个：

```json
{"envs":      {"names": ["stage", "test"]}}
{"envTypes":  {"types": ["development", "test"]}}
{"instances": {"envName": "stage", "instanceIDs": ["pod1", "pod2"]}}
```

| 类型 | 目标 | 不满足条件时（生产环境 / 集群配置不完整 / 未启用 devmode / 无 Running 实例） |
|------|------|------|
| `envs` | 各环境下所有 Running 实例 | 严格模式：整个请求返回错误，不执行任何重启 |
| `envTypes` | 匹配类型的环境下所有 Running 实例 | 仅允许 development / test / staging，含 `production` 直接拒绝；其余不满足条件的环境跳过，并在 `skipped` 中返回原因 |
| `instances` | 指定环境下的指定实例 | 严格模式；实例必须属于该环境 |

接口同步创建 pending 记录、把**本次请求的所有实例放进同一条任务消息**投递到 asynq 后立即返回 `202`：

```json
{"data": {"requestedAt": "2026-10-20T06:34:16Z", "instanceCount": 100, "recordIDs": ["..."], "skipped": []}}
```

CLI 拿到 `recordIDs` 后按记录 ID 精确轮询：先按 `statuses=pending&statuses=running` 只取 1 条拿到未完成数量，
数量归零后再按 ID 拉取全部记录做汇总。查询接口支持 `envNames` / `envTypes` / `recordIDs`（三选一）、`since`、`statuses`；
列表类参数通过重复 query 参数传递（如 `statuses=pending&statuses=running`）。

集群访问使用当前用户的 BCS Token：任务执行时由 worker 按 ctx 中的用户身份重新获取，不写入队列。

### 队列与单实例执行流程

一次请求对应一条 asynq 消息，消息 payload 包含本次所有实例的记录 ID 与集群信息。worker 内按最大 20 个实例并行执行，每个实例：

```
a. cat pid.conf                                   记录旧 PID（仅用于展示）；失败则不执行后续步骤
b. source utils.sh && { bash stop.sh; rm -f "${PID_CONF}"; clear_stop_flag; }
c. 轮询 pid.conf                                  文件出现即视为成功，内容即新 PID（超时 30s）
d. 更新发布记录为 success / failed
```

- 记录先于任务创建，任务重投递时跳过已进入终态的实例，避免重复重启；任务不自动重试（重启非幂等）
- 从 b 开始的步骤不随请求取消（`context.WithoutCancel` + 独立超时），记录反映真实结果
- 每个实例结束时上报 `bkms_devmode_restart_duration_seconds{status=ok|err|timeout}`，并输出一条带耗时的结构化日志

- 先 `source utils.sh`，失败则不执行 stop，避免「进程已被杀、flag 无法清除」
- 删除 `pid.conf`：stop.sh 不会删除该文件，不删则 c 步会误判成功；删除后只有 `start.sh` 在确认进程启动后才会重新写入
- stop.sh 失败也会删除 `pid.conf` 并清除 flag，让 monitor 兜底；该实例直接记为失败，不再轮询
- 拉起由 monitor 完成（发现进程不在且无 stop flag → 调用 `start.sh`），保证业务进程 stdout 指向容器而非 exec 会话

### 记录

复用发布记录表 `devmode_publish_records`：

- 新增 `action` 字段：`publish` / `restart`，历史数据无该字段，按 `publish` 处理
- 新增状态 `pending` / `running`：请求开始时为所有实例写入 `pending`，执行时更新为 `running`，结束时更新为 `success` / `failed`
- `ListLatestByInstance`（实例列表展示最近一次发布信息）排除 `restart` 记录
- `publish history` 同时展示两类记录，以 Action 列区分
- 审计记录按实例写入（`OperationTypeRestart`），每个实例一条

### 已知风险（测试时留意）

- CLI 等待超时（固定 3 分钟）或用户 Ctrl+C 时任务仍在服务端执行，以 `publish history` 为准
- worker 进程异常退出时，记录可能停留在 `running`，任务不自动重试，需要手动重新发起
- 不区分调用来源，同一用户、同一条件并发提交两次时，轮询会同时看到两次的记录（不引入 batchID 的已知限制）
- 同一实例上 publish 与 restart 并发执行时会相互影响 stop flag 与 `pid.conf`
- 进程空窗期可能触发 liveness probe 导致容器重启

## 全局错误定义

| 错误变量 | 说明 |
|---------|------|
| `ErrNotAllowed` | 当前环境不允许使用开发模式 |
| `ErrUnsupportedAppType` | 不支持的应用类型（仅支持 trpc / taf） |
| `ErrAppNameRequired` | 应用名称未指定 |
| `ErrStartupCommandRequired` | 启动命令未指定 |
| `ErrTrpcBinaryPathRequired` | trpc 二进制路径未指定（仅 trpc 类型） |



## 脚本详细说明

### trpc 脚本

#### init.sh — 容器入口点

容器启动时的第一个脚本，替代原始启动命令。

1. 校验 `BKMS_APP_NAME` 环境变量非空
2. 检测 Linux 发行版（支持 debian/redhat/tencentos/suse/alpine/arch）
3. 检查必要依赖命令（flock、awk、ps、nohup、envsubst、md5sum 等），缺失则自动安装
4. 创建 `scripts/`、`bin/`、`logs/` 目录
5. 将 ConfigMap 只读挂载的脚本复制到 `scripts/` 可写目录并赋予执行权限
6. 通过 `exec` 启动 `monitor.sh` 守护进程（替换当前进程，成为 PID 1）

#### utils.sh — 公共变量和函数库

被其他脚本通过 `source` 引用，不独立执行。

- **路径常量**：`BKMS_DEV_MODE_PATH`、`BKMS_DEV_MODE_BIN_PATH`、`BKMS_MONITOR_PATH` 等
- **二进制搜索**：`TRPC_BIN_SEARCH_PATHS` 数组（模板配置路径 > `/usr/local/trpc/bin` > `/usr/local/trpc/conf`）
- **`get_actual_server_name()`**：在搜索路径中不区分大小写查找 `BKMS_APP_NAME` 对应的二进制文件，设置 `TRPC_BIN_PATH`、`SERVER_NAME`、`TRPC_LOG_PATH`
- **进程管理**：`get_process_count()`、`get_process_pid()`、`get_process_pids()`、`is_process_running()`
- **停止标志**：`create_stop_flag()` / `clear_stop_flag()` / `is_stop_flag_set()` — 通过文件标志通知 monitor 暂停自动拉起
- **日志函数**：`log_info` / `log_warn` / `log_error` / `log_fatal`，输出到 `LOG_FILE`

#### start.sh — 启动业务进程

1. 调用 `get_actual_server_name()` 定位二进制文件
2. 清除停止标志，恢复 monitor 托管
3. 通过 `flock` 文件锁实现单例，防止并发启动
4. 检查进程是否已在运行，已运行则跳过
5. 验证可执行文件存在并设置权限
6. 通过 `nohup` 在子 Shell 中后台执行用户启动命令（`{{.BKMS_CUSTOM_START_SCRIPT}}` 模板渲染）
7. 等待 3 秒后验证进程启动成功，记录 PID 到 `pid.conf`

#### stop.sh — 优雅停止业务进程

1. 调用 `get_actual_server_name()` 定位进程
2. 创建停止标志，通知 monitor 暂停自动拉起
3. 获取所有匹配的进程 PID
4. 对每个进程：先发 `SIGTERM`，等待 5 秒优雅退出；超时则发 `SIGKILL` 强制终止
5. 最终验证所有进程已退出

#### monitor.sh — 守护进程

作为容器主进程（PID 1）持续运行，每 10 秒执行一次监控循环。

- **信号转发**：注册 SIGTERM/SIGINT/SIGHUP/SIGUSR1/SIGUSR2 处理器，将信号转发给业务进程；收到 SIGTERM/SIGINT 时执行优雅关闭（SIGTERM → 等待 30 秒 → SIGKILL）
- **进程存活检查**：
    - 进程不存在 → 检查停止标志，无标志则调用 `start.sh` 自动拉起
    - 进程存在 → 检查 PID 是否变化并更新 `pid.conf`
- **Panic 检测**（trpc 独有）：每 10 秒扫描 `TRPC_LOG_PATH` 下最近修改的日志文件末尾 100 行，匹配 `panic:` / `[PANIC]` / `fatal error:` / `runtime error:` / `SIGABRT` / `SIGSEGV` 等模式

#### restart.sh — 二进制热更新

由 `bkms-cli` 调用，用法：`restart.sh <random_name> <md5>`。

1. 校验参数和环境变量，调用 `get_actual_server_name()` 定位当前二进制
2. 创建停止标志，暂停 monitor 自动拉起
3. 校验上传文件存在，MD5 校验（兼容 `md5sum` 和 `md5` 命令）
4. 删除旧二进制，将新文件从 `bin/<random_name>` 移动到 `TRPC_BIN_PATH/<SERVER_NAME>`
5. 向当前进程发送 `SIGUSR2`（kill -12）触发优雅重启
6. 等待旧进程退出（最长 30 秒，超时依次 SIGTERM → SIGKILL）
7. 清除停止标志，验证新进程启动成功并更新 `pid.conf`

---

### taf 脚本

#### init.sh — 容器入口点

与 trpc 版本逻辑完全一致，仅工作目录不同（`/data/bkms/dev-mode/taf`）。

1. 校验 `BKMS_APP_NAME`，检测 OS，检查/安装依赖
2. 创建目录，复制脚本到可写目录
3. `exec` 启动 `monitor.sh`

#### utils.sh — 公共变量和函数库

与 trpc 版本的主要差异：

- **无 `TRPC_BIN_SEARCH_PATHS`**：taf 不预设二进制搜索路径
- **`get_taf_server_info()`**（替代 trpc 的 `get_actual_server_name()`）：从 `ps -ef` 中匹配 `SERVER_NAME_ENV` 的运行进程，动态提取可执行文件的目录和文件名，设置 `TAF_BIN_PATH` 和 `SERVER_NAME`；若进程未运行则返回失败
- **进程匹配**：通过 `SERVER_NAME_ENV` 在进程列表中匹配，排除 `taf-start.sh`、`start.sh`、`monitor.sh` 等管理脚本
- 其余函数（日志、进程管理、停止标志）与 trpc 版本一致

#### start.sh — 启动业务进程

与 trpc 版本的差异：

1. 不需要调用 `get_actual_server_name()` 查找二进制路径
2. 不验证可执行文件是否存在
3. 通过 `nohup ${BKMS_MONITOR_PATH}/taf-start.sh` 启动进程（而非直接执行用户命令）
4. 其余流程一致：清除停止标志 → 单例锁 → 重复检查 → 启动 → 验证 → 记录 PID

#### taf-start.sh — taf 专用启动包装脚本

**不在 assets 目录中**，由 Go 代码在构建 ConfigMap 时根据 `Config.StartupCommand` 动态生成，内容为 `exec <StartupCommand>`。以 wrap 方式执行用户的启动命令，使业务进程直接替换当前 shell 进程。

#### stop.sh — 优雅停止业务进程

与 trpc 版本逻辑一致，差异：

- 通过 `SERVER_NAME_ENV` 匹配进程（而非 `TRPC_BIN_PATH/SERVER_NAME`）
- 优雅等待超时为 15 秒（trpc 为 5 秒）

#### monitor.sh — 守护进程

与 trpc 版本的差异：

- **无 Panic 检测**：taf 版不扫描日志文件检测 panic
- 其余逻辑一致：信号转发、进程存活检查、自动拉起、PID 变化检测

#### restart.sh — 二进制热更新

与 trpc 版本的差异：

1. 调用 `get_taf_server_info()` 从运行中的进程动态确定二进制路径（而非预设搜索路径）
2. 若进程未运行导致无法确定路径，则直接失败退出
3. 使用 `TAF_BIN_PATH` 替代 `TRPC_BIN_PATH` 进行二进制替换
4. 其余流程一致：MD5 校验 → 替换二进制 → SIGUSR2 → 等待旧进程退出 → 验证新进程
