# bkms-cli app component Reference

`bkms-cli app component` 用于管理应用上的组件实例。仅 tRPC / TAF 应用可用。

组件实例有两种形态：

- `ref`：引用工作空间组件实例。
- `inst`：在应用上直接创建的组件实例。

包含以下子命令：

- `list`：列出应用上的组件实例。
- `create`：引用工作空间组件实例。本期不支持创建 `inst`。
- `delete`：按应用内名称删除组件实例，`ref` 与 `inst` 均可删除。删除 `ref` 只移除应用上的引用，不会删除工作空间中的组件实例。

`ref` 不拷贝 properties，部署时使用被引用工作空间组件实例的配置，且不能通过 CLI 编辑。创建或删除后需再次部署才会在集群生效。

## 常用场景

先查看工作空间中可引用的组件实例，再挂到应用上。

```bash
bkms-cli workspace component list
bkms-cli app component create --app my-app --ref shared-limits
```

指定应用内名称：

```bash
bkms-cli app component create --app my-app --ref shared-limits --name my-limits
```

只列出引用类组件实例：

```bash
bkms-cli app component list --app my-app --kind ref
```

删除应用上的组件实例。`--name` 为 `list` 返回的应用内名称。

```bash
bkms-cli app component delete --app my-app --name my-limits
```

## list

列出应用组件实例。`--kind` 可选值为 `ref` 或 `inst`；未指定时返回全部。

```bash
bkms-cli app component list --app my-app
bkms-cli app component list --app my-app --kind ref -o json
bkms-cli app component list --app my-app -o 'jq=[.[] | .name]'
```

## create

`--ref` 为工作空间组件实例名称（`workspace component list` 的 `name`）。`--name` 可选，省略时由服务端生成。

```bash
bkms-cli app component create --app my-app --ref shared-limits
```

创建成功后输出应用内组件实例名称：

```
✓ App component referenced successfully
  Name: shared-limits-a1b2c
```

## delete

按应用内名称删除组件实例，适用于 `ref` 与 `inst`。删除 `ref` 不会删除工作空间中被引用的组件实例。

```bash
bkms-cli app component delete --app my-app --name my-limits
```
