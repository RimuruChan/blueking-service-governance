# bkms-cli E2E 测试

基于 [testscript](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript)：
用 txtar 脚本驱动编译后的 CLI 二进制，做黑盒断言。

## 运行

```bash
# 全量
make e2e-script-test
# 单领域
go test -run TestApp ./test/e2e-script/
# 单脚本
go test -run 'TestApp/crud' ./test/e2e-script/
```

需先准备 `test/e2e-script/.env`（已 gitignore），含 `BKMS_API_URL`、`BKMS_USERNAME`、
`BKMS_TOKEN`、`BKMS_WORKSPACE_ID`、`BKMS_APP_ID`、`BKMS_ENV_NAME`。
`make` 会自动加载；直接用 `go test` 需自行导出并设置 `BKMS_CLI_BIN`。

## 目录

脚本按领域分目录，每个目录一个测试函数：`app/` → `TestApp`、`appspec/` → `TestAppspec`、
`base/` → `TestBase`、`deploy/`、`envvar/`、`extension/`、`unauth/`（未认证场景，配置不含 token）。

Go 侧：`e2e_test.go`（入口）、`setup.go`（环境变量注入 + 登录）、`cmds.go`（自定义命令）。

新增脚本：直接在领域目录下建 `.txtar`。新增领域：`Params.Dir` 不递归，需在
`e2e_test.go` 加一行 `func TestFoo(t *testing.T) { runDomain(t, "foo", setup) }`。

## 脚本变量

Setup 注入（testscript 默认只传 `PATH`）：

- `$BKMS_CLI_BIN` — CLI 路径，所有 `exec` 用它
- `$BKMS_CLI_CONFIG` — 隔离的配置文件，不污染 `~/.bkms`
- `$BKMS_*` — `.env` 各项原样透传
- `$UNIQUE` — 每脚本独享时间戳，用于无冲突资源名
- `$APP_SPEC` — 含唯一 app 名的 spec 文件路径

## 自定义命令

均支持 `!` 反向断言，其余用 testscript 内置命令：

- `jsonhas <key>...` — stdout 是 JSON 且含指定顶层键
- `jsonfield <key>=<value>` — 字段值匹配

错误断言用内置 `stderr <regex>`，成功输出用 `stdout <regex>`（两者均为 RE2 正则，
`(?m)` 多行匹配）。CLI 错误统一走 stderr，正常输出走 stdout。
