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
