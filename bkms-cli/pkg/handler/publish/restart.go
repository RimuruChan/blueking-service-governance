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

package publish

import (
	"context"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/clierr"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/params"
)

const (
	// restartAction 重启记录的操作类型
	restartAction = "restart"
	// recordStatusSuccess 记录成功状态
	recordStatusSuccess = "success"
	// recordStatusFailed 记录失败状态
	recordStatusFailed = "failed"

	// defaultRestartPollInterval 轮询重启记录的间隔
	defaultRestartPollInterval = 2 * time.Second
	// defaultRestartWaitTimeout 等待重启完成的最长时间
	defaultRestartWaitTimeout = 3 * time.Minute
	// restartRecordPageSize 轮询时每页拉取的记录数
	restartRecordPageSize = int64(100)
	// restartRecordChunkSize 按记录 ID 查询时单次携带的 ID 数量
	restartRecordChunkSize = 100
)

// BuildRestartOptions 根据命令行参数（逗号分隔）构造进程重启请求参数：
//   - --env 与 --env-type 互斥，必须且只能指定其中一个
//   - --instance-ids 仅可与单个 --env 一起使用
func BuildRestartOptions(envs, envTypes, instanceIDs string) (client.DevModeRestartOptions, error) {
	envNames := lo.Uniq(params.NormalizeInstIDs(envs, ","))
	types := lo.Uniq(params.NormalizeInstIDs(envTypes, ","))
	instances := lo.Uniq(params.NormalizeInstIDs(instanceIDs, ","))

	switch {
	case len(envNames) > 0 && len(types) > 0:
		return client.DevModeRestartOptions{}, clierr.Usagef("--env and --env-type cannot be used together")
	case len(envNames) == 0 && len(types) == 0:
		return client.DevModeRestartOptions{}, clierr.Usagef("one of --env or --env-type is required")
	case len(instances) > 0 && len(envNames) != 1:
		return client.DevModeRestartOptions{}, clierr.Usagef("--instance-ids can only be used with a single --env")
	}

	switch {
	case len(instances) > 0:
		return client.DevModeRestartOptions{
			Instances: &client.DevModeRestartInstancesTarget{EnvName: envNames[0], InstanceIDs: instances},
		}, nil
	case len(envNames) > 0:
		return client.DevModeRestartOptions{Envs: &client.DevModeRestartEnvsTarget{Names: envNames}}, nil
	default:
		return client.DevModeRestartOptions{EnvTypes: &client.DevModeRestartEnvTypesTarget{Types: types}}, nil
	}
}

// Restart 提交开发模式进程重启（异步），并按同一条件轮询记录直到全部实例进入终态。
// 服务端受理后立即返回，重启任务在队列中执行；Ctrl+C 只停止等待，不影响服务端任务。
func Restart(ctx context.Context, cli client.Client, appID string, opts client.DevModeRestartOptions) error {
	return restartAndWait(ctx, cli, appID, opts, defaultRestartPollInterval, defaultRestartWaitTimeout)
}

// restartAndWait 提交重启并轮询，pollInterval / waitTimeout 便于测试注入
func restartAndWait(
	ctx context.Context,
	cli client.Client,
	appID string,
	opts client.DevModeRestartOptions,
	pollInterval, waitTimeout time.Duration,
) error {
	data, err := cli.RestartDevModeProcesses(ctx, appID, opts)
	if err != nil {
		return errors.Wrapf(err, "restart dev mode processes for app %s", appID)
	}
	console.Info("Restart request submitted, %d instance(s) accepted.", data.InstanceCount)
	for _, env := range data.Skipped {
		console.Warn("  - skipped %s (%s): %s", env.EnvName, env.EnvType, env.Reason)
	}
	if len(data.RecordIDs) == 0 {
		return errors.New("server returned no restart records")
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	deadline := time.Now().Add(waitTimeout)
	lastFinished := -1
	for {
		// 只需知道还有多少实例未完成，按状态过滤并只取 1 条，避免每次拉取全部分页
		inProgress, err := countRestartInProgress(ctx, cli, appID, data.RecordIDs)
		if err != nil {
			return err
		}
		finished := lo.Max([]int{len(data.RecordIDs) - int(inProgress), 0})
		if finished != lastFinished {
			console.Info("Waiting for instances to finish: %d/%d", finished, len(data.RecordIDs))
			lastFinished = finished
		}

		if inProgress == 0 {
			records, listErr := listRestartRecords(ctx, cli, appID, data.RecordIDs)
			if listErr != nil {
				return listErr
			}
			terminal := lo.CountBy(records, func(record client.DevModePublishRecord) bool {
				return isTerminalStatus(record.Status)
			})
			if len(records) >= len(data.RecordIDs) && terminal == len(records) {
				return printRestartResult(records)
			}
		}

		if time.Now().After(deadline) {
			return errors.Errorf(
				"timed out after %s waiting for restart to finish; the task is still running on the server, "+
					"check 'bkms-cli app publish history --app %s' for the final result",
				waitTimeout, appID,
			)
		}
		select {
		case <-ctx.Done():
			return errors.Wrap(
				ctx.Err(),
				"stopped watching; the restart keeps running on the server, "+
					"check 'bkms-cli app publish history' for the final result",
			)
		case <-ticker.C:
		}
	}
}

// countRestartInProgress 统计本次提交仍处于 pending / running 的记录数
func countRestartInProgress(
	ctx context.Context,
	cli client.Client,
	appID string,
	recordIDs []string,
) (int64, error) {
	var total int64
	for _, chunk := range lo.Chunk(recordIDs, restartRecordChunkSize) {
		_, count, err := cli.ListDevModePublishRecords(ctx, appID, client.DevModePublishRecordsQuery{
			RecordIDs: chunk,
			Statuses:  []string{"pending", "running"},
			Page:      1,
			PageSize:  1,
		})
		if err != nil {
			return 0, errors.Wrap(err, "query restart progress")
		}
		total += count
	}
	return total, nil
}

// listRestartRecords 按本次提交的记录 ID 拉取完整记录
func listRestartRecords(
	ctx context.Context,
	cli client.Client,
	appID string,
	recordIDs []string,
) ([]client.DevModePublishRecord, error) {
	var records []client.DevModePublishRecord
	for _, chunk := range lo.Chunk(recordIDs, restartRecordChunkSize) {
		batch, _, err := cli.ListDevModePublishRecords(ctx, appID, client.DevModePublishRecordsQuery{
			RecordIDs: chunk,
			Page:      1,
			PageSize:  restartRecordPageSize,
		})
		if err != nil {
			return nil, errors.Wrap(err, "list restart records")
		}
		records = append(records, lo.Filter(batch, func(record client.DevModePublishRecord, _ int) bool {
			return record.Action == restartAction
		})...)
	}
	return records, nil
}

// isTerminalStatus 判断记录是否已进入终态
func isTerminalStatus(status string) bool {
	return status == recordStatusSuccess || status == recordStatusFailed
}

// printRestartResult 按环境输出重启结果，任一实例失败时返回错误
func printRestartResult(records []client.DevModePublishRecord) error {
	byEnv := lo.GroupBy(records, func(record client.DevModePublishRecord) string { return record.EnvName })
	envNames := lo.Keys(byEnv)
	sort.Strings(envNames)

	failed := 0
	for _, envName := range envNames {
		console.Info("\nEnvironment: %s", envName)
		for _, record := range byEnv[envName] {
			if record.Status == recordStatusFailed {
				failed++
				console.Error("  [FAILED]  %s: %s", record.Instance, record.Message)
				continue
			}
			console.Info("  [SUCCESS] %s: %s", record.Instance, record.Message)
		}
	}

	console.Info("\n==================================================")
	if failed == 0 {
		console.Info("All %d instance(s) restarted successfully!", len(records))
	} else {
		console.Error("Restart finished with %d/%d failed instance(s).", failed, len(records))
	}
	console.Info("==================================================")

	if failed > 0 {
		return errors.Errorf("restart failed on %d instance(s)", failed)
	}
	return nil
}
