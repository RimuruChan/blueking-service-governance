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

package overview

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	deploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/status"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/gpa"
	workloadappmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	resourcessection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/resources"
)

// defaultTrafficLaneName 默认（基线）泳道名称；空字符串表示基线泳道。
// 部署总览只统计该泳道，不展示非基线泳道的部署状态与实例。
const defaultTrafficLaneName = ""

// Service 组装应用部署总览。
type Service struct {
	envStore            envmodel.EnvironmentStore
	appSpecStore        appspec.AppSpecStore
	appModelStore       workloadappmodel.AppModelStore
	gpaConfigStore      gpa.GPAConfigStore
	deployStatusService *deploystatus.DeployStatusService
}

// NewService 创建部署总览 Service。
func NewService(
	envStore envmodel.EnvironmentStore,
	appStore bkmsapp.ApplicationStore,
	appSpecStore appspec.AppSpecStore,
	appModelStore workloadappmodel.AppModelStore,
	buildAutoDeployRecordStore autodeploy.RecordStore,
	appModelDeployRecordStore appmodel.RecordStore,
	helmDeployRecordStore helm.RecordStore,
	gpaConfigStore gpa.GPAConfigStore,
) *Service {
	return &Service{
		envStore:       envStore,
		appSpecStore:   appSpecStore,
		appModelStore:  appModelStore,
		gpaConfigStore: gpaConfigStore,
		deployStatusService: deploystatus.NewDeployStatusService(
			appStore,
			envStore,
			buildAutoDeployRecordStore,
			appModelDeployRecordStore,
			helmDeployRecordStore,
		),
	}
}

// Build 组装 trpc/taf 应用的部署总览。
//
// 行集合与 env.AppIDs 对齐；仅统计默认（基线）泳道。
//
// Args:
//   - application 目标应用，须为 AppModel 类型（trpc/taf）
//
// Returns:
//   - 部署总览结果
//   - error
func (s *Service) Build(ctx context.Context, application *bkmsapp.Application) (*Result, error) {
	if !bkmsapp.IsAppModelType(application.Type) {
		return nil, errors.Errorf("unsupported app type: %s", application.Type)
	}

	trackedEnvs, err := s.listTrackedEnvs(ctx, application)
	if err != nil {
		return nil, err
	}

	autoscalingByEnv, err := s.listAutoscalingEnabledByEnv(ctx, application.ID)
	if err != nil {
		return nil, err
	}

	defaultResources, envOverrideResources, err := s.loadAppResourceSpecs(ctx, application.ID)
	if err != nil {
		return nil, err
	}

	// 批量结果可能含已不在 AppIDs 中的历史环境，assembleEnvRows 只按 trackedEnvs 取用。
	statusesByEnv, deployByEnv, err := s.deployStatusService.ListLatestByAppLane(
		ctx, application.ID, application.Type, defaultTrafficLaneName,
	)
	if err != nil {
		return nil, errors.Wrap(err, "list latest deploy statuses")
	}

	rows, recordsForInstances := assembleEnvRows(
		trackedEnvs, autoscalingByEnv, defaultResources, envOverrideResources, statusesByEnv, deployByEnv,
	)
	instanceCounts := queryInstanceCounts(ctx, recordsForInstances)
	for i := range rows {
		if c, ok := instanceCounts[rows[i].EnvName]; ok {
			rows[i].Instances = c
		}
	}

	return &Result{Envs: rows}, nil
}

// listTrackedEnvs 返回应出现在总览表格中的环境（AppIDs 含该应用）。
func (s *Service) listTrackedEnvs(
	ctx context.Context,
	application *bkmsapp.Application,
) ([]envmodel.Environment, error) {
	envs, err := s.envStore.ListBatchAppEnvs(ctx, application.WorkspaceID, []string{application.ID})
	if err != nil {
		return nil, errors.Wrap(err, "list app environments")
	}
	return lo.Filter(envs, func(env envmodel.Environment, _ int) bool {
		return lo.Contains(env.AppIDs, application.ID)
	}), nil
}

// listAutoscalingEnabledByEnv 返回各环境是否开启 GPA；无配置的环境视为 false。
func (s *Service) listAutoscalingEnabledByEnv(ctx context.Context, appID string) (map[string]bool, error) {
	out := map[string]bool{}
	if s.gpaConfigStore == nil {
		return out, nil
	}
	configs, err := s.gpaConfigStore.ListByApp(ctx, appID)
	if err != nil {
		return nil, errors.Wrap(err, "list gpa configs")
	}
	for _, cfg := range configs {
		if cfg != nil {
			out[cfg.EnvName] = cfg.Enabled
		}
	}
	return out, nil
}

// loadAppResourceSpecs 一次拉取应用全部 AppSpec，拆出默认 resources 与各环境覆盖。
// 若尚无 default 文档，则按 appspec.GetDefault 语义从 AppModel 懒初始化并落库。
func (s *Service) loadAppResourceSpecs(
	ctx context.Context,
	appID string,
) (*appspec.ResourcesSpec, map[string]*appspec.ResourcesSpec, error) {
	specs, err := s.appSpecStore.ListByApp(ctx, appID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "list app specs")
	}

	var defaultResources *appspec.ResourcesSpec
	defaultFound := false
	overrides := make(map[string]*appspec.ResourcesSpec, len(specs))
	for _, spec := range specs {
		if spec == nil {
			continue
		}
		if spec.EnvName == appspec.DefaultEnvName {
			defaultFound = true
			defaultResources = resourcessection.Clone(spec.Resources)
			continue
		}
		if spec.Resources != nil {
			overrides[spec.EnvName] = resourcessection.Clone(spec.Resources)
		}
	}

	if !defaultFound {
		defaultSpec, gErr := appspec.GetDefault(ctx, s.appSpecStore, s.appModelStore, appID)
		if gErr != nil {
			return nil, nil, errors.Wrap(gErr, "get default app spec")
		}
		defaultResources = resourcessection.Clone(defaultSpec.Resources)
	}

	return defaultResources, overrides, nil
}

// assembleEnvRows 在内存中组装表格行，并收集有 AppModel 部署记录的环境供后续查 K8s 实例。
//
// resources 取「默认 + 环境覆盖」合并后的生效值。
// 部署状态 / 部署记录 map 可能含非 tracked 环境，此处只按 trackedEnvs 取用。
func assembleEnvRows(
	trackedEnvs []envmodel.Environment,
	autoscalingByEnv map[string]bool,
	defaultResources *appspec.ResourcesSpec,
	envOverrideResources map[string]*appspec.ResourcesSpec,
	statusesByEnv map[string]*deploystatus.LatestDeployStatus,
	deployByEnv map[string]*appmodel.Record,
) ([]EnvRow, []deployRecordForEnv) {
	rows := make([]EnvRow, 0, len(trackedEnvs))
	records := make([]deployRecordForEnv, 0, len(trackedEnvs))

	for i := range trackedEnvs {
		env := &trackedEnvs[i]
		row := EnvRow{
			EnvID:              env.ID.Hex(),
			EnvName:            env.Name,
			EnvDisplayName:     env.DisplayName,
			EnvType:            env.Type,
			EnvKind:            string(env.GetKind()),
			DeployStatus:       deploystatus.StatusUnknown,
			AutoscalingEnabled: autoscalingByEnv[env.Name],
			Resources: toAPIResourceSpec(
				resourcessection.Merge(defaultResources, envOverrideResources[env.Name]),
			),
		}
		if latest := statusesByEnv[env.Name]; latest != nil {
			row.DeployStatus = latest.Status
			if !latest.StartedAt.IsZero() {
				t := latest.StartedAt
				row.LastDeployStartedAt = &t
			}
		}
		if rec := deployByEnv[env.Name]; rec != nil {
			records = append(records, deployRecordForEnv{EnvName: env.Name, Record: rec})
		}
		rows = append(rows, row)
	}
	return rows, records
}

// toAPIResourceSpec 将 AppSpec resources 转为 API 层结构；nil 字段输出为空字符串。
func toAPIResourceSpec(spec *appspec.ResourcesSpec) ResourceSpec {
	if spec == nil {
		return ResourceSpec{}
	}
	out := ResourceSpec{}
	if spec.CPULimits != nil {
		out.CPULimits = *spec.CPULimits
	}
	if spec.CPURequests != nil {
		out.CPURequests = *spec.CPURequests
	}
	if spec.MemoryLimits != nil {
		out.MemoryLimits = *spec.MemoryLimits
	}
	if spec.MemoryRequests != nil {
		out.MemoryRequests = *spec.MemoryRequests
	}
	return out
}
