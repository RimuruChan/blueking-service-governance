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

// Package client 应用实例相关类型定义
package client

// PolarisInfo 北极星实例注册信息
type PolarisInfo struct {
	// 北极星命名空间
	ServiceNamespace string `json:"serviceNamespace" yaml:"serviceNamespace"`
	// 北极星服务名
	ServiceName string `json:"serviceName" yaml:"serviceName"`
	// 权重
	Weight string `json:"weight" yaml:"weight"`
	// 健康状态
	IsHealthy bool `json:"isHealthy" yaml:"isHealthy"`
	// 是否启用健康检查
	EnableHealthCheck bool `json:"enableHealthCheck" yaml:"enableHealthCheck"`
}

// Instance 应用实例（即 k8s pod）
type Instance struct {
	// 实例 ID（即 k8s pod 的 name）
	ID string `json:"id"`
	// 部署记录 ID
	DeployID string `json:"deployID"`
	// Pod IP
	IP string `json:"ip"`
	// 镜像
	Image string `json:"image"`
	// 重启次数
	RestartCount string `json:"restartCount"`
	// 状态，由 pod.status.phase 等解析获得
	// 常见值: Failed，Pending，Running，Succeeded，Unknown
	Status string `json:"status"`
	// 状态详情，一般为 pod.status.reason
	Message string `json:"message"`
	// 健康状态，即 k8s 探针检查结果
	IsHealthy bool `json:"isHealthy"`
	// 存在时间，格式如：2d1h，24m29s
	Age string `json:"age"`
	// 北极星实例状态列表（一个 Pod 可能注册到多个北极星服务）
	// 若应用未配置北极星或 Pod 未在北极星中注册，则为空列表
	PolarisInfos []PolarisInfo `json:"polarisInfos" yaml:"polarisInfos"`
	// 北极星权重覆盖值，未单独设置时为空字符串
	PolarisWeightOverride string `json:"polarisWeightOverride" yaml:"polarisWeightOverride"`
}

// ListAppInstancesRespData 应用实例列表响应
type ListAppInstancesRespData struct {
	Data PaginatedInstances `json:"data"`
}

// ListAppInstancesOptions 查询应用实例列表参数。
type ListAppInstancesOptions struct {
	// Page 页码，从 1 开始。
	Page int
	// PageSize 每页数量。
	PageSize int
	// PodIPs 按 Pod IP 精确过滤，多值取 OR。
	PodIPs []string
	// NodeIPs 按 Node IP 精确过滤，多值取 OR；与 PodIPs 之间也取 OR。
	NodeIPs []string
}

// PaginatedInstances 分页查询实例结果
type PaginatedInstances struct {
	// 结果数量
	Count string `json:"count"`
	// 查询结果
	Results []Instance `json:"results"`
}

// ExecuteTrpcAdminCmdOptions 执行 Trpc 管理命令请求参数
type ExecuteTrpcAdminCmdOptions struct {
	InstanceIDs []string          `json:"instanceIDs"`
	Method      string            `json:"method"` // GET/POST/PUT
	URL         string            `json:"url"`
	Params      map[string]string `json:"params"`
	Body        string            `json:"body"`
}

// ExecuteTafAdminCmdOptions 执行 Taf 管理命令请求参数
type ExecuteTafAdminCmdOptions struct {
	InstanceIDs []string `json:"instanceIDs"`
	Command     string   `json:"command"`
}

// AdminCmdResult 管理命令执行结果
type AdminCmdResult struct {
	InstanceID string `json:"instanceID"`
	Success    bool   `json:"success"`
	Detail     string `json:"detail"`
}

// ListTrpcAdminCmdsRespData 查询 Trpc 管理命令列表响应
type ListTrpcAdminCmdsRespData struct {
	Data ListTrpcAdminCmdsOutput `json:"data"`
}

// ListTrpcAdminCmdsOutput 查询 Trpc 管理命令列表输出
type ListTrpcAdminCmdsOutput struct {
	Count   string   `json:"count"`
	Results []string `json:"results"`
}

// ExecuteAdminCmdRespData 执行管理命令响应（Trpc/Taf 通用）
type ExecuteAdminCmdRespData struct {
	Data ExecuteAdminCmdOutput `json:"data"`
}

// ExecuteAdminCmdOutput 执行管理命令输出
type ExecuteAdminCmdOutput struct {
	Count   string           `json:"count"`
	Results []AdminCmdResult `json:"results"`
}

// PortForwardTunnelOptions 端口转发隧道参数。
type PortForwardTunnelOptions struct {
	// InstanceID 目标 Pod 实例 ID。
	InstanceID string
	// RemotePort 目标 Pod 端口。
	RemotePort int
	// LocalPort CLI 本地监听端口，用于服务端审计。
	LocalPort int
}

// UpdateInstancePolarisOptions 更新实例北极星配置参数
type UpdateInstancePolarisOptions struct {
	// TrafficLaneName 泳道名称
	TrafficLaneName string `json:"trafficLaneName"`
	// InstanceIDs 实例 ID 列表
	InstanceIDs []string `json:"instanceIDs"`
	// Weight 目标权重（nil 表示不修改）
	Weight *int `json:"weight,omitempty"`
	// Isolate 隔离状态（nil 表示不修改）
	Isolate *bool `json:"isolate,omitempty"`
}

// BatchDeleteInstancesOptions 批量删除实例参数
type BatchDeleteInstancesOptions struct {
	// TrafficLaneName 泳道名称
	TrafficLaneName string `json:"trafficLaneName"`
	// InstanceIDs 实例 ID 列表
	InstanceIDs []string `json:"instanceIDs"`
}
