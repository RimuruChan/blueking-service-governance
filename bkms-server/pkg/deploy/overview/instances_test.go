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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("instance helpers", func() {
	Describe("isPodReady", func() {
		It("returns true when Ready condition is True", func() {
			Expect(isPodReady(map[string]any{
				"status": map[string]any{
					"conditions": []any{
						map[string]any{"type": "Ready", "status": "True"},
					},
				},
			})).To(BeTrue())
		})

		It("returns true for PodCompleted reason", func() {
			Expect(isPodReady(map[string]any{
				"status": map[string]any{
					"conditions": []any{
						map[string]any{
							"type": "Ready", "status": "False", "reason": "PodCompleted",
						},
					},
				},
			})).To(BeTrue())
		})

		It("returns false when Ready is absent or false", func() {
			Expect(isPodReady(map[string]any{})).To(BeFalse())
			Expect(isPodReady(map[string]any{
				"status": map[string]any{
					"conditions": []any{
						map[string]any{"type": "Ready", "status": "False"},
					},
				},
			})).To(BeFalse())
		})
	})

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
})
