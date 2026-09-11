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

// Package helm release.go 提供基于 Helm SDK 的 Release 原子化查询能力
package helm

import (
	"strconv"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	helmrelease "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
)

// GetReleaseStatus 获取 Release 详细状态
func GetReleaseStatus(cfg *action.Configuration, releaseName string) (*Release, error) {
	statusAction := action.NewStatus(cfg)
	release, err := statusAction.Run(releaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "get release %s status", releaseName)
	}

	return newReleaseFromHelm(release), nil
}

// GetReleaseByChart 在当前 namespace 中按 Chart 元数据名称筛选未卸载的 Release。
// 优先匹配 preferredName，否则选择 Release 名称字典序最小的候选。
// 保留失败、pending 和 unknown 状态，由调用方判断是否可用。
func GetReleaseByChart(cfg *action.Configuration, chartName, preferredName string) (*Release, error) {
	releases, err := ListReleases(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "list releases for chart %s", chartName)
	}
	release := FindReleaseByChart(releases, chartName, preferredName)
	if release == nil {
		return nil, errors.Wrapf(driver.ErrReleaseNotFound, "no installed release for chart %s", chartName)
	}
	return release, nil
}

// ListReleases 查询当前 namespace 中各未卸载 Release 的最新版本，包含状态和 Values。
func ListReleases(cfg *action.Configuration) ([]*Release, error) {
	list := action.NewList(cfg)
	list.StateMask = (action.ListAll | action.ListUnknown) &^ action.ListUninstalled
	releases, err := list.Run()
	if err != nil {
		return nil, errors.Wrap(err, "list helm releases")
	}
	result := make([]*Release, 0, len(releases))
	for _, release := range releases {
		result = append(result, newReleaseFromHelm(release))
	}
	return result, nil
}

// FindReleaseByChart 从列表中匹配 Chart，同名优先，否则按名称字典序选择；无匹配时返回 nil。
func FindReleaseByChart(releases []*Release, chartName, preferredName string) *Release {
	var matched *Release
	for _, release := range releases {
		if release.Chart.Name == "" || release.Chart.Name != chartName {
			continue
		}
		if release.Name == preferredName {
			return release
		}
		if matched == nil || release.Name < matched.Name {
			matched = release
		}
	}
	return matched
}

func newReleaseFromHelm(release *helmrelease.Release) *Release {
	result := &Release{
		Name:      release.Name,
		Namespace: release.Namespace,
		Version:   strconv.Itoa(release.Version),
		Values:    release.Config,
		DeployResult: DeployResult{
			Status:      release.Info.Status,
			Description: release.Info.Description,
			CreatedAt:   release.Info.LastDeployed.String(),
		},
	}
	if release.Chart != nil && release.Chart.Metadata != nil {
		result.Chart = Chart{
			Name:        release.Chart.Metadata.Name,
			Version:     release.Chart.Metadata.Version,
			AppVersion:  release.Chart.Metadata.AppVersion,
			Description: release.Chart.Metadata.Description,
		}
	}
	return result
}

// GetReleaseValues 获取 Release 当前使用的 Values
// revision 为 0 时获取最新版本的 Values
func GetReleaseValues(cfg *action.Configuration, releaseName string, revision int) (map[string]any, error) {
	getValues := action.NewGetValues(cfg)
	if revision > 0 {
		getValues.Version = revision
	}
	values, err := getValues.Run(releaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "get release %s values (revision=%d)", releaseName, revision)
	}
	return values, nil
}

// GetReleaseManifest 获取 Release 的 Manifest（包含所有已部署资源的 YAML）
func GetReleaseManifest(cfg *action.Configuration, releaseName string) (string, error) {
	getAction := action.NewGet(cfg)
	release, err := getAction.Run(releaseName)
	if err != nil {
		return "", errors.Wrapf(err, "get release %s manifest", releaseName)
	}
	return release.Manifest, nil
}
