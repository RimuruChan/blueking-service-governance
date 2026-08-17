# 工作空间 AppSpec 默认规则多环境类型索引

规则字段由单个 `envType` 调整为 `envTypes` 数组后，唯一约束改为按数组元素展开。

- workspace_app_spec_rules 新建索引：
  - workspaceID_1_configType_1_envTypes_1: AppDefault RuleStoreMongo：同一 workspace + configType 下，任一环境类型最多被一条规则占用；`envTypes` 为数组，MongoDB 按元素建立 multikey 唯一索引。

旧版单字段索引 `workspaceID_1_configType_1_envType_1` 曾由 `NewRuleStoreMongo` 在进程启动时创建，从未进入 migration。线上与测试环境尚未实际使用该集合；发布时直接删除相关集合即可，本迁移不再处理旧索引的删除或回建。
