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
	"maps"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	podstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/workload/pod"
)

// deployRecordForEnv 携带查 K8s 实例所需的最新 AppModel 部署记录。
type deployRecordForEnv struct {
	EnvName string
	Record  *appmodel.Record
}

// instanceCountsByEnv envName -> 实例数。
// 缺 key 或值为 nil 均表示不可用（序列化为 JSON null）；与「0 运行 / 0 异常」不同。
type instanceCountsByEnv map[string]*InstanceCounts

// clusterDeployBatch 同一集群上需要一并查询的环境部署记录。
// 约定：同一集群内各环境的 (clusterID, namespace) 唯一。
type clusterDeployBatch struct {
	clusterID string
	items     []deployRecordForEnv
}

// clusterQuerier 单集群内查询实例数所需的客户端与共享并发闸门。
// 客户端按集群创建一次，供该集群下各环境复用。
type clusterQuerier struct {
	pods *k8sclient.PodClient
	gd   *k8sclient.Client
	sem  *semaphore.Weighted
}

// newClusterQuerier 创建单集群查询器；集群配置只解析一次，Pod 与 GameDeployment 客户端共用。
func newClusterQuerier(clusterID string, sem *semaphore.Weighted) *clusterQuerier {
	clusterCfg := cluster.NewConfig(clusterID)
	return &clusterQuerier{
		pods: k8sclient.NewPodClient(clusterCfg),
		gd:   k8sclient.NewWithGVR(clusterCfg, gvr.GameDeploy),
		sem:  sem,
	}
}

// queryInstanceCounts 按集群并发查询各环境实例数。
//
// 集群之间并发；集群内各环境并发；单环境内 Pod List 与 GameDeployment Get 并发。
// 三层扇出均不设上限，真正的在途请求数由 sem 统一约束。
// 单环境失败只影响该环境（保持 nil），不中断其它环境/集群，也不使整次总览失败。
//
// Args:
//   - sem 本次请求内所有集群回查共享的在途请求闸门
//   - records 已过滤到表格行内、且含 AppModel 部署记录的环境
//
// Returns:
//   - envName -> 实例数；失败或无法定位 workload 的环境不出现或为 nil
func queryInstanceCounts(
	ctx context.Context,
	sem *semaphore.Weighted,
	records []deployRecordForEnv,
) instanceCountsByEnv {
	out := make(instanceCountsByEnv, len(records))
	if len(records) == 0 {
		return out
	}

	byCluster := groupDeployRecordsByCluster(records)
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	for _, batch := range byCluster {
		g.Go(func() error {
			counts := queryInstanceCountsForCluster(gctx, sem, batch.clusterID, batch.items)
			mu.Lock()
			defer mu.Unlock()
			maps.Copy(out, counts)
			return nil
		})
	}
	_ = g.Wait()
	return out
}

// groupDeployRecordsByCluster 按 ClusterID 分组；无 ClusterID 的记录无法查 K8s，直接丢弃。
func groupDeployRecordsByCluster(records []deployRecordForEnv) map[string]*clusterDeployBatch {
	byCluster := map[string]*clusterDeployBatch{}
	for _, item := range records {
		if item.Record == nil || item.Record.ClusterID == "" {
			continue
		}
		b, ok := byCluster[item.Record.ClusterID]
		if !ok {
			b = &clusterDeployBatch{clusterID: item.Record.ClusterID}
			byCluster[item.Record.ClusterID] = b
		}
		b.items = append(b.items, item)
	}
	return byCluster
}

// queryInstanceCountsForCluster 并发查询单集群上各环境的实例数。
//
// 每个环境独立发起：
//   - Pod：命名空间内按 LabelSelector List（避免 AllNamespaces 宽拉）
//   - GameDeployment：按 ns/name Get（避免全量 List）
//
// Pod 与 GD 在同一环境内并发；环境之间也并发。
// 任一环境的查询失败只跳过该环境（instances 保持 nil），不影响同集群其它环境。
//
// Args:
//   - sem 本次请求内所有集群回查共享的在途请求闸门
//   - clusterID BCS / 本地集群 ID
//   - items 同属于该集群的环境部署记录（约定 namespace 互不重复）
//
// Returns:
//   - envName -> 实例数；失败环境不写入
func queryInstanceCountsForCluster(
	ctx context.Context,
	sem *semaphore.Weighted,
	clusterID string,
	items []deployRecordForEnv,
) instanceCountsByEnv {
	out := make(instanceCountsByEnv, len(items))
	if len(items) == 0 {
		return out
	}

	querier := newClusterQuerier(clusterID, sem)

	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	for _, item := range items {
		g.Go(func() error {
			counts, err := queryInstanceCountsForEnv(gctx, querier, item)
			if err != nil {
				log.ErrorAttrs(gctx, "query deploy overview instances failed",
					slog.String("cluster_id", clusterID),
					slog.String("env_name", item.EnvName),
					slog.String("namespace", item.Record.Namespace),
					slog.Any("error", err),
				)
				return nil
			}
			if counts == nil {
				// 缺 GD / replicas 等不可用场景，按设计降级为 null，不记错误日志
				return nil
			}
			mu.Lock()
			out[item.EnvName] = counts
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()
	return out
}

// queryInstanceCountsForEnv 查询单个环境的运行/期望/异常实例数。
// Pod List 与 GameDeployment Get 并发；任一失败或缺少 GD 时返回 (nil, err/nil) 由调用方降级。
func queryInstanceCountsForEnv(
	ctx context.Context,
	querier *clusterQuerier,
	item deployRecordForEnv,
) (*InstanceCounts, error) {
	gdName := extractGameDeployName(item.Record)
	if gdName == "" {
		return nil, nil
	}

	var (
		pods     []unstructured.Unstructured
		expected int32
		podsErr  error
		gdErr    error
		gdOK     bool
	)

	g, gctx := errgroup.WithContext(ctx)
	// 错误经 podsErr / gdErr 带回，两个查询都返回 nil，避免其中一个失败就取消另一个
	g.Go(func() error {
		pods, podsErr = listPods(gctx, querier, item)
		return nil
	})
	g.Go(func() error {
		expected, gdOK, gdErr = getGameDeployReplicas(gctx, querier, item, gdName)
		return nil
	})
	_ = g.Wait()

	if podsErr != nil {
		return nil, podsErr
	}
	if gdErr != nil {
		return nil, gdErr
	}
	if !gdOK {
		return nil, nil
	}

	running, abnormal := countPodStates(pods)
	return &InstanceCounts{
		Running:  running,
		Expected: expected,
		Abnormal: abnormal,
	}, nil
}

// countPodStates 统计 Ready 与非 Ready 的 Pod 数。
func countPodStates(pods []unstructured.Unstructured) (running, abnormal int32) {
	for _, pod := range pods {
		if podstatus.IsReady(pod.Object) {
			running++
		} else {
			abnormal++
		}
	}
	return running, abnormal
}

// extractGameDeployName 从部署记录的 ResourceKeys 中取 GameDeployment 名称；没有则返回空串。
func extractGameDeployName(rec *appmodel.Record) string {
	for _, key := range rec.ResourceKeys {
		if key.Kind == k8skind.GameDeploy {
			return key.Name
		}
	}
	return ""
}

// listPods 在环境命名空间内按 LabelSelector List Pod。
//
// 指定 ResourceVersion="0"，让 apiserver 直接从 watch cache 返回，不必走 etcd 的 quorum read：
// 这是本接口里最大的一笔查询（整个 Pod 列表），而总览只用于展示运行/异常实例数，
// 可以接受 cache 落后主库几百毫秒；需要强一致读的场景（如部署前置校验）不应复用本函数。
func listPods(
	ctx context.Context,
	querier *clusterQuerier,
	item deployRecordForEnv,
) ([]unstructured.Unstructured, error) {
	if err := querier.sem.Acquire(ctx, 1); err != nil {
		return nil, errors.Wrap(err, "acquire k8s request slot")
	}
	defer querier.sem.Release(1)

	ns := item.Record.Namespace
	sel := labels.SelectorFromSet(item.Record.LabelSelector).String()
	list, err := querier.pods.List(ctx, ns, metav1.ListOptions{LabelSelector: sel, ResourceVersion: "0"})
	if err != nil {
		return nil, errors.Wrapf(err, "list pods in namespace %s", ns)
	}
	return list.Items, nil
}

// getGameDeployReplicas 按 ns/name Get GameDeployment 并读取 spec.replicas。
// 找不到或 replicas 缺失时 ok=false（调用方视为该环境实例不可用）。
func getGameDeployReplicas(
	ctx context.Context,
	querier *clusterQuerier,
	item deployRecordForEnv,
	gdName string,
) (replicas int32, ok bool, err error) {
	if err = querier.sem.Acquire(ctx, 1); err != nil {
		return 0, false, errors.Wrap(err, "acquire k8s request slot")
	}
	defer querier.sem.Release(1)

	res, err := querier.gd.Get(ctx, item.Record.Namespace, gdName, metav1.GetOptions{})
	if err != nil {
		return 0, false, errors.Wrapf(
			err, "get game deployment %s/%s", item.Record.Namespace, gdName,
		)
	}
	replicas, ok = extractGameDeployReplicas(res.Object)
	return replicas, ok, nil
}

// extractGameDeployReplicas 读取 GameDeployment.spec.replicas。
// 字段缺失时返回 ok=false：K8s 缺省虽为 1，总览更稳妥地视为「期望数不可用」。
func extractGameDeployReplicas(manifest map[string]any) (int32, bool) {
	if manifest == nil {
		return 0, false
	}
	replicas, found, _ := unstructured.NestedInt64(manifest, "spec", "replicas")
	if !found {
		return 0, false
	}
	if replicas < 0 || replicas > int64(^uint32(0)>>1) {
		return 0, false
	}
	return int32(replicas), true //nolint:gosec // bounded by check above
}
