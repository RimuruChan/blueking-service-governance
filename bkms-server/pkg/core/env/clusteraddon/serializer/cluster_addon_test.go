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
	"github.com/gin-gonic/gin/binding"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon/serializer"
)

var _ = Describe("Cluster addon namespace validation", func() {
	DescribeTable(
		"accepts an omitted namespace or bcs-system",
		func(input any, valid bool) {
			err := binding.Validator.ValidateStruct(input)
			if valid {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(MatchError(ContainSubstring("Namespace")))
				Expect(err).To(MatchError(ContainSubstring("oneof")))
			}
		},
		Entry("list with omitted namespace", serializer.ListClusterAddonsQueryInput{}, true),
		Entry("list with bcs-system", serializer.ListClusterAddonsQueryInput{Namespace: "bcs-system"}, true),
		Entry("list with custom namespace", serializer.ListClusterAddonsQueryInput{Namespace: "operators"}, false),
		Entry("upsert with omitted namespace", serializer.UpsertClusterAddonInput{ChartVersion: "1.0.0"}, true),
		Entry(
			"upsert with bcs-system",
			serializer.UpsertClusterAddonInput{Namespace: "bcs-system", ChartVersion: "1.0.0"},
			true,
		),
		Entry(
			"upsert with custom namespace",
			serializer.UpsertClusterAddonInput{Namespace: "operators", ChartVersion: "1.0.0"},
			false,
		),
		Entry("delete with omitted namespace", serializer.DeleteClusterAddonQueryInput{}, true),
		Entry("delete with bcs-system", serializer.DeleteClusterAddonQueryInput{Namespace: "bcs-system"}, true),
		Entry("delete with custom namespace", serializer.DeleteClusterAddonQueryInput{Namespace: "operators"}, false),
	)
})
