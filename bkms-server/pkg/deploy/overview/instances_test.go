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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

var _ = Describe("instance helpers", func() {
	Describe("extractGameDeployReplicas", func() {
		It("returns replicas from spec", func() {
			replicas, ok := extractGameDeployReplicas(map[string]any{
				"spec": map[string]any{"replicas": int64(5)},
			})
			Expect(ok).To(BeTrue())
			Expect(replicas).To(Equal(int32(5)))
		})

		It("returns not found when replicas missing", func() {
			_, ok := extractGameDeployReplicas(map[string]any{"spec": map[string]any{}})
			Expect(ok).To(BeFalse())
		})
	})

	Describe("countPodStates", func() {
		newPod := func(phase, readyStatus string) unstructured.Unstructured {
			return unstructured.Unstructured{Object: map[string]any{
				"status": map[string]any{
					"phase":      phase,
					"conditions": []any{map[string]any{"type": "Ready", "status": readyStatus}},
				},
			}}
		}

		It("counts running and ready pods as running, the others as abnormal", func() {
			running, abnormal := countPodStates([]unstructured.Unstructured{
				newPod("Running", "True"), newPod("Running", "False"), newPod("Running", "True"),
			})
			Expect(running).To(Equal(int32(2)))
			Expect(abnormal).To(Equal(int32(1)))
		})

		It("counts terminated pods as abnormal even when reported as completed", func() {
			running, abnormal := countPodStates([]unstructured.Unstructured{
				{Object: map[string]any{"status": map[string]any{
					"phase": "Failed",
					"conditions": []any{
						map[string]any{"type": "Ready", "status": "False", "reason": "PodCompleted"},
					},
				}}},
			})
			Expect(running).To(BeZero())
			Expect(abnormal).To(Equal(int32(1)))
		})

		It("counts pods without a Ready condition as abnormal", func() {
			running, abnormal := countPodStates([]unstructured.Unstructured{
				{Object: map[string]any{"status": map[string]any{"phase": "Running"}}},
			})
			Expect(running).To(BeZero())
			Expect(abnormal).To(Equal(int32(1)))
		})

		It("returns zeros for an empty pod list", func() {
			running, abnormal := countPodStates(nil)
			Expect(running).To(BeZero())
			Expect(abnormal).To(BeZero())
		})
	})

	Describe("extractGameDeployName", func() {
		It("picks the GameDeployment out of the recorded resources", func() {
			name := extractGameDeployName(&appmodel.Record{ResourceKeys: appmodel.ResourceKeys{
				{Kind: k8skind.SVC, Name: "svc-app"},
				{Kind: k8skind.GameDeploy, Name: "app"},
			}})
			Expect(name).To(Equal("app"))
		})

		It("returns an empty name when no GameDeployment was recorded", func() {
			Expect(extractGameDeployName(&appmodel.Record{})).To(BeEmpty())
		})
	})

	Describe("groupDeployRecordsByCluster", func() {
		It("groups records of the same cluster together", func() {
			byCluster := groupDeployRecordsByCluster([]deployRecordForEnv{
				{EnvName: "dev", Record: &appmodel.Record{ClusterID: "cls-1"}},
				{EnvName: "test", Record: &appmodel.Record{ClusterID: "cls-1"}},
				{EnvName: "prod", Record: &appmodel.Record{ClusterID: "cls-2"}},
			})
			Expect(byCluster).To(HaveLen(2))
			Expect(byCluster["cls-1"]).To(HaveLen(2))
			Expect(byCluster["cls-2"]).To(HaveLen(1))
		})

		It("drops records that cannot locate a cluster", func() {
			Expect(groupDeployRecordsByCluster([]deployRecordForEnv{
				{EnvName: "no-record"},
				{EnvName: "no-cluster", Record: &appmodel.Record{}},
			})).To(BeEmpty())
		})
	})

	// 以下两个降级分支都在发起集群调用之前返回，因此 querier 无需持有真实客户端。
	Describe("queryInstanceCountsForEnv", func() {
		newItem := func(record *appmodel.Record) deployRecordForEnv {
			return deployRecordForEnv{EnvName: "prod", Record: record}
		}

		It("reports unavailable when the deploy record has no label selector", func() {
			counts, err := queryInstanceCountsForEnv(context.Background(), &clusterQuerier{}, newItem(
				&appmodel.Record{
					Namespace:    "ns-1",
					ResourceKeys: appmodel.ResourceKeys{{Kind: k8skind.GameDeploy, Name: "app"}},
				},
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(counts).To(BeNil())
		})

		It("reports unavailable when the deploy record has no GameDeployment", func() {
			counts, err := queryInstanceCountsForEnv(context.Background(), &clusterQuerier{}, newItem(
				&appmodel.Record{
					Namespace:     "ns-1",
					LabelSelector: map[string]string{"app.kubernetes.io/name": "app"},
					ResourceKeys:  appmodel.ResourceKeys{{Kind: k8skind.SVC, Name: "app"}},
				},
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(counts).To(BeNil())
		})
	})
})
