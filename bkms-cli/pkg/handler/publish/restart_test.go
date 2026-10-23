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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client/mocks"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/clierr"
)

var _ = Describe("BuildRestartOptions", func() {
	It("builds envs target with deduplicated env names", func() {
		opts, err := BuildRestartOptions("stage, test,stage", "", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(opts).To(Equal(client.DevModeRestartOptions{
			Envs: &client.DevModeRestartEnvsTarget{Names: []string{"stage", "test"}},
		}))
	})

	It("builds envTypes target", func() {
		opts, err := BuildRestartOptions("", "development,test", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(opts).To(Equal(client.DevModeRestartOptions{
			EnvTypes: &client.DevModeRestartEnvTypesTarget{Types: []string{"development", "test"}},
		}))
	})

	It("builds instances target with a single env", func() {
		opts, err := BuildRestartOptions("stage", "", "pod-1,pod-2")
		Expect(err).NotTo(HaveOccurred())
		Expect(opts).To(Equal(client.DevModeRestartOptions{
			Instances: &client.DevModeRestartInstancesTarget{
				EnvName:     "stage",
				InstanceIDs: []string{"pod-1", "pod-2"},
			},
		}))
	})

	DescribeTable("rejects invalid flag combinations",
		func(envs, envTypes, instances, msg string) {
			_, err := BuildRestartOptions(envs, envTypes, instances)
			Expect(err).To(HaveOccurred())
			var usageErr *clierr.UsageError
			Expect(errors.As(err, &usageErr)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring(msg))
		},
		Entry("both env and env-type", "stage", "test", "", "cannot be used together"),
		Entry("neither env nor env-type", "", "", "", "one of --env or --env-type is required"),
		Entry("blank env", " , ", "", "", "one of --env or --env-type is required"),
		Entry("instances with multiple envs", "stage,test", "", "pod-1", "single --env"),
		Entry("instances with env-type", "", "test", "pod-1", "single --env"),
	)
})

var _ = Describe("Restart", func() {
	const appID = "demo-app"

	var (
		ctx  context.Context
		cli  *mocks.MockClient
		opts client.DevModeRestartOptions
	)

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
		opts = client.DevModeRestartOptions{Envs: &client.DevModeRestartEnvsTarget{Names: []string{"stage"}}}
	})

	recordIDs := func(count int) []string {
		ids := make([]string, 0, count)
		for i := 0; i < count; i++ {
			ids = append(ids, fmt.Sprintf("%024d", i))
		}
		return ids
	}
	accepted := func(count int) *client.DevModeRestartData {
		return &client.DevModeRestartData{InstanceCount: count, RecordIDs: recordIDs(count)}
	}
	newRecord := func(instance, status string) client.DevModePublishRecord {
		return client.DevModePublishRecord{
			Action: "restart", EnvName: "stage", Instance: instance, Status: status,
			Message: "message of " + instance,
		}
	}
	expectProgress := func(count int) *mocks.MockClient_ListDevModePublishRecords_Call {
		return cli.EXPECT().ListDevModePublishRecords(
			mock.Anything, appID, client.DevModePublishRecordsQuery{
				RecordIDs: recordIDs(count), Statuses: []string{"pending", "running"}, Page: 1, PageSize: 1,
			},
		)
	}
	expectFinal := func(count int) *mocks.MockClient_ListDevModePublishRecords_Call {
		return cli.EXPECT().ListDevModePublishRecords(
			mock.Anything, appID, client.DevModePublishRecordsQuery{
				RecordIDs: recordIDs(count), Page: 1, PageSize: 100,
			},
		)
	}

	It("submits the request and waits until all instances are terminal", func() {
		cli.EXPECT().RestartDevModeProcesses(mock.Anything, appID, opts).Return(accepted(2), nil)
		expectProgress(2).Return([]client.DevModePublishRecord{}, int64(2), nil).Once()
		expectProgress(2).Return([]client.DevModePublishRecord{}, int64(0), nil).Once()
		expectFinal(2).Return(
			[]client.DevModePublishRecord{newRecord("pod-1", "success"), newRecord("pod-2", "success")}, int64(2), nil,
		)

		Expect(restartAndWait(ctx, cli, appID, opts, 10*time.Millisecond, time.Second)).To(Succeed())
	})

	It("returns an error when any instance failed", func() {
		cli.EXPECT().RestartDevModeProcesses(mock.Anything, appID, opts).Return(accepted(2), nil)
		expectProgress(2).Return([]client.DevModePublishRecord{}, int64(0), nil)
		expectFinal(2).Return([]client.DevModePublishRecord{
			newRecord("pod-1", "success"), newRecord("pod-2", "failed"),
		}, int64(2), nil)

		err := restartAndWait(ctx, cli, appID, opts, 10*time.Millisecond, time.Second)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("restart failed on 1 instance(s)"))
	})

	It("times out when records never reach a terminal state", func() {
		cli.EXPECT().RestartDevModeProcesses(mock.Anything, appID, opts).Return(accepted(1), nil)
		expectProgress(1).Return([]client.DevModePublishRecord{newRecord("pod-1", "pending")}, int64(1), nil)

		err := restartAndWait(ctx, cli, appID, opts, 10*time.Millisecond, 50*time.Millisecond)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("timed out"))
	})

	It("returns an error when the server returns no records", func() {
		cli.EXPECT().RestartDevModeProcesses(mock.Anything, appID, opts).Return(accepted(0), nil)

		err := restartAndWait(ctx, cli, appID, opts, 10*time.Millisecond, time.Second)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("server returned no restart records"))
	})

	It("wraps the client error", func() {
		cli.EXPECT().RestartDevModeProcesses(mock.Anything, appID, opts).Return(nil, errors.New("server down"))

		err := restartAndWait(ctx, cli, appID, opts, 10*time.Millisecond, time.Second)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("server down"))
	})
})
