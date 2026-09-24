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

package publish

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	devmode "github.com/RimuruChan/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
	"github.com/RimuruChan/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
	watchplugin "github.com/RimuruChan/blueking-service-governance/bkms-server/pkg/workload/instance/watch/plugin"
)

// publishRecord 构造一条发布记录；只填展示需要的最小字段
func publishRecord(instance, status string) *devmode.PublishRecord {
	return &devmode.PublishRecord{
		Instance:   instance,
		BinaryName: "server",
		MD5:        "abc123",
		Status:     devmode.PublishStatus(status),
		Operator:   "test-user",
		UpdatedAt:  time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC),
	}
}

func snapshotOf(ids ...string) []watchplugin.InstanceSnapshot {
	snapshot := make([]watchplugin.InstanceSnapshot, 0, len(ids))
	for _, id := range ids {
		snapshot = append(snapshot, watchplugin.InstanceSnapshot{ID: id})
	}
	return snapshot
}

func publishStatus(payload any) *serializer.PublishStatusOutputObj {
	status, ok := payload.(*serializer.PublishStatusOutputObj)
	Expect(ok).To(BeTrue())
	return status
}

var _ = Describe("Plugin", func() {
	ctx := context.Background()

	It("returns the latest publish status of instances that have records", func() {
		p := New(func(_ context.Context, instanceIDs []string) (map[string]*devmode.PublishRecord, error) {
			Expect(instanceIDs).To(Equal([]string{"pod-1", "pod-2"}))

			return map[string]*devmode.PublishRecord{
				"pod-1": publishRecord("pod-1", string(devmode.PublishStatusSuccess)),
				"pod-2": publishRecord("pod-2", string(devmode.PublishStatusFailed)),
			}, nil
		})

		payloads, err := p.Fetch(ctx, snapshotOf("pod-1", "pod-2"))

		Expect(err).NotTo(HaveOccurred())
		Expect(p.Name()).To(Equal("devmodePublish"))

		Expect(payloads).To(HaveKey("pod-1"))
		Expect(publishStatus(payloads["pod-1"]).Status).To(Equal("success"))
		Expect(publishStatus(payloads["pod-2"]).Status).To(Equal("failed"))
		Expect(publishStatus(payloads["pod-2"]).BinaryName).To(Equal("server"))
		Expect(publishStatus(payloads["pod-2"]).Operator).To(Equal("test-user"))
	})

	// 无发布记录的实例直接缺 key：Runner 不推事件，前端保持「无发布状态」
	It("omits instances without publish records", func() {
		p := New(func(_ context.Context, _ []string) (map[string]*devmode.PublishRecord, error) {
			return map[string]*devmode.PublishRecord{}, nil
		})

		payloads, err := p.Fetch(ctx, snapshotOf("pod-1", "pod-2"))

		Expect(err).NotTo(HaveOccurred())
		Expect(payloads).To(BeEmpty())
	})

	// 拉取失败直接抛给 Runner：由它跳过本轮，页面保留上次已知状态
	It("propagates the fetch error", func() {
		p := New(func(_ context.Context, _ []string) (map[string]*devmode.PublishRecord, error) {
			return nil, errors.New("db down")
		})

		_, err := p.Fetch(ctx, snapshotOf("pod-1"))

		Expect(err).To(MatchError(ContainSubstring("db down")))
	})
})
