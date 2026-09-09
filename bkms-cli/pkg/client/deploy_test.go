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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeployPrecheckResult", func() {
	Describe("Normalize", func() {
		It("passes when both lists are empty", func() {
			result := &DeployPrecheckResult{}
			result.Normalize()
			Expect(result.Passed).To(BeTrue())
			Expect(result.UndefinedVars).NotTo(BeNil())
			Expect(result.MissingRequiredClusterAddons).NotTo(BeNil())
		})

		It("fails when only env vars are missing", func() {
			result := &DeployPrecheckResult{
				UndefinedVars: []UndefinedEnvVar{{Key: "DB_HOST"}},
			}
			result.Normalize()
			Expect(result.Passed).To(BeFalse())
		})

		It("fails when only required cluster addons are missing", func() {
			result := &DeployPrecheckResult{
				MissingRequiredClusterAddons: []ClusterAddonReference{{Name: "game", DisplayName: "Gamedeploy"}},
			}
			result.Normalize()
			Expect(result.Passed).To(BeFalse())
		})

		It("fails when both lists have findings", func() {
			result := &DeployPrecheckResult{
				UndefinedVars:                []UndefinedEnvVar{{Key: "DB_HOST"}},
				MissingRequiredClusterAddons: []ClusterAddonReference{{Name: "hook", DisplayName: "Hook-operator"}},
			}
			result.Normalize()
			Expect(result.Passed).To(BeFalse())
		})
	})

	It("omits passed from JSON and keeps named finding lists", func() {
		result := &DeployPrecheckResult{
			UndefinedVars:                []UndefinedEnvVar{},
			MissingRequiredClusterAddons: []ClusterAddonReference{{Name: "game", DisplayName: "Gamedeploy"}},
		}
		result.Normalize()
		payload, err := json.Marshal(result)
		Expect(err).NotTo(HaveOccurred())
		Expect(payload).To(MatchJSON(`{
			"undefinedVars": [],
			"missingRequiredClusterAddons": [{"name":"game","displayName":"Gamedeploy"}]
		}`))
	})
})
