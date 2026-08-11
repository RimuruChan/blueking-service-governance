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

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/gpa"
)

var _ = Describe("autoscaling helpers", func() {
	var (
		ctx context.Context
		svc *Service
		env *envmodel.Environment
	)

	BeforeEach(func() {
		ctx = context.Background()
		svc = &Service{gpaService: gpa.NewGPAService(nil)}
		env = &envmodel.Environment{
			Name:    "prod",
			Cluster: envmodel.BizCluster{ClusterID: "cls-1", Namespace: "ns-1"},
		}
	})

	Describe("queryAutoscalingStatusForEnv", func() {
		It("maps CR status fields on success", func() {
			mockey.PatchConvey("gpa get succeeds", GinkgoT(), func() {
				mockey.Mock((*gpa.GPAService).Get).Return(&gpa.GPAStatus{
					CurrentReplicas: 3,
					DesiredReplicas: 5,
					LastScaleTime:   "2026-08-11T10:00:00Z",
					Phase:           "Active",
					StatusMessage:   "scaling in progress",
				}, nil).Build()

				status := svc.queryAutoscalingStatusForEnv(ctx, env, "gpa-app")
				Expect(status).NotTo(BeNil())
				Expect(status.CurrentReplicas).To(Equal(int32(3)))
				Expect(status.DesiredReplicas).To(Equal(int32(5)))
				Expect(status.LastScaleTime).To(Equal("2026-08-11T10:00:00Z"))
				Expect(status.Phase).To(Equal("Active"))
				Expect(status.StatusMessage).To(Equal("scaling in progress"))
			})
		})

		It("returns nil when the CR does not exist in cluster", func() {
			mockey.PatchConvey("gpa CR not found", GinkgoT(), func() {
				mockey.Mock((*gpa.GPAService).Get).Return(nil, gpa.ErrCRNotFound).Build()

				Expect(svc.queryAutoscalingStatusForEnv(ctx, env, "gpa-app")).To(BeNil())
			})
		})

		It("returns nil when the cluster query fails", func() {
			mockey.PatchConvey("gpa get fails", GinkgoT(), func() {
				mockey.Mock((*gpa.GPAService).Get).Return(nil, errors.New("cluster unreachable")).Build()

				Expect(svc.queryAutoscalingStatusForEnv(ctx, env, "gpa-app")).To(BeNil())
			})
		})

		It("returns nil when the cluster query panics", func() {
			mockey.PatchConvey("gpa get panics", GinkgoT(), func() {
				mockey.Mock((*gpa.GPAService).Get).To(func(
					_ *gpa.GPAService, _ context.Context, _ *envmodel.Environment, _ string,
				) (*gpa.GPAStatus, error) {
					panic("failed to build config from local kubeconfig")
				}).Build()

				Expect(svc.queryAutoscalingStatusForEnv(ctx, env, "gpa-app")).To(BeNil())
			})
		})
	})
})
