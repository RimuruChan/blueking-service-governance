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
	// AppComponentSourceReference 应用组件来自空间组件引用
	AppComponentSourceReference = "reference"
	// AppComponentSourceCustom 应用组件为自定义实例
	AppComponentSourceCustom = "custom"
)

// AppComponent 应用组件（来自 GET /apps/:appID 的 appModelSpec.components）
type AppComponent struct {
	// 应用内组件名称
	Name string `json:"name" yaml:"name"`
	// 来源：reference（引用空间组件）或 custom（自定义实例）。由 CLI 根据 refWorkspaceCompName 计算。
	Source string `json:"source" yaml:"source"`
	// 组件类型（引用时由空间组件回填）
	Type string `json:"type" yaml:"type"`
	// 组件版本（引用时由空间组件回填）
	Version string `json:"version" yaml:"version"`
	// 引用的空间组件名称，非空表示引用
	RefWorkspaceCompName string `json:"refWorkspaceCompName,omitempty" yaml:"refWorkspaceCompName,omitempty"`
	// 生效范围类型：global / environment
	ScopeType string `json:"scopeType" yaml:"scopeType"`
	// 生效的环境列表
	ScopeEnvNames []string `json:"scopeEnvNames" yaml:"scopeEnvNames"`
	// 组件属性（引用时为空间组件的值）
	Properties map[string]any `json:"properties" yaml:"properties" table:"-"`
}

// WorkspaceComponent 工作空间组件实例
type WorkspaceComponent struct {
	// 组件名称，用于应用引用
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
