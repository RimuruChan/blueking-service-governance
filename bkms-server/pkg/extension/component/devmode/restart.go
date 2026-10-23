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

package devmode

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

const (
	// restartExecTimeout 单次读取 pid.conf 的 exec 超时时间
	restartExecTimeout = 10 * time.Second
	// restartStopTimeout 执行 stop.sh 并清除停止标志的超时时间（taf 优雅退出最长约 15s）
	restartStopTimeout = 60 * time.Second
	// defaultRestartWaitStartTimeout 等待 monitor 拉起进程（pid.conf 重新生成）的默认超时时间
	defaultRestartWaitStartTimeout = 30 * time.Second
	// defaultRestartPollInterval 轮询 pid.conf 的默认间隔
	defaultRestartPollInterval = time.Second
)

// 脚本通过 `bash -c <script> bash <mountPath>` 执行，$1 为脚本挂载路径；
// source utils.sh 时屏蔽输出，避免污染 stdout。
const (
	// readPIDScript 读取 pid.conf，文件不存在时以非 0 退出码结束
	readPIDScript = `source "$1/utils.sh" >/dev/null 2>&1 && cat "${PID_CONF}" 2>/dev/null`

	// stopProcessScript 停止业务进程并清除停止标志，交由 monitor 重新拉起。
	// source 失败则不 stop；stop 失败也删除 pid.conf 并清 flag，退出码透传 stop.sh。
	// 删除 pid.conf 是因为只有 start.sh 在进程启动后才会重新写入，以此判断已拉起。
	stopProcessScript = `source "$1/utils.sh" >/dev/null 2>&1 || { echo "failed to source $1/utils.sh" >&2; exit 1; }
bash "$1/stop.sh"
rc=$?
rm -f "${PID_CONF}"
clear_stop_flag
exit ${rc}`
)

// PodExecutor 在 pod 内执行命令，*k8sclient.PodClient 实现了该接口
type PodExecutor interface {
	Exec(ctx context.Context, namespace, podName, container string, cmd []string) (*k8sclient.ExecResult, error)
}

// RestartTarget 单个环境下的进程重启目标
type RestartTarget struct {
	// EnvName 环境名称
	EnvName string
	// ClusterID 实例所在集群 ID
	ClusterID string
	// Address 集群 API 完整地址
	Address string
	// Namespace 实例所在命名空间
	Namespace string
	// Container 业务容器名称
	Container string
	// MountPath devmode 脚本挂载路径
	MountPath string
	// InstanceIDs 目标实例（pod）名称列表
	InstanceIDs []string
}

// RestartInstanceResult 单个实例的重启结果
type RestartInstanceResult struct {
	// EnvName 环境名称
	EnvName string
	// Instance 实例（pod）名称
	Instance string
	// Status 重启结果：success / failed
	Status RecordStatus
	// Message 结果说明
	Message string
	// OldPID 重启前的业务进程 PID（pid.conf 不存在时为空）
	OldPID string
	// NewPID 重启后的业务进程 PID
	NewPID string
}

// Restarter 开发模式业务进程重启器：stop 业务进程后由 monitor 重新拉起，pod 不重建
type Restarter struct {
	records          *PublishRecordStore
	pollInterval     time.Duration
	waitStartTimeout time.Duration
}

// RestarterOption Restarter 可选配置
type RestarterOption func(*Restarter)

// WithRestartPollInterval 设置轮询 pid.conf 的间隔
func WithRestartPollInterval(d time.Duration) RestarterOption {
	return func(r *Restarter) { r.pollInterval = d }
}

// WithRestartWaitStartTimeout 设置等待进程被拉起的超时时间
func WithRestartWaitStartTimeout(d time.Duration) RestarterOption {
	return func(r *Restarter) { r.waitStartTimeout = d }
}

// NewRestarter 创建 Restarter，records 用于写入 restart 类型的发布记录
func NewRestarter(records *PublishRecordStore, opts ...RestarterOption) *Restarter {
	r := &Restarter{
		records:          records,
		pollInterval:     defaultRestartPollInterval,
		waitStartTimeout: defaultRestartWaitStartTimeout,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// errWaitStartTimeout 等待 monitor 拉起进程超时
var errWaitStartTimeout = errors.New("timed out waiting for monitor to start the process")

// PreparedInstance 已写入 pending 记录、等待异步执行的实例
type PreparedInstance struct {
	// RecordID 发布记录 ID
	RecordID bson.ObjectID
	// Target 实例所属环境的重启目标
	Target *RestartTarget
	// Instance 实例（pod）名称
	Instance string
}

// Prepare 为所有目标实例写入 pending 记录，返回待执行实例列表。
func (r *Restarter) Prepare(
	ctx context.Context, appID, operator string, targets []RestartTarget,
) ([]PreparedInstance, error) {
	prepared := lo.FlatMap(targets, func(target RestartTarget, i int) []PreparedInstance {
		return lo.Map(targets[i].InstanceIDs, func(instance string, _ int) PreparedInstance {
			return PreparedInstance{Target: &targets[i], Instance: instance}
		})
	})
	if len(prepared) == 0 {
		return nil, nil
	}

	records := lo.Map(prepared, func(_ PreparedInstance, i int) *PublishRecord {
		return &PublishRecord{
			AppID:    appID,
			EnvName:  prepared[i].Target.EnvName,
			Action:   RecordActionRestart,
			Instance: prepared[i].Instance,
			Status:   RecordStatusPending,
			Operator: operator,
		}
	})
	// CreateMany 会将生成的 ID 回填到各 record 中
	if err := r.records.CreateMany(ctx, records); err != nil {
		return nil, errors.Wrapf(err, "create restart records for app %s", appID)
	}
	for i := range prepared {
		prepared[i].RecordID = records[i].ID
	}
	return prepared, nil
}

// Execute 执行单个实例的重启并更新记录状态，executor 由调用方按集群构建。
func (r *Restarter) Execute(
	ctx context.Context,
	appID string,
	recordID bson.ObjectID,
	target *RestartTarget,
	instance string,
	executor PodExecutor,
) RestartInstanceResult {
	// 记录更新不随请求取消，避免记录停留在 running
	recordCtx := context.WithoutCancel(ctx)
	r.updateRecord(recordCtx, recordID, RecordStatusRunning, "")

	startedAt := time.Now()
	oldPID, newPID, err := r.restartInstance(ctx, target, instance, executor)
	result := newRestartInstanceResult(target.EnvName, instance, oldPID, newPID, err)
	r.observe(ctx, appID, result, err, startedAt)
	r.updateRecord(recordCtx, recordID, result.Status, result.Message)
	return result
}

// MarkFailed 将 pending 状态的记录直接标记为失败（如任务投递失败）
func (r *Restarter) MarkFailed(ctx context.Context, recordID bson.ObjectID, message string) {
	r.updateRecord(context.WithoutCancel(ctx), recordID, RecordStatusFailed, message)
}

// updateRecord 更新重启记录状态（best-effort，失败仅记录日志）
func (r *Restarter) updateRecord(ctx context.Context, id bson.ObjectID, status RecordStatus, message string) {
	update := map[string]any{"status": status, "message": message}
	if err := r.records.Update(ctx, update, store.ByID(id)); err != nil {
		log.WarnAttrs(ctx, "update devmode restart record failed",
			slog.String("record_id", id.Hex()),
			slog.String("status", string(status)),
			slog.String("err", err.Error()),
		)
	}
}

// observe 上报单实例重启耗时指标并记录日志
func (r *Restarter) observe(
	ctx context.Context, appID string, result RestartInstanceResult, err error, startedAt time.Time,
) {
	status := metrics.StatusOK
	if err != nil {
		status = lo.Ternary(errors.Is(err, errWaitStartTimeout), metrics.StatusTimeout, metrics.StatusErr)
	}
	metrics.DevModeRestartFinished(status, startedAt)
	log.InfoAttrs(ctx, "devmode restart instance finished",
		slog.String("app_id", appID),
		slog.String("env_name", result.EnvName),
		slog.String("instance", result.Instance),
		slog.String("status", string(result.Status)),
		slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		slog.String("old_pid", result.OldPID),
		slog.String("new_pid", result.NewPID),
		slog.String("message", result.Message),
	)
}

// newRestartInstanceResult 将单实例的执行结果转换为 RestartInstanceResult
func newRestartInstanceResult(envName, instance, oldPID, newPID string, err error) RestartInstanceResult {
	result := RestartInstanceResult{EnvName: envName, Instance: instance, OldPID: oldPID}
	if err != nil {
		result.Status = RecordStatusFailed
		result.Message = err.Error()
		return result
	}
	result.NewPID = newPID
	result.Status = RecordStatusSuccess
	result.Message = fmt.Sprintf("process restarted, pid: %s -> %s", lo.CoalesceOrEmpty(oldPID, "none"), newPID)
	return result
}

// restartInstance 读取旧 PID → stop → 等待 monitor 拉起；读旧 PID 失败时不执行 stop。
func (r *Restarter) restartInstance(
	ctx context.Context, target *RestartTarget, instance string, executor PodExecutor,
) (oldPID, newPID string, err error) {
	oldPID, _, err = r.readPID(ctx, target, instance, executor)
	if err != nil {
		return "", "", errors.Wrap(err, "read pid.conf")
	}

	// 一旦开始 stop，后续步骤（stop 与等待拉起）不随请求取消，保证进程状态与记录一致
	opCtx := context.WithoutCancel(ctx)
	stopCtx, cancel := context.WithTimeout(opCtx, restartStopTimeout)
	defer cancel()
	stopRet, err := r.exec(stopCtx, target, instance, stopProcessScript, executor)
	if err != nil {
		return oldPID, "", errors.Wrap(err, "stop process")
	}
	if stopRet.ExitCode != 0 {
		if stderr := strings.TrimSpace(stopRet.Stderr); stderr != "" {
			return oldPID, "", errors.Errorf("stop process: stop.sh exited with code %d: %s", stopRet.ExitCode, stderr)
		}
		return oldPID, "", errors.Errorf("stop process: stop.sh exited with code %d", stopRet.ExitCode)
	}

	newPID, err = r.waitProcessStarted(opCtx, target, instance, executor)
	if err != nil {
		return oldPID, "", errors.Wrap(err, "process stopped but not started")
	}
	return oldPID, newPID, nil
}

// waitProcessStarted 轮询 pid.conf 直到文件出现或超时，返回新 PID
func (r *Restarter) waitProcessStarted(
	ctx context.Context, target *RestartTarget, instance string, executor PodExecutor,
) (string, error) {
	waitCtx, cancel := context.WithTimeout(ctx, r.waitStartTimeout)
	defer cancel()

	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// exec 异常（如网络抖动）不中断轮询，直到超时
			pid, exists, err := r.readPID(waitCtx, target, instance, executor)
			if err == nil && exists {
				return pid, nil
			}
		case <-waitCtx.Done():
			return "", errors.Wrapf(errWaitStartTimeout, "pid.conf not created within %s", r.waitStartTimeout)
		}
	}
}

// readPID 读取 pid.conf，返回 PID 及文件是否存在
func (r *Restarter) readPID(
	ctx context.Context, target *RestartTarget, instance string, executor PodExecutor,
) (string, bool, error) {
	execCtx, cancel := context.WithTimeout(ctx, restartExecTimeout)
	defer cancel()

	ret, err := r.exec(execCtx, target, instance, readPIDScript, executor)
	if err != nil {
		return "", false, err
	}
	if ret.ExitCode != 0 {
		return "", false, nil
	}
	return strings.TrimSpace(ret.Stdout), true, nil
}

// exec 在目标实例内执行脚本，$1 为脚本挂载路径
func (r *Restarter) exec(
	ctx context.Context, target *RestartTarget, instance, script string, executor PodExecutor,
) (*k8sclient.ExecResult, error) {
	cmd := []string{"bash", "-c", script, "bash", target.MountPath}
	ret, err := executor.Exec(ctx, target.Namespace, instance, target.Container, cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "exec in instance %s of env %s", instance, target.EnvName)
	}
	return ret, nil
}
