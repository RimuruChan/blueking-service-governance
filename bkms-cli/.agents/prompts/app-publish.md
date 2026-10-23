# bkms-cli app publish Reference

`bkms-cli app publish` 用于将本地二进制文件发布到应用开发模式（dev mode）容器中，并串行对目标实例执行上传与重启流程。

该命令仅适用于已开启 dev mode 的非正式环境。发布前会先校验应用、环境、dev mode 配置和目标实例；发布时会把本地文件上传到容器工作目录下的 `bin` 路径，并执行 `restart.sh` 完成切换。

BCS Auth Info 由服务端自动通过当前登录凭据获取，无需手动传入 token。

实例选择有两种方式：

- 使用 `--instance-ids` 显式指定要发布的实例，多个实例用逗号分隔。
- 使用 `--all` 自动获取当前环境下所有 `Running` 状态实例。

## 常用场景

发布到指定实例，适合小范围验证。

```bash
# 发布到单个实例
bkms-cli app publish --app myapp --env stage -f ./bin/server --instance-ids pod-1

# 发布到多个实例
bkms-cli app publish --app myapp --env stage -f ./bin/server --instance-ids pod-1,pod-2
```

自动发布到当前环境全部 Running 实例，适合开发模式下做整体验证。

```bash
# 自动选择所有 Running 实例
bkms-cli app publish --app myapp --env stage -f ./bin/server --all
```

## publish

`publish` 命令负责把本地二进制文件上传到开发模式容器，并逐个实例执行重启脚本。命令执行前会打印本次将操作的实例总数和实例名列表，随后按实例串行处理。

```bash
# 发布到指定实例
bkms-cli app publish --app myapp --env test -f ./bin/server --instance-ids pod-1

# 发布到多个实例
bkms-cli app publish --app myapp --env test -f ./bin/server --instance-ids pod-1,pod-2

# 自动发布到所有 Running 实例
bkms-cli app publish --app myapp --env test -f ./bin/server --all
```

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--app` | 是 | 应用 ID |
| `--env` | 是 | 环境名称 |
| `-f, --file` | 是 | 本地二进制文件路径 |
| `--instance-ids` | 否 | 实例 ID（pod 名），逗号分隔；与 `--all` 二选一 |
| `--all` | 否 | 发布到所有 Running 实例；与 `--instance-ids` 二选一 |

### 使用约束

- `--instance-ids` 和 `--all` 互斥，二选一。
- 未使用 `--all` 时，必须显式提供 `--instance-ids`。
- 目标环境必须是已开启 dev mode 的非正式环境。

## publish restart

`publish restart` 只重启开发模式容器内的业务进程，不替换二进制，Pod 不重建。接口是异步的：服务端校验目标、创建 pending 记录并把**本次请求的所有实例放进同一条队列消息**后立即返回；worker 在消息内按最大 20 个实例并行执行 stop → 等待 monitor 拉起。

CLI 提交成功后会按**本次返回的记录 ID**周期性查询重启进度（每 2 秒查询一次 `pending` / `running` 记录数），全部进入 `success` / `failed` 终态后拉取完整记录并打印结果；最长等待 3 分钟。等待超时或按 Ctrl+C 只影响 CLI 的展示，服务端任务会继续执行，最终结果以 `publish history` 为准。

目标必须且只能用 `--env` 或 `--env-type` 中的一个来指定：

- `--env`：一个或多个环境，重启其中所有 Running 实例。任一环境不满足条件（未开启 dev mode、正式环境、没有 Running 实例等）时整个请求报错，不会执行任何重启。
- `--env-type`：一个或多个环境类型（development / test / staging），重启匹配环境下的所有 Running 实例。不满足条件的环境会被跳过，并输出跳过原因。
- `--instance-ids`：只重启指定实例，必须搭配单个 `--env` 使用。

```bash
# 重启多个环境的全部 Running 实例
bkms-cli app publish restart --app myapp --env stage,test

# 按环境类型重启
bkms-cli app publish restart --app myapp --env-type development,test

# 重启指定环境下的指定实例
bkms-cli app publish restart --app myapp --env stage --instance-ids pod-1,pod-2
```

只要有实例重启失败，命令就以非 0 退出码结束。

目标实例较多时（多个环境，或单个环境实例数超过 10 个），等待时间会更长；若超过 3 分钟仍未完成，CLI 会提示仍在执行，此时以 `publish history` 中的记录为准，不要直接重复提交。

## publish history

`publish history` 查询发布和重启记录，每个实例一条，按时间倒序排列。`ACTION` 列用来区分 `publish` 和 `restart`，`ENVNAME` 列标明记录所属环境。

查询范围必须且只能用 `--env` 或 `--env-type` 中的一个来指定：`--env` 只查单个环境；`--env-type` 跨该应用所有匹配类型的环境查询，并在服务端统一分页（多个类型用逗号分隔）。

```bash
# 单个环境
bkms-cli app publish history --app myapp --env stage

# 按环境类型跨环境查询
bkms-cli app publish history --app myapp --env-type development,test
```
