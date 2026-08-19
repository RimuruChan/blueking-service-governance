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

package client

const (
	// AppComponentKindRef 引用工作空间组件实例
	AppComponentKindRef = "ref"
	// AppComponentKindInst 应用上直接创建的组件实例
	AppComponentKindInst = "inst"
)

// AppComponent 应用组件实例
type AppComponent struct {
	// 应用内组件实例名称
	Name string `json:"name" yaml:"name"`
	// 形态：ref 或 inst。refWorkspaceCompName 非空时为 ref。
	Kind string `json:"kind" yaml:"kind"`
	// 组件类型
	Type string `json:"type" yaml:"type"`
	// 组件版本
	Version string `json:"version" yaml:"version"`
	// 引用的工作空间组件实例名称，非空表示 ref
	RefWorkspaceCompName string `json:"refWorkspaceCompName,omitempty" yaml:"refWorkspaceCompName,omitempty"`
	// 生效范围类型：global / environment
	ScopeType string `json:"scopeType" yaml:"scopeType"`
	// 生效的环境列表
	ScopeEnvNames []string `json:"scopeEnvNames" yaml:"scopeEnvNames"`
	// 组件属性。ref 使用被引用工作空间组件实例的值。
	Properties map[string]any `json:"properties" yaml:"properties" table:"-"`
}

// WorkspaceComponent 工作空间组件实例
type WorkspaceComponent struct {
	// 组件实例名称，应用引用时使用
	Name string `json:"name" yaml:"name"`
	// 组件类型
	Type string `json:"type" yaml:"type"`
	// 组件版本
	Version string `json:"version" yaml:"version"`
	// 生效范围类型：global / environment
	ScopeType string `json:"scopeType" yaml:"scopeType"`
	// 生效的环境列表
	ScopeEnvNames []string `json:"scopeEnvNames" yaml:"scopeEnvNames"`
	// 所属工作空间 ID
	WorkspaceID string `json:"workspaceID" yaml:"workspaceID" table:"-"`
	// 引用该空间组件的应用 ID 列表
	RefAppIDs []string `json:"refAppIDs" yaml:"refAppIDs" table:"-"`
	// 组件属性
	Properties map[string]any `json:"properties" yaml:"properties" table:"-"`
	// 创建时间
	CreatedAt string `json:"createdAt" yaml:"createdAt" table:"-"`
	// 更新时间
	UpdatedAt string `json:"updatedAt" yaml:"updatedAt" table:"-"`
}

// ListWorkspaceComponentsRespData 获取工作空间组件列表返回数据
type ListWorkspaceComponentsRespData struct {
	Data []WorkspaceComponent `json:"data"`
}

// CreateAppComponentRespData 创建应用组件返回数据
type CreateAppComponentRespData struct {
	Data struct {
		Name string `json:"name"`
	} `json:"data"`
}
