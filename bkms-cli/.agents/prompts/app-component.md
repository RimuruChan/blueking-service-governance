# bkms-cli app component Reference

`bkms-cli app component` 用于把工作空间组件实例引用到应用上。当前只支持引用，不支持按组件定义自定义实例化。仅 tRPC / TAF 应用可用。

包含以下子命令：

- `list`：列出应用上的组件（引用和自定义都会展示）。
- `create`：按空间组件名称创建引用。
- `delete`：按应用内组件名称删除挂载（不会删除空间组件本身）。

引用不会拷贝 properties。部署时使用空间组件的配置；引用组件不能通过 CLI 编辑。创建或删除后需要再部署才会在集群生效。

## 常用场景

先查看空间里有哪些可引用实例，再挂到应用上。

```bash
bkms-cli workspace component list
bkms-cli app component create --app my-app --ref shared-limits
```

指定应用内名称，避免使用后端生成的随机后缀。

```bash
bkms-cli app component create --app my-app --ref shared-limits --name my-limits
```

查看应用当前组件，只看引用项。

```bash
bkms-cli app component list --app my-app --source reference
```

删除应用组件（引用或自定义）。`--name` 是 `list` 返回的应用内名称。

```bash
bkms-cli app component delete --app my-app --name my-limits
```

## list

列出应用组件。`source` 为 `reference` 表示引用空间组件，`custom` 表示自定义实例。`--source` 可过滤。

```bash
bkms-cli app component list --app my-app
bkms-cli app component list --app my-app --source reference -o json
bkms-cli app component list --app my-app -o 'jq=[.[] | .name]'
```

## create

`--ref` 是空间组件名称（`workspace component list` 的 `name`）。`--name` 可选，省略时由后端生成。

```bash
bkms-cli app component create --app my-app --ref shared-limits
```

创建成功后输出应用内组件名称：

```
✓ App component referenced successfully
  Name: shared-limits-a1b2c
```

## delete

只解除应用侧挂载，空间组件仍在。

```bash
bkms-cli app component delete --app my-app --name my-limits
```
