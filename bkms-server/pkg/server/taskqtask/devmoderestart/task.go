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

// Package devmoderestart 实现开发模式进程重启的异步任务。
// 一条消息对应一次请求的全部实例，消息内按最大 10 个实例并行执行。
package devmoderestart

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/rest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/bcstoken"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

const (
	taskName = "devmode.restart"
	// maxConcurrency 单次重启任务内并行执行的实例数上限
	maxConcurrency = 10
)

// Instance 单个实例的重启参数
type Instance struct {
	// RecordID 发布记录 ID（hex）
	RecordID string `json:"recordID"`
	// EnvName 环境名称
	EnvName string `json:"envName"`
	// Instance 实例（pod）名称
	Instance string `json:"instance"`
	// ClusterID 实例所在集群 ID
	ClusterID string `json:"clusterID"`
	// Address 集群 API 完整地址
	Address string `json:"address"`
	// Namespace 实例所在命名空间
	Namespace string `json:"namespace"`
	// Container 业务容器名称
	Container string `json:"container"`
	// MountPath devmode 脚本挂载路径
	MountPath string `json:"mountPath"`
}

// Args 一次重启请求的任务参数，只携带主键与集群信息，不携带凭据。
type Args struct {
	// AppID 应用 ID
	AppID string `json:"appID"`
	// WorkspaceID 工作空间 ID，用于审计
	WorkspaceID string `json:"workspaceID"`
	// BCSProjectCode BCS 项目 Code，用于执行时获取用户 Token
	BCSProjectCode string `json:"bcsProjectCode"`
	// Instances 本次请求涉及的全部实例
	Instances []Instance `json:"instances"`
}

// Task 一次重启请求对应的任务。重启非幂等，失败不自动重试，由用户手动重新发起。
var Task = taskq.NewTaskType(taskName, handle, asynq.MaxRetry(0))

// Enqueue 投递一条重启任务，本次请求的所有实例都在同一条消息中
func Enqueue(ctx context.Context, args Args) error {
	return taskq.Enqueue(ctx, Task.NewTask(args))
}

// handle asynq 入口
func handle(ctx context.Context, args Args) error {
	return newRunner(storereg.G().DB).run(ctx, args)
}

// runner 执行一次重启请求并更新记录
type runner struct {
	records *devmode.PublishRecordStore
}

func newRunner(db *mongo.Database) *runner {
	return &runner{records: store.Of(db, devmode.PublishRecordMeta)}
}

func (r *runner) run(ctx context.Context, args Args) error {
	if len(args.Instances) == 0 {
		return nil
	}

	restarter := devmode.NewRestarter(r.records)
	token, err := bcstoken.FetchActiveUserToken(ctx, args.BCSProjectCode)
	if err != nil {
		for i := range args.Instances {
			r.markFailed(ctx, restarter, args.Instances[i].RecordID, err.Error())
		}
		return errors.Wrap(taskq.ErrStopRetry, err.Error())
	}

	results := make([]devmode.RestartInstanceResult, len(args.Instances))
	var g errgroup.Group
	g.SetLimit(maxConcurrency)
	for i := range args.Instances {
		g.Go(func() error {
			results[i] = r.restartInstance(
				ctx, restarter, token, args.AppID, args.WorkspaceID, args.Instances[i],
			)
			return nil
		})
	}
	_ = g.Wait()

	failed := lo.CountBy(results, func(result devmode.RestartInstanceResult) bool {
		return result.Status != devmode.RecordStatusSuccess
	})
	if failed > 0 {
		return errors.Wrapf(taskq.ErrStopRetry, "%d instance(s) failed", failed)
	}
	return nil
}

// restartInstance 执行单实例重启、更新记录并写审计
func (r *runner) restartInstance(
	ctx context.Context,
	restarter *devmode.Restarter,
	token, appID, workspaceID string,
	inst Instance,
) devmode.RestartInstanceResult {
	newFailedResult := func(err error) devmode.RestartInstanceResult {
		return devmode.RestartInstanceResult{
			EnvName: inst.EnvName, Instance: inst.Instance,
			Status: devmode.RecordStatusFailed, Message: err.Error(),
		}
	}

	recordID, err := bson.ObjectIDFromHex(inst.RecordID)
	if err != nil {
		return newFailedResult(errors.Wrapf(err, "invalid record id %s", inst.RecordID))
	}

	// 重投递时跳过已完成的记录，避免重复重启
	record, err := r.records.GetByID(ctx, recordID)
	if err != nil {
		restarter.MarkFailed(ctx, recordID, errors.Wrapf(err, "get restart record").Error())
		return newFailedResult(errors.Wrapf(err, "get restart record %s", inst.RecordID))
	}
	if record.Status == devmode.RecordStatusSuccess || record.Status == devmode.RecordStatusFailed {
		return devmode.RestartInstanceResult{EnvName: inst.EnvName, Instance: inst.Instance, Status: record.Status}
	}

	executor := k8sclient.NewPodClient(&cluster.Config{
		Rest: &rest.Config{
			BearerToken:     token,
			Host:            inst.Address,
			TLSClientConfig: rest.TLSClientConfig{Insecure: true},
		},
		ClusterID: inst.ClusterID,
	})
	result := restarter.Execute(ctx, appID, recordID, &devmode.RestartTarget{
		EnvName:   inst.EnvName,
		ClusterID: inst.ClusterID,
		Address:   inst.Address,
		Namespace: inst.Namespace,
		Container: inst.Container,
		MountPath: inst.MountPath,
	}, inst.Instance, executor)

	audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeRestart,
		audit.ResourceTypeInstance,
		result.Instance,
		audit.WithResult(resultStatus(result)),
		audit.WithDataAfter(map[string]any{
			"oldPID":  result.OldPID,
			"newPID":  result.NewPID,
			"message": result.Message,
		}),
		audit.WithWorkspaceID(workspaceID),
		audit.WithAppID(appID),
		audit.WithEnvName(inst.EnvName),
	)
	return result
}

// markFailed 将无法执行（如获取 Token 失败）的记录标记为失败
func (r *runner) markFailed(
	ctx context.Context, restarter *devmode.Restarter, recordID, message string,
) {
	id, err := bson.ObjectIDFromHex(recordID)
	if err != nil {
		return
	}
	restarter.MarkFailed(ctx, id, message)
}

// resultStatus 将重启结果映射为审计结果
func resultStatus(result devmode.RestartInstanceResult) audit.Result {
	if result.Status == devmode.RecordStatusSuccess {
		return audit.ResultSuccess
	}
	return audit.ResultFailed
}
