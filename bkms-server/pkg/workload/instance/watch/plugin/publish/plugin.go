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

// Package publish 以 Watch 插件的形式为流内实例补充最近一次开发模式发布状态
package publish

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	devmode "github.com/RimuruChan/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
	"github.com/RimuruChan/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
	watchplugin "github.com/RimuruChan/blueking-service-governance/bkms-server/pkg/workload/instance/watch/plugin"
)

// pluginName 写入事件的 plugin 字段，前端据此落到 latestPublish
const pluginName = "devmodePublish"

// Lister 拉取指定实例列表各自最近一次的发布记录
type Lister func(ctx context.Context, instanceIDs []string) (map[string]*devmode.PublishRecord, error)

// Plugin 开发模式发布状态插件
type Plugin struct {
	lister Lister
}

var _ watchplugin.Plugin = &Plugin{}

// New 创建发布状态插件
func New(lister Lister) *Plugin {
	return &Plugin{lister: lister}
}

// Name 插件名
func (p *Plugin) Name() string {
	return pluginName
}

// Fetch 返回各实例最近的发布状态；无记录的实例缺省，Runner 不推事件
func (p *Plugin) Fetch(
	ctx context.Context,
	snapshot []watchplugin.InstanceSnapshot,
) (map[string]any, error) {
	// 发布记录以 pod name（即实例 ID）为键，不涉及 IP
	instanceIDs := lo.Map(snapshot, func(instance watchplugin.InstanceSnapshot, _ int) string {
		return instance.ID
	})

	latestByInstance, err := p.lister(ctx, instanceIDs)
	if err != nil {
		return nil, errors.Wrap(err, "list latest devmode publish status")
	}

	payloads := make(map[string]any, len(snapshot))
	for _, instance := range snapshot {
		record, ok := latestByInstance[instance.ID]
		if !ok {
			continue
		}
		payloads[instance.ID] = new(serializer.PublishStatusOutputObj).FromModel(record)
	}

	return payloads, nil
}
