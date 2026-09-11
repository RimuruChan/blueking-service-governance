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

package serializer_test

import (
	"encoding/json"

	"github.com/go-playground/validator/v10"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/dashboard/serializer"
)

var _ = Describe("AppDashboard Serializer", func() {
	var validate *validator.Validate

	BeforeEach(func() {
		validate = validator.New()
		validate.SetTagName("binding")
	})

	Describe("AppDashboardCreateInput", func() {
		It("should require uid", func() {
			Expect(validate.Struct(serializer.AppDashboardCreateInput{})).To(HaveOccurred())
			Expect(validate.Struct(serializer.AppDashboardCreateInput{UID: "u"})).
				NotTo(HaveOccurred())
		})
	})

	Describe("AppDashboardUpdateInput", func() {
		It("should allow empty update (uid optional)", func() {
			Expect(validate.Struct(serializer.AppDashboardUpdateInput{})).NotTo(HaveOccurred())
		})

		It("should accept uid", func() {
			uid := "new-uid"
			Expect(validate.Struct(serializer.AppDashboardUpdateInput{UID: &uid})).NotTo(HaveOccurred())
		})
	})

	Describe("NewDashboardOutputs", func() {
		It("should convert dashboard items into minimal output", func() {
			items := []*bkmapi.DashboardItem{
				{UID: "test-uid-1", Title: "Test Dashboard", URL: "/grafana/d/test-uid-1/test"},
			}

			outputs := serializer.NewDashboardOutputs(items)

			Expect(outputs).To(HaveLen(1))
			Expect(outputs[0].UID).To(Equal("test-uid-1"))
			Expect(outputs[0].Title).To(Equal("Test Dashboard"))
		})
	})

	Describe("ListDashboardsResp", func() {
		It("should marshal to the flat dashboard list shape", func() {
			resp := serializer.ListDashboardsResp{
				Data: []*serializer.DashboardOutput{
					{
						UID:   "test-uid-1",
						Title: "Test Dashboard",
						URL:   "/grafana/d/test-uid-1/test",
					},
				},
			}

			data, err := json.Marshal(resp)
			Expect(err).NotTo(HaveOccurred())

			var decoded serializer.ListDashboardsResp
			Expect(json.Unmarshal(data, &decoded)).NotTo(HaveOccurred())
			Expect(decoded.Data).To(HaveLen(1))
			Expect(decoded.Data[0].UID).To(Equal("test-uid-1"))
		})
	})
})
