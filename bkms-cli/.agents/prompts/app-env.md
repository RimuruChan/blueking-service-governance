# 应用特性环境

特性环境只属于当前应用。创建时从指定的已有环境复制基础信息，以及该应用在该环境中的相关配置，并为特性环境单独创建命名空间。创建后与来源环境相互独立，之后互不影响。
应用命令的 `--app` 支持应用 ID 或名称，`--workspace` 可使用已配置的默认值。

## 常用流程

先查询工作空间标准环境，选择源环境创建特性环境，再使用创建结果中的
`name` 作为部署、配置和实例命令的 `--env` 参数。

```bash
bkms-cli env list -o json
bkms-cli app env create --app my-app --source-env staging --display-name "功能验证环境" -o json
bkms-cli app env list --app my-app
```

列表输出示例：

```text
NAME            DISPLAY NAME   TYPE      STATUS   KIND
feat-my-app-1   功能验证环境   staging   Ready    feature
```

使用 `NAME` 列的环境名称，例如 `feat-my-app-1`，作为后续命令的 `--env` 参数；
`DISPLAY NAME` 是展示名称，不用于定位环境。

```bash
# 将 <特性环境名称> 替换为列表 NAME 列的值（与创建结果中的 name 相同）
bkms-cli app deploy create --app my-app --env <特性环境名称> -f deploy.yaml
bkms-cli app deploy list --app my-app --env <特性环境名称>
```

## app env list

返回当前应用拥有的特性环境。
`status` 表示环境就绪状态（Ready / NotReady）。
默认表格展示名称、展示名、类别、类型和就绪状态；JSON/YAML 保留来源 ID、应用归属和集群等完整信息。

```bash
bkms-cli app env list --workspace ws-demo --app my-app
bkms-cli app env list --app my-app -o json
bkms-cli app env list --app my-app -o 'jq=[.[] | .name]'
```

## app env create

此命令仅创建特性环境。工作空间标准环境使用顶层 `env create` 创建。
`--source-env` 接受源标准环境的名称或 ID，`--display-name` 为必填展示名称。
特性环境不能用作源环境，服务端负责完整业务校验和命名空间分配。
创建结果包含新环境的 `id`、`name`、`kind`、来源 ID 和集群信息。

```bash
bkms-cli app env create --app my-app --source-env staging --display-name "功能验证环境" -o json
```

创建和列表命令支持 `-o json|yaml|table|jq=<表达式>`。

## app env delete

按名称或 ID 删除当前应用的特性环境。
默认展示应用、环境和集群信息并要求确认，脚本可使用 `--yes` 跳过确认。

```bash
bkms-cli app env delete --app my-app --env feat-my-app-1
bkms-cli app env delete --app my-app --env feat-my-app-1 --yes
```

删除前必须先卸载该环境中的应用部署，服务端会拒绝删除仍有部署应用的环境。
