/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

// Package overview 聚合应用在各环境下的部署总览数据。
package overview

import "time"

// InstanceCounts 单环境实例数。EnvRow.Instances 为 nil 表示不可用（缺 workload 或集群错误）。
type InstanceCounts struct {
	Running  int32
	Expected int32
	Abnormal int32
}

// ResourceSpec 生效资源规格字符串（Kubernetes quantity，透传）。
type ResourceSpec struct {
	CPULimits      string
	CPURequests    string
	MemoryLimits   string
	MemoryRequests string
}

// AutoscalingMetric 单条 GPA 扩缩容指标（cpu / memory 利用率阈值）。
type AutoscalingMetric struct {
	// Resource 指标资源类型：cpu / memory
	Resource string
	// AverageUtilization 平均使用率阈值（百分比，1-100）
	AverageUtilization int32
}

// AutoscalingStatus 集群中 GPA CR 的运行状态（与 GPA 详情接口 status 对齐）。
// AutoscalingInfo.Status 为 nil 表示 CR 不存在或查询失败（不阻断总览）。
type AutoscalingStatus struct {
	CurrentReplicas int32
	DesiredReplicas int32
	LastScaleTime   string
	// Phase：Active / Paused / Limited / Failed / Initializing / Unknown
	Phase string
	// StatusMessage 非 True condition 的汇总消息（出错时前端可展示）
	StatusMessage string
}

// AutoscalingInfo 环境 GPA 配置摘要，供总览展示实例区间与指标阈值。
// EnvRow.Autoscaling 为 nil 表示该环境无 GPA 配置。
type AutoscalingInfo struct {
	// Enabled 是否启用（false 时配置仍返回，供前端区分「未配置」与「已关闭」）
	Enabled bool
	// CRName GPA CR metadata.name，用于回查集群状态（不对外序列化）
	CRName string
	// MinReplicas / MaxReplicas 实例数区间
	MinReplicas int32
	MaxReplicas int32
	// Metrics 指标列表（可能为空，例如仅配置了定时扩缩容）
	Metrics []AutoscalingMetric
	// ComputeByLimits true 时利用率以 limits 为基准，false 时以 requests 为基准
	ComputeByLimits bool
	// Status 集群 CR 运行状态；未启用 / CR 缺失 / 查询失败时为 nil
	Status *AutoscalingStatus
}

// EnvRow 部署总览表格的一行。
type EnvRow struct {
	EnvID               string
	EnvName             string
	EnvDisplayName      string
	EnvType             string
	EnvKind             string
	DeployStatus        string
	LastDeployStartedAt *time.Time
	Autoscaling         *AutoscalingInfo
	Resources           ResourceSpec
	Instances           *InstanceCounts
}

// Result 单个应用的部署总览结果。
type Result struct {
	Envs []EnvRow
}
