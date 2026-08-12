# 北极星配置：回填 envWeights 并删除旧的 weight

## 背景

北极星权重从「一条配置一个权重」改为「按环境维护」后，`polaris_configs` 的顶层 `weight` 被 `envWeights`（`map[envName]int32`）取代，见 `design_notes/polaris_config_auto_sync.md`。

存量数据没有随之调整，于是出现认知漂移：

- 现网 PolarisConfig CR 里的 `spec.services[0].weight` 仍是旧代码写入的顶层 `weight`（接口默认 `10`）；
- 新代码通过 `GetEnvWeight` 取值，`envWeights` 缺少该环境的 key 时回落到 `DefaultEnvWeight = 100`。

只要这类环境触发一次 CR patch（改 scope、改权重、重新部署），线上单实例权重就会从 `10` 跳到 `100`。本迁移把旧 `weight` 回填到各环境，让平台认知与现网保持一致，随后删除废弃字段。

## 迁移语句说明

`up` 是一条 `update` 命令，`multi: true`，`u` 使用聚合管道，逐文档计算后原地更新。

**筛选条件 `q`**

```json
{ "weight": { "$exists": true } }
```

只处理还残留顶层 `weight` 的文档。字段缺失说明是新代码写入的数据，不需要迁移；`weight` 为 `0` 属于显式配置，照常迁移。

**管道第一阶段：`$set envWeights`**

`$mergeObjects` 合并两个对象，后者覆盖前者：

1. 回填对象，由 `$arrayToObject` + `$map` 构造，把目标环境集合的每个环境映射成 `{ k: 环境名, v: 旧 weight }`。目标环境集合是 `$setUnion` 求的并集：
   - `scopeEnvNames`：已创建、在 scope 内的环境，含尚未部署的；
   - `envStates` 的 key（`$objectToArray` 取出后 `$map` 提 `k`）：有过部署快照的环境，含已离开 scope 但仍在运行的。
   两侧都用 `$ifNull` 兜底成空数组 / 空对象，避免字段缺失导致整个表达式失败。
2. 文档已有的 `envWeights`，同样用 `$ifNull` 兜底成 `{}`。

已有的 `envWeights` 放在后面，意味着**显式设置过的环境权重原样保留，只补齐缺失的 key**。这也让迁移天然幂等：重复执行不会改变结果。

环境名可以直接作为 map key，`store.go` 的 `envFieldPrefix` 已禁止环境名包含 `.`、`$` 等字符。

**管道第二阶段：`$unset weight`**

删除废弃的顶层字段。和回填放在同一条管道里，保证两者要么一起生效、要么都不生效，不会出现「删了 weight 但没回填」的中间态。

## down

无法还原：`envWeights` 是按环境的多个值，退不回单个 `weight`。`down.json` 是空命令数组，回滚只会把 `schema_migrations` 的版本号退回 `000002`，数据保持迁移后的状态。

## 验证

迁移后在业务库执行：

```js
// 应为 0：不应再有文档残留顶层 weight
db.polaris_configs.countDocuments({ weight: { $exists: true } })

// 抽查：scope 内环境都应在 envWeights 里有 key
db.polaris_configs.find({}, { name: 1, scopeEnvNames: 1, envWeights: 1 }).limit(20)
```

第一条查询结果非 `0`，说明滚动发布窗口期内仍有旧版本 Pod 写入过 `weight`（迁移 Job 先于新版本 Pod 启动，但旧 Pod 尚未完全下线）。这类文档需要按同样规则手工补一次。

## 附带说明

若某条配置的 `scopeEnvNames` 与 `envStates` 都为空，回填对象为空，文档会得到一个空的 `envWeights: {}`。模型上该字段是 `omitempty`，读取行为不受影响。
