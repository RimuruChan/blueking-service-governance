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

// EnvRow 部署总览表格的一行。
type EnvRow struct {
	EnvID               string
	EnvName             string
	EnvDisplayName      string
	EnvType             string
	EnvKind             string
	DeployStatus        string
	LastDeployStartedAt *time.Time
	AutoscalingEnabled  bool
	Resources           ResourceSpec
	Instances           *InstanceCounts
}

// Result 单个应用的部署总览结果。
type Result struct {
	Envs []EnvRow
}
