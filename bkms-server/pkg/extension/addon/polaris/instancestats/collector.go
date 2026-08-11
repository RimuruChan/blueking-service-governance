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

// Package instancestats 按环境统计北极星实例数（健康 / 隔离 / 总数）。
//
// 匹配方式与实例列表一致：同一北极星服务下的全部实例会混在一次查询结果里，
// 通过「本环境主部署 Pod IP ∩ 配置 ServicePort」把实例归属到具体环境。
package instancestats

import (
	"context"

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"

	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	polarisInfra "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/polaris"
)

// Stats 单个环境匹配到的北极星实例统计。
// healthy 与 isolated 相互独立，同一实例可同时计入两者。
type Stats struct {
	HealthyInstanceCount  int // isHealthy == true
	IsolatedInstanceCount int // isIsolated == true
	TotalInstanceCount    int // 匹配到本环境的实例总数
}

// Result 一次 Collect 的汇总结果。
type Result struct {
	// EnvStats 各环境匹配到的实例统计
	EnvStats map[string]Stats
	// PolarisInstanceCount 北极星服务下全部实例数（含非平台注册），未拉取时为 0
	PolarisInstanceCount int
}

// Collector 拉取并汇总北极星配置在各环境下的实例统计。
type Collector struct {
	deployRecordStore appmodeldeploy.RecordStore
}

// NewCollector 创建统计器。
func NewCollector(deployRecordStore appmodeldeploy.RecordStore) *Collector {
	return &Collector{deployRecordStore: deployRecordStore}
}

// Collect 返回配置关联各环境的实例统计，以及北极星服务全量实例数。
// 未部署环境直接返回全 0；部署记录 / Pod / 北极星查询失败则整体报错。
func (c *Collector) Collect(
	ctx context.Context,
	appID string,
	config *polaris.PolarisConfig,
) (*Result, error) {
	envNames := envNames(config)
	envStats := make(map[string]Stats, len(envNames))

	// 同一配置下各环境共享同一个北极星服务，实例列表只需拉取一次
	var (
		instances  []*polarisInfra.Instance
		loadedInst bool
	)

	for _, envName := range envNames {
		// 未部署：没有 AppliedFields，不打集群 / 北极星，固定返回 0
		if !config.GetEnvState(envName).IsDeployed() {
			envStats[envName] = Stats{}
			continue
		}

		// 仅统计主部署（空泳道），与实例列表默认视图一致
		record, err := c.deployRecordStore.GetLatest(ctx, appID, envName, "")
		if err != nil {
			return nil, errors.Wrapf(err, "get latest deploy record for env %s", envName)
		}
		podIPs, err := listPodIPs(ctx, record)
		if err != nil {
			return nil, errors.Wrapf(err, "list pod IPs for env %s", envName)
		}

		if !loadedInst {
			instances, err = polarisInfra.GetInstances(ctx, config.PolarisNamespace, config.PolarisName)
			if err != nil {
				return nil, errors.Wrapf(
					err,
					"get polaris instances for service %s/%s",
					config.PolarisNamespace,
					config.PolarisName,
				)
			}
			loadedInst = true
		}

		// 用本环境 Pod IP 集合从全量北极星实例中筛出属于该环境的子集
		envStats[envName] = CountMatched(podIPs, config.ServicePort, instances)
	}

	return &Result{
		EnvStats: envStats,
		PolarisInstanceCount: lo.CountBy(instances, func(inst *polarisInfra.Instance) bool {
			return inst != nil
		}),
	}, nil
}

// CountMatched 按 Pod IP + 服务端口匹配北极星实例并统计。
//
// 匹配规则（与实例列表 MergePolarisInfoToAppInstances 一致）：
//  1. 实例 IP 落在本环境 Pod IP 集合中
//  2. 实例 Port 等于配置的 ServicePort
//
// healthy / isolated 按字段独立计数，可重叠。
func CountMatched(
	podIPs map[string]struct{},
	servicePort int32,
	instances []*polarisInfra.Instance,
) Stats {
	matched := lo.Filter(instances, func(inst *polarisInfra.Instance, _ int) bool {
		if inst == nil {
			return false
		}
		_, ok := podIPs[inst.IP]
		// 与 MergePolarisInfoToAppInstances 一致：经 int64 比较端口
		return ok && int64(inst.Port) == int64(servicePort)
	})
	return Stats{
		TotalInstanceCount: len(matched),
		HealthyInstanceCount: lo.CountBy(matched, func(inst *polarisInfra.Instance) bool {
			return inst.IsHealthy
		}),
		IsolatedInstanceCount: lo.CountBy(matched, func(inst *polarisInfra.Instance) bool {
			return inst.IsIsolated
		}),
	}
}

// envNames 返回需要统计的环境集合：scopeEnvNames ∪ EnvStates keys（与 envStates 展示范围一致）。
func envNames(config *polaris.PolarisConfig) []string {
	return lo.Uniq(append(config.ScopeEnvNames, lo.Keys(config.EnvStates)...))
}

// listPodIPs 根据主部署记录拉取该环境全部 Pod，提取 podIP 集合用于后续匹配。
func listPodIPs(ctx context.Context, record *appmodeldeploy.Record) (map[string]struct{}, error) {
	client := k8sclient.NewWithGVR(cluster.NewConfig(record.ClusterID), gvr.Po)
	labelSelector := labels.SelectorFromSet(record.LabelSelector).String()
	pods, err := client.List(ctx, record.Namespace, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"list namespace %s labelsSelector [%s] pods",
			record.Namespace,
			labelSelector,
		)
	}

	ips := lo.FilterMap(pods.Items, func(pod unstructured.Unstructured, _ int) (string, bool) {
		ip := mapx.GetStr(pod.Object, "status.podIP")
		return ip, ip != ""
	})
	return lo.SliceToMap(ips, func(ip string) (string, struct{}) {
		return ip, struct{}{}
	}), nil
}
