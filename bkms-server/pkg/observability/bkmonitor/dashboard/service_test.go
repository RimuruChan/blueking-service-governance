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

package dashboard

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	bkmapi "github.com/RimuruChan/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
)

var _ = Describe("dashboardInfoMap", func() {
	It("should flatten the directory tree into a uid-indexed map", func() {
		tree := []*bkmapi.DashboardDirectoryNode{
			{
				ID:    0,
				UID:   "",
				Title: "General",
				Dashboards: []bkmapi.DashboardItem{
					{
						ID:    1,
						UID:   "test-uid-1",
						Title: "Dashboard One",
						URI:   "db/one",
						URL:   "/grafana/d/test-uid-1/one",
					},
				},
			},
			{
				ID:    100,
				UID:   "folder-uid-1",
				Title: "Folder One",
				Dashboards: []bkmapi.DashboardItem{
					{
						ID:    2,
						UID:   "test-uid-2",
						Title: "Dashboard Two",
						URI:   "db/two",
						URL:   "/grafana/d/test-uid-2/two",
					},
				},
			},
		}

		infoMap := dashboardInfoMap(tree)

		Expect(infoMap).To(HaveLen(2))
		Expect(infoMap["test-uid-1"].ID).To(Equal(int64(1)))
		Expect(infoMap["test-uid-2"].URL).To(Equal("/grafana/d/test-uid-2/two"))
	})
})

var _ = Describe("assembleDashboards", func() {
	It("should assemble bound records with bkmonitor info", func() {
		records := []AppDashboard{
			{AppID: "app-a", UID: "test-uid-1", Title: "Custom Title"},
			{AppID: "app-a", UID: "test-uid-2", Title: "Another Title"},
		}
		infoMap := map[string]bkmapi.DashboardItem{
			"test-uid-1": {UID: "test-uid-1", URL: "/grafana/d/test-uid-1/one"},
		}

		items := assembleDashboards(records, infoMap)

		Expect(items).To(HaveLen(2))

		// title comes from the binding record, url comes from bkmonitor
		Expect(items[0].Title).To(Equal("Custom Title"))
		Expect(items[0].URL).To(Equal("/grafana/d/test-uid-1/one"))

		// records missing from bkmonitor are still returned with empty url
		Expect(items[1].UID).To(Equal("test-uid-2"))
		Expect(items[1].Title).To(Equal("Another Title"))
		Expect(items[1].URL).To(BeEmpty())
	})
})
