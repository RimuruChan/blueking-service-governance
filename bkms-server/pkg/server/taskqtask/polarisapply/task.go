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

// Package polarisapply 实现北极星配置动态下发的 asynq 任务。
//
// 一条消息对应一个环境的 PolarisConfig CR Upsert。投递侧调用 Enqueue；
// 消费侧在 taskqtask.Setup 中调用一次 Init 后挂载 DynamicApplyTask。
// 任务包只负责队列适配和重试策略，查询、渲染及版本校验由 Polaris 服务完成。
package polarisapply

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	depsvcmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

const (
	dynamicApplyTaskName = "polaris.dynamic_apply"
	dynamicApplyMaxRetry = 10
)

// Args 是单个环境下发 PolarisConfig CR 的任务参数。
// 只携带业务主键，避免把渲染结果和配置快照放进队列；handler 执行时重新读取最新数据。
type Args struct {
	AppID      string `json:"appID"`
	ConfigName string `json:"configName"`
	EnvName    string `json:"envName"`
}

var (
	// DynamicApplyTask 向单个目标环境下发 PolarisConfig CR。
	DynamicApplyTask = taskq.NewTaskType(
		dynamicApplyTaskName,
		handle,
		asynq.MaxRetry(dynamicApplyMaxRetry),
	)

	dynamicApplyService *polaris.DynamicApplyService
	envStateManager     *polaris.PolarisEnvStateManager
)

// Init 注入动态下发所需的 Store。由 worker 在 taskqtask.Setup 中调用一次。
func Init(
	applicationStore bkmsapp.ApplicationStore,
	polarisConfigStore polaris.PolarisConfigStore,
	environmentStore bkmsenv.EnvironmentStore,
	modelStore appmodel.AppModelStore,
	scopedEnvVarStore envvars.ScopedEnvVarStore,
	depSvcInstStore depsvcmodel.ServiceInstanceStore,
) {
	reader := envvars.NewUnifiedEnvVarsReader(
		scopedEnvVarStore,
		depenvvars.NewReader(depSvcInstStore),
		polarisenvvars.NewReader(polarisConfigStore),
	)
	envStateManager = polaris.NewPolarisEnvStateManager(polarisConfigStore)
	dynamicApplyService = polaris.NewDynamicApplyService(
		applicationStore,
		polarisConfigStore,
		environmentStore,
		modelStore,
		reader,
		envStateManager,
	)
}

// Enqueue 为单个环境投递一条动态下发任务。
func Enqueue(ctx context.Context, appID, configName, envName string) error {
	return taskq.Enqueue(
		ctx,
		DynamicApplyTask.NewTask(Args{
			AppID:      appID,
			ConfigName: configName,
			EnvName:    envName,
		}),
	)
}

func handle(ctx context.Context, args Args) error {
	if dynamicApplyService == nil {
		return errors.Wrap(taskq.ErrStopRetry, "polaris dynamic apply is not initialized")
	}

	configUpdatedAt, err := dynamicApplyService.Apply(
		ctx,
		args.AppID,
		args.ConfigName,
		args.EnvName,
	)
	if err != nil {
		switch {
		case errors.Is(err, bkmsapp.ErrAppNotFound):
			return errors.Wrapf(taskq.ErrStopRetry, "app %s not found", args.AppID)
		case errors.Is(err, polaris.ErrConfigNotFound):
			return errors.Wrapf(
				taskq.ErrStopRetry,
				"polaris config %s/%s not found",
				args.AppID,
				args.ConfigName,
			)
		case errors.Is(err, appmodel.ErrAppModelNotFound):
			return errors.Wrap(taskq.ErrStopRetry, "app model not found")
		case errors.Is(err, bkmsenv.ErrEnvNotFound):
			return errors.Wrapf(taskq.ErrStopRetry, "env %s not found", args.EnvName)
		case errors.Is(err, polaris.ErrDynamicApplyNotReady):
			return errors.Wrap(taskq.ErrStopRetry, err.Error())
		default:
			return fail(ctx, args, configUpdatedAt, err)
		}
	}

	recordResult(ctx, args, configUpdatedAt, nil)
	return nil
}

// fail 记录重试进度，并在最后一次尝试时标记 exhausted。
// 结果在此处写入，是因为 exhausted 回调只有主键，无法安全确定配置版本。
func fail(ctx context.Context, args Args, configUpdatedAt time.Time, applyErr error) error {
	attempt, total, exhausted := retryProgress(ctx)
	recordResult(
		ctx,
		args,
		configUpdatedAt,
		errors.New(formatLastError(applyErr, attempt, total, exhausted)),
	)
	return applyErr
}

func recordResult(ctx context.Context, args Args, configUpdatedAt time.Time, applyErr error) {
	if envStateManager == nil {
		return
	}
	if err := envStateManager.RecordDynamicApplyResult(
		ctx,
		args.AppID,
		args.ConfigName,
		args.EnvName,
		configUpdatedAt,
		applyErr,
	); err != nil {
		log.Errorf(ctx, "record polaris CR apply result failed, app=%s config=%s env=%s: %v",
			args.AppID, args.ConfigName, args.EnvName, err)
	}
}

func retryProgress(ctx context.Context) (attempt, total int, exhausted bool) {
	retried, retryCountOK := asynq.GetRetryCount(ctx)
	maxRetry, maxRetryOK := asynq.GetMaxRetry(ctx)
	return retried + 1, maxRetry + 1, retryCountOK && maxRetryOK && retried >= maxRetry
}

func formatLastError(err error, attempt, total int, exhausted bool) string {
	base := ""
	if err != nil {
		base = err.Error()
	}
	if exhausted {
		return fmt.Sprintf("%s (retry %d/%d, retries exhausted)", base, attempt, total)
	}
	return fmt.Sprintf("%s (retry %d/%d)", base, attempt, total)
}
