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

package clusteraddon

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/storage/driver"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
)

// GetSupportedActions 根据当前安装状态返回支持的操作列表
func GetSupportedActions(status AddonStatus) []string {
	switch status {
	case "", helm.StatusUninstalled, helm.StatusNotFound:
		return []string{"install"}
	case helm.StatusDeployed:
		return []string{"upgrade", "uninstall"}
	case helm.StatusFailed:
		return []string{"install", "uninstall"}
	default:
		// pending-xxx 等中间状态，不允许操作
		return nil
	}
}

// FillAvailableVersions 从仓库 Index 填充可用版本列表
func (c *HelmChartInfo) FillAvailableVersions(repoIndex *RepoIndex) {
	versions := repoIndex.ListChartVersions(c.ChartName)
	c.AvailableVersions = versions
	if len(versions) > 0 {
		// 使用仓库中最新版本作为默认版本
		c.DefaultChartVersion = versions[0]
	}
}

// FillClusterStatus 从集群实时查询 Helm Release 状态并填充到 ClusterAddonInfo
// NOTE: 目前是对各个插件单独判断，实际上大部分插件会安装在同一个 namespace 下，
// 后续考虑是否要优化为 list namespace 下的所有 helm release 以后集中判断
func (info *ClusterAddonInfo) FillClusterStatus(ctx context.Context, clusterID string, addonDef *ClusterAddonDef) {
	defer func() {
		info.SupportedActions = GetSupportedActions(info.InstallInfo.Status)
	}()

	releaseName := GenerateReleaseName(addonDef)
	cfg, release, err := QueryAddonStatus(ctx, clusterID, info.InstallInfo.Namespace, addonDef)
	if err != nil {
		log.Warnf(ctx, "query addon status: %v", err)
		info.InstallInfo.Status = helm.StatusUnknown
		info.InstallInfo.Message = err.Error()
		return
	}

	// 填充状态信息
	info.InstallInfo.Status = release.DeployResult.Status
	info.InstallInfo.Message = release.DeployResult.Description
	info.InstallInfo.CurrentChartVersion = release.Chart.Version
	if info.InstallInfo.Status == helm.StatusNotFound {
		return
	}

	// 获取当前 Values
	values, vErr := helm.GetReleaseValues(cfg, releaseName, 0)
	if vErr != nil {
		log.Warnf(ctx, "get release values for %s : %v", releaseName, vErr)
		info.InstallInfo.Message = fmt.Sprintf("get release values failed: %v", vErr)
		return
	}
	if len(values) > 0 {
		yamlData, mErr := yaml.Marshal(values)
		if mErr == nil {
			info.InstallInfo.CurrentValues = string(yamlData)
		}
	}
}

// BuildAddonInfoList 根据插件定义列表、仓库 Index 和集群状态，构建完整的 ClusterAddonInfo 列表
func BuildAddonInfoList(
	ctx context.Context,
	addonDefs []*ClusterAddonDef,
	env *envmodel.Environment,
	namespace string,
	repoIndex *RepoIndex,
) []*ClusterAddonInfo {
	var addons []*ClusterAddonInfo
	for _, addonDef := range ApplicableAddonDefs(addonDefs, env) {
		ns := addonDef.GetNamespace(namespace)
		info := NewAddonInfoFromDef(addonDef, ns)
		info.ChartInfo.FillAvailableVersions(repoIndex)
		info.FillClusterStatus(ctx, env.Cluster.ClusterID, addonDef)
		addons = append(addons, info)
	}

	return addons
}

// QueryAddonStatus 查询组件 Release 状态和元数据，返回的 Helm 配置可继续用于读取 Values。
func QueryAddonStatus(
	ctx context.Context, clusterID, namespace string, def *ClusterAddonDef,
) (*action.Configuration, *helm.Release, error) {
	releaseName := GenerateReleaseName(def)
	debugLog := helm.NewHelmDebugLogger(ctx, releaseName, "query-addon-status")
	cfg, err := helm.NewActionConfiguration(clusterID, namespace, debugLog)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "init helm configuration for addon %s in cluster %s namespace %s",
			def.Name, clusterID, namespace)
	}
	release, err := helm.GetReleaseStatus(cfg, releaseName)
	if errors.Is(err, driver.ErrReleaseNotFound) {
		return cfg, &helm.Release{
			Name: releaseName, Namespace: namespace,
			DeployResult: helm.DeployResult{Status: helm.StatusNotFound},
		}, nil
	}
	if err != nil {
		return nil, nil, errors.Wrapf(
			err,
			"query addon %s status in cluster %s namespace %s",
			def.Name,
			clusterID,
			namespace,
		)
	}
	if release.DeployResult.Status == helm.StatusUnknown {
		return nil, nil, errors.Errorf(
			"addon %s release status is unknown in cluster %s namespace %s", def.Name, clusterID, namespace,
		)
	}
	return cfg, release, nil
}

// ApplicableAddonDefs 返回适用于环境集群的组件定义，供列表展示和部署检查共用。
func ApplicableAddonDefs(defs []*ClusterAddonDef, env *envmodel.Environment) []*ClusterAddonDef {
	return lo.Filter(defs, func(def *ClusterAddonDef, _ int) bool {
		return def.IsApplicableToEnv(env)
	})
}

// InspectRequiredAddons 查询必选组件状态。缺失作为检查结果，查询失败作为错误返回。
func InspectRequiredAddons(
	ctx context.Context,
	defs []*ClusterAddonDef,
	appType string,
	env *envmodel.Environment,
	namespace string,
) ([]AddonReference, error) {
	var missing []AddonReference
	for _, def := range ApplicableAddonDefs(defs, env) {
		if !lo.Contains(def.RequiredForAppTypes, appType) {
			continue
		}
		addon := AddonReference{Name: def.Name, DisplayName: def.DisplayName}
		_, release, err := QueryAddonStatus(ctx, env.Cluster.ClusterID, def.GetNamespace(namespace), def)
		if err != nil {
			return nil, errors.Wrap(err, "inspect required cluster addons")
		}
		if release.DeployResult.Status != helm.StatusDeployed {
			missing = append(missing, addon)
		}
	}
	return missing, nil
}
