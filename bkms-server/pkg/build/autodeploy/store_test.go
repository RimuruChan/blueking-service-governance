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

package autodeploy_test

import (
	"context"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
)

var _ = Describe("RecordStoreMongo", func() {
	var (
		store           autodeploy.RecordStore
		diApp           *fxtest.App
		ctx             context.Context
		appID           string
		trafficLaneName string
	)

	// newRecord 构造同一应用下指定环境的记录，buildID 保证 appID + buildID 唯一索引不冲突。
	newRecord := func(envName, imageTag, laneName string) *autodeploy.Record {
		return &autodeploy.Record{
			WorkspaceID:     "test-workspace-" + stringx.Random(6),
			AppID:           appID,
			AppType:         "trpc",
			EnvName:         envName,
			TrafficLaneName: laneName,
			BuildID:         "build-" + stringx.Random(8),
			ImageTag:        imageTag,
			Stage:           autodeploy.StageBuild,
			Status:          "pending",
			Operator:        "admin",
			StartedAt:       time.Now(),
		}
	}

	BeforeEach(func() {
		diApp = fxtest.New(
			GinkgoT(),
			autodeploy.FxModule,
			fx.Populate(&store),
		)
		diApp.RequireStart()

		ctx = context.Background()
		appID = "test-app-" + stringx.Random(6)
		trafficLaneName = ""
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	Describe("Create", func() {
		It("should create a record and set timestamps", func() {
			record := newRecord("stag", "v1", trafficLaneName)

			Expect(store.Create(ctx, record)).To(Succeed())

			got, err := store.GetByBuildID(ctx, appID, record.BuildID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ID).NotTo(Equal(bson.NilObjectID))
			Expect(got.AppID).To(Equal(appID))
			Expect(got.EnvName).To(Equal("stag"))
			Expect(got.ImageTag).To(Equal("v1"))
			Expect(got.Stage).To(Equal(autodeploy.StageBuild))
			Expect(got.Status).To(Equal("pending"))
			Expect(got.CreatedAt).NotTo(BeZero())
			Expect(got.UpdatedAt).NotTo(BeZero())
			Expect(got.UpdatedAt).To(Equal(got.CreatedAt))
		})
	})

	Describe("Update", func() {
		It("should update mutable fields and refresh updatedAt", func() {
			record := newRecord("stag", "v1", trafficLaneName)
			Expect(store.Create(ctx, record)).To(Succeed())

			created, err := store.GetByBuildID(ctx, appID, record.BuildID)
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(5 * time.Millisecond)

			endedAt := time.Now()
			created.DeployID = "deploy-" + stringx.Random(8)
			created.Stage = autodeploy.StageDeploy
			created.Status = "deployed"
			created.Message = "ok"
			created.EndedAt = endedAt

			Expect(store.Update(ctx, created)).To(Succeed())

			got, err := store.GetByBuildID(ctx, appID, record.BuildID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.DeployID).To(Equal(created.DeployID))
			Expect(got.Stage).To(Equal(autodeploy.StageDeploy))
			Expect(got.Status).To(Equal("deployed"))
			Expect(got.Message).To(Equal("ok"))
			Expect(got.EndedAt.Unix()).To(Equal(endedAt.Unix()))
			Expect(got.UpdatedAt.UnixMilli()).To(BeNumerically(">", created.CreatedAt.UnixMilli()))
			// Update 只更新指定字段，ImageTag 等保持不变
			Expect(got.ImageTag).To(Equal("v1"))
		})

		It("should return ErrRecordNotFound when updating a missing record", func() {
			record := newRecord("stag", "v1", trafficLaneName)
			record.ID = bson.NewObjectID()

			err := store.Update(ctx, record)
			Expect(err).To(MatchError(autodeploy.ErrRecordNotFound))
		})
	})

	Describe("GetLatest", func() {
		It("should return the newest record for the env and traffic lane", func() {
			older := newRecord("stag", "stag-v1", trafficLaneName)
			Expect(store.Create(ctx, older)).To(Succeed())
			time.Sleep(5 * time.Millisecond)

			newer := newRecord("stag", "stag-v2", trafficLaneName)
			Expect(store.Create(ctx, newer)).To(Succeed())
			Expect(store.Create(ctx, newRecord("prod", "prod-v1", trafficLaneName))).To(Succeed())
			Expect(store.Create(ctx, newRecord("stag", "lane-v1", "lane-a"))).To(Succeed())

			got, err := store.GetLatest(ctx, appID, "stag", trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ImageTag).To(Equal("stag-v2"))
			Expect(got.BuildID).To(Equal(newer.BuildID))
		})

		It("should return ErrRecordNotFound when no record exists", func() {
			_, err := store.GetLatest(ctx, appID, "stag", trafficLaneName)
			Expect(err).To(MatchError(autodeploy.ErrRecordNotFound))
		})
	})

	Describe("ListLatestByApp", func() {
		It("should return the newest record of each env", func() {
			Expect(store.Create(ctx, newRecord("stag", "stag-v1", trafficLaneName))).To(Succeed())
			time.Sleep(5 * time.Millisecond)
			Expect(store.Create(ctx, newRecord("stag", "stag-v2", trafficLaneName))).To(Succeed())
			Expect(store.Create(ctx, newRecord("prod", "prod-v1", trafficLaneName))).To(Succeed())

			latestByEnv, err := store.ListLatestByApp(ctx, appID, trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latestByEnv).To(HaveLen(2))
			Expect(latestByEnv["stag"].ImageTag).To(Equal("stag-v2"))
			Expect(latestByEnv["prod"].ImageTag).To(Equal("prod-v1"))
		})

		It("should only return records of the given traffic lane", func() {
			Expect(store.Create(ctx, newRecord("stag", "base-v1", trafficLaneName))).To(Succeed())
			Expect(store.Create(ctx, newRecord("stag", "lane-v1", "lane-a"))).To(Succeed())

			latestByEnv, err := store.ListLatestByApp(ctx, appID, trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latestByEnv).To(HaveLen(1))
			Expect(latestByEnv["stag"].ImageTag).To(Equal("base-v1"))

			laneByEnv, err := store.ListLatestByApp(ctx, appID, "lane-a")
			Expect(err).NotTo(HaveOccurred())
			Expect(laneByEnv).To(HaveLen(1))
			Expect(laneByEnv["stag"].ImageTag).To(Equal("lane-v1"))
		})

		It("should not return records of other apps", func() {
			Expect(store.Create(ctx, newRecord("stag", "mine-v1", trafficLaneName))).To(Succeed())

			other := newRecord("stag", "other-v1", trafficLaneName)
			other.AppID = "test-app-" + stringx.Random(6)
			Expect(store.Create(ctx, other)).To(Succeed())

			latestByEnv, err := store.ListLatestByApp(ctx, appID, trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latestByEnv).To(HaveLen(1))
			Expect(latestByEnv["stag"].ImageTag).To(Equal("mine-v1"))
		})

		It("should return an empty map when the app has no record", func() {
			latestByEnv, err := store.ListLatestByApp(ctx, "non-existent-app", trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latestByEnv).To(BeEmpty())
		})
	})

	Describe("GetByBuildID", func() {
		It("should return the record matching appID and buildID", func() {
			record := newRecord("stag", "v1", trafficLaneName)
			Expect(store.Create(ctx, record)).To(Succeed())

			got, err := store.GetByBuildID(ctx, appID, record.BuildID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.BuildID).To(Equal(record.BuildID))
			Expect(got.ImageTag).To(Equal("v1"))
		})

		It("should return ErrRecordNotFound for unknown buildID", func() {
			_, err := store.GetByBuildID(ctx, appID, "missing-build")
			Expect(err).To(MatchError(autodeploy.ErrRecordNotFound))
		})

		It("should not return another app's record with the same buildID", func() {
			record := newRecord("stag", "v1", trafficLaneName)
			Expect(store.Create(ctx, record)).To(Succeed())

			_, err := store.GetByBuildID(ctx, "other-app", record.BuildID)
			Expect(err).To(MatchError(autodeploy.ErrRecordNotFound))
		})
	})

	Describe("GetByDeployID", func() {
		It("should return the record matching appID and deployID", func() {
			record := newRecord("stag", "v1", trafficLaneName)
			Expect(store.Create(ctx, record)).To(Succeed())

			created, err := store.GetByBuildID(ctx, appID, record.BuildID)
			Expect(err).NotTo(HaveOccurred())

			deployID := "deploy-" + stringx.Random(8)
			created.DeployID = deployID
			created.Stage = autodeploy.StageDeploy
			created.Status = "deploying"
			Expect(store.Update(ctx, created)).To(Succeed())

			got, err := store.GetByDeployID(ctx, appID, deployID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.BuildID).To(Equal(record.BuildID))
			Expect(got.DeployID).To(Equal(deployID))
			Expect(got.Stage).To(Equal(autodeploy.StageDeploy))
		})

		It("should return ErrRecordNotFound for unknown deployID", func() {
			_, err := store.GetByDeployID(ctx, appID, "missing-deploy")
			Expect(err).To(MatchError(autodeploy.ErrRecordNotFound))
		})

		It("should not return another app's record with the same deployID", func() {
			record := newRecord("stag", "v1", trafficLaneName)
			Expect(store.Create(ctx, record)).To(Succeed())

			created, err := store.GetByBuildID(ctx, appID, record.BuildID)
			Expect(err).NotTo(HaveOccurred())

			deployID := "deploy-" + stringx.Random(8)
			created.DeployID = deployID
			Expect(store.Update(ctx, created)).To(Succeed())

			_, err = store.GetByDeployID(ctx, "other-app", deployID)
			Expect(err).To(MatchError(autodeploy.ErrRecordNotFound))
		})

		It("should not match empty deployID", func() {
			// Create 后 DeployID 为空；GetByDeployID("", "") 不应误命中
			Expect(store.Create(ctx, newRecord("stag", "v1", trafficLaneName))).To(Succeed())

			_, err := store.GetByDeployID(ctx, appID, "")
			Expect(err).To(MatchError(autodeploy.ErrRecordNotFound))
		})
	})

	// 保留一个组合场景，覆盖 Create → Update → GetLatest 主路径
	Describe("end-to-end flow", func() {
		It("should reflect deploy progress through store APIs", func() {
			record := newRecord("stag", "v1", trafficLaneName)
			Expect(store.Create(ctx, record)).To(Succeed())

			created, err := store.GetByBuildID(ctx, appID, record.BuildID)
			Expect(err).NotTo(HaveOccurred())

			deployID := "deploy-" + stringx.Random(8)
			created.DeployID = deployID
			created.Stage = autodeploy.StageDeploy
			created.Status = "deployed"
			created.Message = "done"
			created.EndedAt = time.Now()
			Expect(store.Update(ctx, created)).To(Succeed())

			latest, err := store.GetLatest(ctx, appID, "stag", trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest.DeployID).To(Equal(deployID))
			Expect(latest.Status).To(Equal("deployed"))

			byDeploy, err := store.GetByDeployID(ctx, appID, deployID)
			Expect(err).NotTo(HaveOccurred())
			Expect(byDeploy.BuildID).To(Equal(record.BuildID))
		})
	})
})
