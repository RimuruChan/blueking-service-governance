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

// Package serializer 定义应用仪表盘绑定相关 Gin 输入输出结构。
package serializer

import (
	"github.com/samber/lo"

	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
)

// AppURIInput 路径参数（仅应用 ID）。
type AppURIInput struct {
	AppID string `uri:"appID" binding:"required,uri_slug"`
}

// AppDashboardURIInput 应用仪表盘路径参数。
type AppDashboardURIInput struct {
	AppID string `uri:"appID" binding:"required,uri_slug"`
	UID   string `uri:"uid" binding:"required,min=1"`
}

// AppDashboardCreateInput 创建应用仪表盘绑定请求。
type AppDashboardCreateInput struct {
	// UID 仪表盘 uid
	UID string `json:"uid" binding:"required,min=1"`
}

// AppDashboardUpdateInput 更新应用仪表盘绑定请求。
type AppDashboardUpdateInput struct {
	// UID 新的仪表盘 uid（仅支持变更绑定的仪表盘）
	UID *string `json:"uid" binding:"omitempty,min=1"`
}

// DashboardOutput 仪表盘输出。
type DashboardOutput struct {
	// UID 仪表盘 uid
	UID string `json:"uid"`
	// Title 仪表盘标题
	Title string `json:"title"`
	// URL 仪表盘访问 URL
	URL string `json:"url"`
}

// NewDashboardOutputs 将仪表盘信息列表转换为输出列表。
func NewDashboardOutputs(items []*bkmapi.DashboardItem) []*DashboardOutput {
	return lo.Map(items, func(item *bkmapi.DashboardItem, _ int) *DashboardOutput {
		return &DashboardOutput{
			UID:   item.UID,
			Title: item.Title,
			URL:   item.URL,
		}
	})
}

// ListDashboardsResp 获取应用绑定仪表盘列表的响应。
type ListDashboardsResp struct {
	// Data 应用绑定的仪表盘列表
	Data []*DashboardOutput `json:"data"`
}

// EmptyOutput is the JSON response for APIs that return no data.
type EmptyOutput struct{}
