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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/RimuruChan/blueking-service-governance/bkms-server/pkg/common/testutil"
)

var _ = Describe("AppDashboardStoreMongo", func() {
	var (
		ctx   context.Context
		diApp *fxtest.App
		store AppDashboardStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		err := testutil.CleanupCollection("bkmonitor_app_dashboard")
		Expect(err).NotTo(HaveOccurred())

		diApp = fxtest.New(
			GinkgoT(),
			FxModule,
			fx.Populate(&store),
		)
		diApp.RequireStart()
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	Describe("Create", func() {
		It("should create a binding successfully", func() {
			d := &AppDashboard{AppID: "app-a", UID: "uid-1", Title: "My Dashboard", Creator: "user1"}
			id, err := store.Create(ctx, d)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).NotTo(Equal(bson.NilObjectID))
			Expect(d.CreatedAt).NotTo(BeZero())
		})

		It("should reject duplicate (appID, uid)", func() {
			_, err := store.Create(ctx, &AppDashboard{AppID: "app-a", UID: "uid-1", Title: "t1"})
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, &AppDashboard{AppID: "app-a", UID: "uid-1", Title: "t2"})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrDuplicate)).To(BeTrue())
		})

		It("should allow same uid across different apps", func() {
			_, err := store.Create(ctx, &AppDashboard{AppID: "app-a", UID: "uid-1", Title: "t1"})
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, &AppDashboard{AppID: "app-b", UID: "uid-1", Title: "t2"})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ListByApp", func() {
		BeforeEach(func() {
			_, _ = store.Create(ctx, &AppDashboard{AppID: "app-a", UID: "uid-1", Title: "t1"})
			_, _ = store.Create(ctx, &AppDashboard{AppID: "app-a", UID: "uid-2", Title: "t2"})
			_, _ = store.Create(ctx, &AppDashboard{AppID: "app-b", UID: "uid-3", Title: "t3"})
		})

		It("should list bindings by app ID", func() {
			list, err := store.ListByApp(ctx, "app-a")
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(HaveLen(2))

			list, err = store.ListByApp(ctx, "app-b")
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].UID).To(Equal("uid-3"))
		})

		It("should return empty list when no bindings", func() {
			list, err := store.ListByApp(ctx, "app-empty")
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(BeEmpty())
		})
	})

	Describe("Update", func() {
		BeforeEach(func() {
			_, _ = store.Create(ctx, &AppDashboard{AppID: "app-a", UID: "uid-1", Title: "t1"})
		})

		It("should update title", func() {
			newTitle := "New Title"
			err := store.Update(ctx, "app-a", "uid-1", "tester", &AppDashboardUpdateData{Title: &newTitle})
			Expect(err).NotTo(HaveOccurred())

			list, err := store.ListByApp(ctx, "app-a")
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Title).To(Equal("New Title"))
			Expect(list[0].Updater).To(Equal("tester"))
		})

		It("should update uid", func() {
			newUID := "uid-new"
			err := store.Update(ctx, "app-a", "uid-1", "tester", &AppDashboardUpdateData{UID: &newUID})
			Expect(err).NotTo(HaveOccurred())

			list, err := store.ListByApp(ctx, "app-a")
			Expect(err).NotTo(HaveOccurred())
			Expect(list[0].UID).To(Equal("uid-new"))
		})

		It("should reject duplicate uid on update", func() {
			_, _ = store.Create(ctx, &AppDashboard{AppID: "app-a", UID: "uid-2", Title: "t2"})

			newUID := "uid-2"
			err := store.Update(ctx, "app-a", "uid-1", "tester", &AppDashboardUpdateData{UID: &newUID})
			Expect(errors.Is(err, ErrDuplicate)).To(BeTrue())
		})

		It("should return not found when binding missing", func() {
			newTitle := "x"
			err := store.Update(ctx, "app-a", "uid-missing", "tester", &AppDashboardUpdateData{Title: &newTitle})
			Expect(errors.Is(err, ErrNotFound)).To(BeTrue())
		})
	})

	Describe("Delete", func() {
		BeforeEach(func() {
			_, _ = store.Create(ctx, &AppDashboard{AppID: "app-a", UID: "uid-1", Title: "t1"})
		})

		It("should delete binding", func() {
			err := store.Delete(ctx, "app-a", "uid-1")
			Expect(err).NotTo(HaveOccurred())

			list, err := store.ListByApp(ctx, "app-a")
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(BeEmpty())
		})

		It("should return not found when binding missing", func() {
			err := store.Delete(ctx, "app-a", "uid-missing")
			Expect(errors.Is(err, ErrNotFound)).To(BeTrue())
		})
	})
})
