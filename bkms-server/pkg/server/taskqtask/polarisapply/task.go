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
package polarisapply

import (
	"context"
	"fmt"

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
// 只携带业务主键；handler 执行时重新读取最新配置。
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
	).OnExhausted(handleExhausted)

	appStore        bkmsapp.ApplicationStore
	configStore     polaris.PolarisConfigStore
	envStore        bkmsenv.EnvironmentStore
	appModelStore   appmodel.AppModelStore
	envVarsReader   *envvars.UnifiedEnvVarsReader
	envStateManager *polaris.PolarisEnvStateManager
	crApplier       = polaris.NewCRApplier()
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
	appStore = applicationStore
	configStore = polarisConfigStore
	envStore = environmentStore
	appModelStore = modelStore
	envVarsReader = envvars.NewUnifiedEnvVarsReader(
		scopedEnvVarStore,
		depenvvars.NewReader(depSvcInstStore),
		polarisenvvars.NewReader(polarisConfigStore),
	)
	envStateManager = polaris.NewPolarisEnvStateManager(polarisConfigStore)
}

// Enqueue 为单个环境投递一条动态下发任务。
func Enqueue(ctx context.Context, appID, configName, envName string) error {
	err := taskq.Enqueue(
		ctx,
		DynamicApplyTask.NewTask(Args{
			AppID:      appID,
			ConfigName: configName,
			EnvName:    envName,
		}),
		asynq.TaskID(dynamicApplyTaskID(appID, configName, envName)),
	)
	if errors.Is(err, asynq.ErrTaskIDConflict) {
		return nil
	}
	return err
}

func dynamicApplyTaskID(appID, configName, envName string) string {
	return dynamicApplyTaskName + ":" + appID + ":" + configName + ":" + envName
}

func handle(ctx context.Context, args Args) error {
	if appStore == nil {
		return errors.Wrap(taskq.ErrStopRetry, "polaris dynamic apply is not initialized")
	}

	app, err := appStore.GetApp(ctx, args.AppID)
	if err != nil {
		if errors.Is(err, bkmsapp.ErrAppNotFound) {
			return errors.Wrapf(taskq.ErrStopRetry, "app %s not found", args.AppID)
		}
		return fail(ctx, args, err)
	}

	config, err := configStore.Get(ctx, args.AppID, args.ConfigName)
	if err != nil {
		if errors.Is(err, polaris.ErrConfigNotFound) {
			return errors.Wrapf(taskq.ErrStopRetry, "polaris config %s/%s not found", args.AppID, args.ConfigName)
		}
		return fail(ctx, args, err)
	}
	if !envStateManager.IsEnvReadyForDynamicApply(config, args.EnvName) {
		return errors.Wrapf(taskq.ErrStopRetry, "env %s is not ready for dynamic apply", args.EnvName)
	}

	appModel, err := appModelStore.GetAppModel(ctx, app.ID)
	if err != nil {
		return fail(ctx, args, errors.Wrap(err, "get app model for polaris CR apply"))
	}

	env, err := envStore.GetByName(ctx, app.WorkspaceID, app.ID, args.EnvName)
	if err != nil {
		return fail(ctx, args, errors.Wrapf(err, "get env %s", args.EnvName))
	}
	envVars, err := envVarsReader.ListVars(ctx, *env, app, appModel)
	if err != nil {
		return fail(ctx, args, errors.Wrapf(err, "build env vars for %s", args.EnvName))
	}
	if err = crApplier.Apply(ctx, app, env, config, envVars.ToMap()); err != nil {
		return fail(ctx, args, err)
	}

	recordResult(ctx, args.AppID, args.ConfigName, args.EnvName, nil)
	return nil
}

// fail 记录本次失败的重试进度到 LastError，并返回原错误供 asynq 重试。
func fail(ctx context.Context, args Args, applyErr error) error {
	attempt, total := retryProgress(ctx)
	recordResult(
		ctx, args.AppID, args.ConfigName, args.EnvName,
		errors.New(formatLastError(applyErr, attempt, total, false)),
	)
	return applyErr
}

// handleExhausted 在重试耗尽时覆盖 LastError，追加 exhausted 标记。
func handleExhausted(ctx context.Context, args Args, lastErr error) {
	if envStateManager == nil {
		return
	}
	attempt, total := retryProgress(ctx)
	recordResult(
		ctx, args.AppID, args.ConfigName, args.EnvName,
		errors.New(formatLastError(lastErr, attempt, total, true)),
	)
}

func recordResult(ctx context.Context, appID, configName, envName string, applyErr error) {
	if err := envStateManager.RecordDynamicApplyResult(
		ctx, appID, configName, envName, applyErr,
	); err != nil {
		log.Errorf(ctx, "record polaris CR apply result failed, app=%s config=%s env=%s: %v",
			appID, configName, envName, err)
	}
}

func retryProgress(ctx context.Context) (attempt, total int) {
	retried, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)
	return retried + 1, maxRetry + 1
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
