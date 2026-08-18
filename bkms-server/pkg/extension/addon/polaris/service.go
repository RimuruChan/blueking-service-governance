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

package polaris

import (
	"context"
	stderrors "errors"

	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

// DynamicApplyEnqueuer 为单个可动态下发环境投递 asynq 任务。
type DynamicApplyEnqueuer func(ctx context.Context, appID, configName, envName string) error

// PolarisConfigService 负责配置管理和北极星平台服务生命周期。
type PolarisConfigService struct {
	polarisConfigStore  PolarisConfigStore
	platformManager     *PolarisPlatformManager
	envStateManager     *PolarisEnvStateManager
	applier             *CRApplier
	envStore            bkmsenv.EnvironmentStore
	enqueueDynamicApply DynamicApplyEnqueuer
}

// NewPolarisConfigService 创建北极星配置服务。
func NewPolarisConfigService(
	store PolarisConfigStore,
	platformManager *PolarisPlatformManager,
	envStateManager *PolarisEnvStateManager,
	envStore bkmsenv.EnvironmentStore,
	enqueue DynamicApplyEnqueuer,
) *PolarisConfigService {
	return &PolarisConfigService{
		polarisConfigStore:  store,
		platformManager:     platformManager,
		envStateManager:     envStateManager,
		applier:             NewCRApplier(),
		envStore:            envStore,
		enqueueDynamicApply: enqueue,
	}
}

// Create 创建北极星配置，并按需创建北极星服务实例
func (s *PolarisConfigService) Create(
	ctx context.Context,
	app *bkmsapp.Application,
	config *PolarisConfig,
	createNewService bool,
) error {
	if createNewService {
		result, err := s.platformManager.CreateService(ctx, &CreatePolarisServiceParams{
			PolarisName:      config.PolarisName,
			PolarisNamespace: config.PolarisNamespace,
			Operator:         config.Operator,
			WorkspaceID:      app.WorkspaceID,
			AppID:            app.ID,
			ScopeEnvNames:    config.ScopeEnvNames,
		})
		if err != nil {
			return errors.Wrap(err, "create polaris service")
		}
		config.PolarisToken = result.Token
		config.DepSvcInstID = result.ServiceInstanceID
	}

	// 过滤掉 scope 外且未部署的环境权重，并为 scope 内未设置权重的环境补充默认值
	config.EnvWeights = s.envStateManager.reconcileEnvWeightsForScope(
		config.ScopeEnvNames, config.EnvWeights, nil,
	)

	return s.polarisConfigStore.Create(ctx, config)
}

// Update 更新北极星配置
func (s *PolarisConfigService) Update(
	ctx context.Context,
	app *bkmsapp.Application,
	oldConfig *PolarisConfig,
	updateData *ConfigUpdateData,
) (*PolarisConfig, error) {
	if updateData.ScopeEnvNames != nil {
		// scope 变化时保留仍有效的权重，并为新增环境补充默认值。
		updateData.envWeights = s.envStateManager.reconcileEnvWeightsForScope(
			updateData.ScopeEnvNames,
			oldConfig.EnvWeights,
			oldConfig.EnvStates,
		)
	}

	if err := s.polarisConfigStore.Update(ctx, app.ID, oldConfig.Name, updateData); err != nil {
		return nil, errors.Wrap(err, "update polaris config")
	}

	newConfig, err := s.polarisConfigStore.Get(ctx, app.ID, oldConfig.Name)
	if err != nil {
		return nil, errors.Wrap(err, "get updated polaris config")
	}
	envNames, err := s.envStateManager.PrepareDynamicApply(ctx, newConfig)
	if err != nil {
		return newConfig, errors.Wrap(err, "prepare dynamic polaris apply")
	}
	// 请求结束后仍需保证任务投递，避免客户端断开导致配置更新成功但任务丢失。
	if err = s.triggerDynamicApply(context.WithoutCancel(ctx), newConfig, envNames); err != nil {
		return newConfig, errors.Wrap(err, "enqueue polaris dynamic apply")
	}
	return newConfig, nil
}

// Delete 删除北极星配置，并按需删除北极星服务实例
func (s *PolarisConfigService) Delete(
	ctx context.Context,
	app *bkmsapp.Application,
	config *PolarisConfig,
) error {
	if !config.DepSvcInstID.IsZero() {
		if err := s.platformManager.DeleteService(ctx, &DeleteServiceParams{
			ServiceInstanceID: config.DepSvcInstID,
			AppID:             app.ID,
		}); err != nil {
			return errors.Wrap(err, "delete polaris service")
		}
	}
	if err := s.polarisConfigStore.Delete(ctx, app.ID, config.Name); err != nil {
		return errors.Wrap(err, "delete polaris config")
	}
	return nil
}

// triggerDynamicApply 为每个可下发环境投递一条 asynq 任务。
// 某个环境投递失败会写入该环境 LastError，不阻止其余环境入队；返回全部投递错误的汇总。
func (s *PolarisConfigService) triggerDynamicApply(
	ctx context.Context,
	config *PolarisConfig,
	envNames []string,
) error {
	var errs []error
	for _, envName := range envNames {
		if err := s.enqueueDynamicApply(ctx, config.AppID, config.Name, envName); err != nil {
			enqueueErr := errors.Wrapf(err, "enqueue polaris dynamic apply for env %s", envName)
			if recErr := s.envStateManager.RecordDynamicApplyResult(
				ctx, config.AppID, config.Name, envName, config.UpdatedAt, enqueueErr,
			); recErr != nil {
				log.Errorf(ctx, "record polaris enqueue failure failed, app=%s config=%s env=%s: %v",
					config.AppID, config.Name, envName, recErr)
			}
			errs = append(errs, enqueueErr)
		}
	}
	return stderrors.Join(errs...)
}

func (s *PolarisConfigService) patchEnvWeight(
	ctx context.Context,
	app *bkmsapp.Application,
	config *PolarisConfig,
	envName string,
	weight int32,
) error {
	env, err := s.envStore.GetByName(ctx, app.WorkspaceID, app.ID, envName)
	if err != nil {
		return errors.Wrapf(err, "get env %s", envName)
	}
	return s.applier.PatchWeight(ctx, app, env, config, weight)
}

// UpdateEnvWeight 更新指定环境的北极星实例权重；已部署环境会先同步 Patch 集群资源，成功后再持久化。
func (s *PolarisConfigService) UpdateEnvWeight(
	ctx context.Context,
	app *bkmsapp.Application,
	config *PolarisConfig,
	envName string,
	weight int32,
) (*PolarisConfig, error) {
	isDeployed := config.GetEnvState(envName).IsDeployed()
	if isDeployed {
		if err := s.patchEnvWeight(ctx, app, config, envName, weight); err != nil {
			log.Errorf(ctx, "patch polaris CR weight failed, app=%s config=%s env=%s: %v",
				app.ID, config.Name, envName, err)
			return nil, errors.Wrap(err, "patch env weight")
		}
	}

	if err := s.polarisConfigStore.UpsertEnvWeight(ctx, app.ID, config.Name, envName, weight); err != nil {
		if isDeployed {
			log.Errorf(ctx, "persist polaris env weight after cluster patch failed, app=%s config=%s env=%s: %v",
				app.ID, config.Name, envName, err)
		}
		return nil, errors.Wrap(err, "update env weight")
	}

	// 重新读取最新配置
	newConfig, err := s.polarisConfigStore.Get(ctx, app.ID, config.Name)
	if err != nil {
		return nil, errors.Wrap(err, "get updated polaris config")
	}

	return newConfig, nil
}
