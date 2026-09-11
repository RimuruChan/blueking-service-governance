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

package client

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("flattenDashboardDirectoryTree", func() {
	It("should flatten the directory tree into a dashboard list", func() {
		nodes := []dashboardDirectoryNode{
			{
				Title: "General",
				Dashboards: []dashboardDirectoryRawItem{
					{UID: "uid-1", Title: "dashboard-1", URL: "/grafana/d/uid-1"},
					{UID: "uid-2", Title: "dashboard-2", URL: "/grafana/d/uid-2"},
				},
			},
			{
				Title: "Folder A",
				Dashboards: []dashboardDirectoryRawItem{
					{UID: "uid-3", Title: "dashboard-3", URL: "/grafana/d/uid-3"},
				},
			},
		}

		items := flattenDashboardDirectoryTree(nodes)

		Expect(items).To(HaveLen(3))
		Expect(items[0]).To(Equal(DashboardDirectoryItem{
			Folder: "General", UID: "uid-1", Title: "dashboard-1", URL: "/grafana/d/uid-1",
		}))
		Expect(items[1].Folder).To(Equal("General"))
		Expect(items[2]).To(Equal(DashboardDirectoryItem{
			Folder: "Folder A", UID: "uid-3", Title: "dashboard-3", URL: "/grafana/d/uid-3",
		}))
	})

	It("should skip empty directories", func() {
		nodes := []dashboardDirectoryNode{
			{Title: "General", Dashboards: []dashboardDirectoryRawItem{
				{UID: "uid-1", Title: "dashboard-1", URL: "/grafana/d/uid-1"},
			}},
			{Title: "Empty Folder", Dashboards: []dashboardDirectoryRawItem{}},
		}

		items := flattenDashboardDirectoryTree(nodes)

		Expect(items).To(HaveLen(1))
		Expect(items[0].UID).To(Equal("uid-1"))
	})

	It("should return an empty list for an empty tree", func() {
		items := flattenDashboardDirectoryTree(nil)
		Expect(items).To(BeEmpty())
	})
})
