# bkms-cli workspace component Reference

`bkms-cli workspace component list` 列出工作空间中的组件实例。应用可通过 `bkms-cli app component create --ref <name>` 引用其中的 `name`。

若已执行 `workspace set`，可不传 `--workspace`。

## 常用场景

列出默认工作空间中的组件实例。

```bash
bkms-cli workspace component list
```

指定工作空间，并以 JSON 输出完整字段。

```bash
bkms-cli workspace component list --workspace ws-demo -o json
```

提取可引用的组件实例名称。

```bash
bkms-cli workspace component list -o 'jq=[.[] | .name]'
```
