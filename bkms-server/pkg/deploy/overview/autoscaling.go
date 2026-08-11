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
	"log/slog"
	"sync"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/gpa"
)

// autoscalingStatusByEnv envName -> GPA 运行状态；缺 key 表示未启用 GPA 或回查失败。
type autoscalingStatusByEnv map[string]*AutoscalingStatus

// listAutoscalingConfigsByEnv 返回各环境 GPA 配置摘要；无配置的环境不出现在 map 中。
func (s *Service) listAutoscalingConfigsByEnv(
	ctx context.Context,
	appID string,
) (map[string]*AutoscalingInfo, error) {
	out := map[string]*AutoscalingInfo{}
	if s.gpaConfigStore == nil {
		return out, nil
	}
	configs, err := s.gpaConfigStore.ListByApp(ctx, appID)
	if err != nil {
		return nil, errors.Wrap(err, "list gpa configs")
	}
	for _, cfg := range configs {
		if cfg == nil {
			continue
		}
		metrics := make([]AutoscalingMetric, 0, len(cfg.Metrics))
		for _, m := range cfg.Metrics {
			metrics = append(metrics, AutoscalingMetric{
				Resource:           string(m.Resource),
				AverageUtilization: m.AverageUtilization,
			})
		}
		out[cfg.EnvName] = &AutoscalingInfo{
			Enabled:         cfg.Enabled,
			CRName:          cfg.Name,
			MinReplicas:     cfg.MinReplicas,
			MaxReplicas:     cfg.MaxReplicas,
			Metrics:         metrics,
			ComputeByLimits: cfg.ComputeByLimits,
		}
	}
	return out, nil
}

// queryAutoscalingStatuses 为已启用 GPA 的环境并发回查集群 CR 状态。
//
// rows 只用于读取待查环境与 CR 名，不写回；回查失败的环境不出现在结果中，
// 不使整次总览失败（与 instances 降级策略一致）。
//
// Returns:
//   - envName -> GPA 运行状态
func (s *Service) queryAutoscalingStatuses(
	ctx context.Context,
	trackedEnvs []envmodel.Environment,
	rows []EnvRow,
) autoscalingStatusByEnv {
	out := make(autoscalingStatusByEnv, len(rows))
	if s.gpaService == nil {
		return out
	}
	envByName := lo.KeyBy(trackedEnvs, func(env envmodel.Environment) string { return env.Name })

	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrentK8sRequests)
	for i := range rows {
		info := rows[i].Autoscaling
		if info == nil || !info.Enabled || info.CRName == "" {
			continue
		}
		env, ok := envByName[rows[i].EnvName]
		if !ok || env.Cluster.ClusterID == "" {
			continue
		}
		envName := rows[i].EnvName
		g.Go(func() error {
			status := s.queryAutoscalingStatusForEnv(gctx, &env, info.CRName)
			if status == nil {
				return nil
			}
			mu.Lock()
			out[envName] = status
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()
	return out
}

// queryAutoscalingStatusForEnv 回查单个环境的 GPA CR 状态。
//
// CR 不存在、集群不可达时返回 nil，
// 由调用方降级为「状态不可用」。
func (s *Service) queryAutoscalingStatusForEnv(
	ctx context.Context,
	env *envmodel.Environment,
	crName string,
) *AutoscalingStatus {
	status, err := s.gpaService.Get(ctx, env, crName)
	if err != nil {
		if !errors.Is(err, gpa.ErrCRNotFound) {
			log.ErrorAttrs(ctx, "query deploy overview gpa status failed",
				slog.String("env_name", env.Name),
				slog.String("cluster_id", env.Cluster.ClusterID),
				slog.String("gpa_name", crName),
				slog.Any("error", err),
			)
		}
		return nil
	}
	return &AutoscalingStatus{
		CurrentReplicas: status.CurrentReplicas,
		DesiredReplicas: status.DesiredReplicas,
		LastScaleTime:   status.LastScaleTime,
		Phase:           status.Phase,
		StatusMessage:   status.StatusMessage,
	}
}
