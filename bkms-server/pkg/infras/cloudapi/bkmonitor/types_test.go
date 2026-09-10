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

package bkmonitor

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewUpdateApmServiceConfigReq", func() {
	It("negates a positive bk_biz_id into a container project id", func() {
		req := NewUpdateApmServiceConfigReq(123, "app", "svc", nil, nil)
		Expect(req.BkBizID).To(Equal(int64(-123)))
		Expect(req.AppName).To(Equal("app"))
		Expect(req.ServiceName).To(Equal("svc"))
	})

	It("keeps a negative bk_biz_id unchanged", func() {
		req := NewUpdateApmServiceConfigReq(-456, "app", "svc", nil, nil)
		Expect(req.BkBizID).To(Equal(int64(-456)))
	})
})

var _ = Describe("UpdateApmServiceConfigReq validation", func() {
	It("accepts a valid request", func() {
		Expect(Validate(&UpdateApmServiceConfigReq{
			BkBizID:     -2,
			AppName:     "checkout",
			ServiceName: "checkout-api",
			Owners:      []string{"alice"},
			IncrementalK8sRelations: []ApmServiceK8sRelation{{
				BcsClusterID: "BCS-K8S-00000",
				Namespace:    "prod",
				Kind:         "Deployment",
				Name:         "checkout-api",
			}},
		})).To(Succeed())
	})

	It("rejects a request with positive bk_biz_id", func() {
		Expect(Validate(&UpdateApmServiceConfigReq{
			BkBizID:     2,
			AppName:     "checkout",
			ServiceName: "checkout-api",
		})).To(HaveOccurred())
	})

	It("rejects a request without app_name or service_name", func() {
		Expect(Validate(&UpdateApmServiceConfigReq{
			BkBizID: -2,
		})).To(HaveOccurred())
	})
})
