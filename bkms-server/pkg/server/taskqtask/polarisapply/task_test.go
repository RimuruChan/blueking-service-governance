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

package polarisapply

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"testing"
	"time"

	"github.com/bytedance/mockey"
	"github.com/hibiken/asynq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	depsvcmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

var _ = Describe("Polaris dynamic apply task", func() {
	var (
		ctx               context.Context
		diApp             *fxtest.App
		appStore          bkmsapp.ApplicationStore
		envStore          bkmsenv.EnvironmentStore
		envService        *env.EnvService
		store             polaris.PolarisConfigStore
		appModelStore     appmodel.AppModelStore
		scopedEnvVarStore envvars.ScopedEnvVarStore
		depSvcStore       depsvcmodel.ServiceStore
		depSvcInstStore   depsvcmodel.ServiceInstanceStore
		envStateManager   *polaris.PolarisEnvStateManager
		service           *polaris.PolarisConfigService
		app               *bkmsapp.Application
		environment       *bkmsenv.Environment
		otherEnvironment  *bkmsenv.Environment
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			env.FxModule,
			appmodel.FxModule,
			envvars.FxModule,
			depsvcmodel.FxModule,
			polaris.FxModule,
			fx.Populate(
				&appStore,
				&envStore,
				&envService,
				&store,
				&appModelStore,
				&scopedEnvVarStore,
				&depSvcStore,
				&depSvcInstStore,
				&envStateManager,
			),
		)
		diApp.RequireStart()

		Init(appStore, store, envStore, appModelStore, scopedEnvVarStore, depSvcInstStore)
		service = polaris.NewPolarisConfigService(
			store,
			polaris.NewPolarisPlatformManager(depSvcStore, depSvcInstStore, store),
			envStateManager,
			envStore,
			Enqueue,
		)
		app = dbfactory.Application(ctx, appStore)
		environment = dbfactory.Env(ctx, envService, app.WorkspaceID)
		otherEnvironment = dbfactory.Env(ctx, envService, app.WorkspaceID)
	})

	AfterEach(func() {
		_ = store.DeleteByApp(ctx, app.ID)
		diApp.RequireStop()
	})

	runHandler := func(args Args) error {
		payload, err := json.Marshal(args)
		Expect(err).NotTo(HaveOccurred())
		task := asynq.NewTask(DynamicApplyTask.Name(), payload)
		return DynamicApplyTask.Handler()(ctx, task)
	}

	createDeployedConfig := func(name string, envNames []string) *polaris.PolarisConfig {
		applied := redeployFields("k1", "t1", 8080)
		states := make(map[string]polaris.PolarisEnvState, len(envNames))
		for _, envName := range envNames {
			states[envName] = envState(applied)
		}
		config := newTestConfig(app.ID, name, envNames, states)
		Expect(store.Create(ctx, config)).To(Succeed())
		return config
	}

	Describe("Update enqueue", func() {
		It("should enqueue one asynq task per ready environment", func() {
			config := createDeployedConfig("cfg-enqueue", []string{environment.Name, otherEnvironment.Name})
			var envNames []string
			mockey.PatchConvey("enqueue succeeds", GinkgoT(), func() {
				mockey.Mock(taskq.Enqueue).To(func(
					_ context.Context, task *taskq.Task, _ ...asynq.Option,
				) error {
					Expect(task).NotTo(BeNil())
					envNames = append(envNames, "called")
					return nil
				}).Build()

				_, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
					Direct: lo.ToPtr(false),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(envNames).To(HaveLen(2))
			})
		})

		It("should record lastError when asynq enqueue fails without failing Update", func() {
			config := createDeployedConfig("cfg-enqueue-fail", []string{environment.Name})
			mockey.PatchConvey("enqueue fails", GinkgoT(), func() {
				mockey.Mock(taskq.Enqueue).Return(errors.New("asynq unavailable")).Build()

				_, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
					Direct: lo.ToPtr(false),
				})
				Expect(err).NotTo(HaveOccurred())

				stored, getErr := store.Get(ctx, app.ID, config.Name)
				Expect(getErr).NotTo(HaveOccurred())
				Expect(stored.GetEnvState(environment.Name).LastError).To(And(
					ContainSubstring("asynq unavailable"),
					ContainSubstring(environment.Name),
				))
			})
		})

		It("should keep enqueuing other envs and record lastError for failed ones", func() {
			config := createDeployedConfig(
				"cfg-enqueue-partial",
				[]string{environment.Name, otherEnvironment.Name},
			)
			mockey.PatchConvey("first env enqueue fails", GinkgoT(), func() {
				var calls int
				mockey.Mock(taskq.Enqueue).To(func(
					_ context.Context, _ *taskq.Task, _ ...asynq.Option,
				) error {
					calls++
					if calls == 1 {
						return errors.New("asynq unavailable")
					}
					return nil
				}).Build()

				_, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
					Direct: lo.ToPtr(false),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(calls).To(Equal(2))

				stored, getErr := store.Get(ctx, app.ID, config.Name)
				Expect(getErr).NotTo(HaveOccurred())
				Expect(stored.GetEnvState(environment.Name).LastError).To(ContainSubstring("asynq unavailable"))
				Expect(stored.GetEnvState(otherEnvironment.Name).LastError).To(BeEmpty())
			})
		})

		It("should not enqueue when no environment is ready for dynamic apply", func() {
			config := newTestConfig(app.ID, "cfg-no-apply", []string{environment.Name}, nil)
			Expect(store.Create(ctx, config)).To(Succeed())
			var enqueued bool
			mockey.PatchConvey("enqueue must not be called", GinkgoT(), func() {
				mockey.Mock(taskq.Enqueue).To(func(
					_ context.Context, _ *taskq.Task, _ ...asynq.Option,
				) error {
					enqueued = true
					return nil
				}).Build()

				_, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
					Direct: lo.ToPtr(false),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(enqueued).To(BeFalse())
			})
		})

		It("should enqueue repeated updates without a fixed task ID", func() {
			config := createDeployedConfig("cfg-repeated-enqueue", []string{environment.Name})
			var calls int
			mockey.PatchConvey("repeated updates enqueue independently", GinkgoT(), func() {
				mockey.Mock(taskq.Enqueue).To(func(
					_ context.Context, _ *taskq.Task, opts ...asynq.Option,
				) error {
					calls++
					for _, opt := range opts {
						Expect(opt.Type()).NotTo(Equal(asynq.TaskIDOpt))
					}
					return nil
				}).Build()

				updated, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{
					Direct: lo.ToPtr(true),
				})
				Expect(err).NotTo(HaveOccurred())
				_, err = service.Update(ctx, app, updated, &polaris.ConfigUpdateData{
					Direct: lo.ToPtr(false),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(calls).To(Equal(2))
			})
		})
	})

	Describe("handler", func() {
		It("should stop retry when the task is not initialized", func() {
			dynamicApplyService = nil
			err := runHandler(Args{
				AppID: app.ID, ConfigName: "missing", EnvName: environment.Name,
			})
			Expect(stderrors.Is(err, asynq.SkipRetry)).To(BeTrue())
		})

		It("should stop retry when the app no longer exists", func() {
			err := runHandler(Args{
				AppID: "missing-app", ConfigName: "cfg", EnvName: environment.Name,
			})
			Expect(stderrors.Is(err, asynq.SkipRetry)).To(BeTrue())
		})

		It("should stop retry when the config no longer exists", func() {
			err := runHandler(Args{
				AppID: app.ID, ConfigName: "missing-config", EnvName: environment.Name,
			})
			Expect(stderrors.Is(err, asynq.SkipRetry)).To(BeTrue())
		})

		It("should stop retry when the app model no longer exists", func() {
			config := createDeployedConfig("cfg-missing-model", []string{environment.Name})
			err := runHandler(Args{
				AppID: app.ID, ConfigName: config.Name, EnvName: environment.Name,
			})
			Expect(stderrors.Is(err, asynq.SkipRetry)).To(BeTrue())
		})

		It("should stop retry when the environment no longer exists", func() {
			const missingEnvName = "missing-env"
			config := newTestConfig(
				app.ID,
				"cfg-missing-env",
				[]string{missingEnvName},
				map[string]polaris.PolarisEnvState{missingEnvName: envState(redeployFields("k1", "t1", 8080))},
			)
			Expect(store.Create(ctx, config)).To(Succeed())
			Expect(appModelStore.CreateAppModel(ctx, &appmodel.AppModel{AppID: app.ID})).To(Succeed())
			DeferCleanup(func() { _ = appModelStore.DeleteAppModel(ctx, app.ID) })

			err := runHandler(Args{
				AppID: app.ID, ConfigName: config.Name, EnvName: missingEnvName,
			})
			Expect(stderrors.Is(err, asynq.SkipRetry)).To(BeTrue())
		})

		It("should record lastError with retry progress when apply fails", func() {
			config := createDeployedConfig("cfg-retry-progress", []string{environment.Name})
			Expect(appModelStore.CreateAppModel(ctx, &appmodel.AppModel{AppID: app.ID})).To(Succeed())
			DeferCleanup(func() { _ = appModelStore.DeleteAppModel(ctx, app.ID) })

			mockey.PatchConvey("cluster discovery fails with retry progress", GinkgoT(), func() {
				mockPolarisDiscoveryFailure()
				mockey.Mock(asynq.GetRetryCount).Return(2, true).Build()
				mockey.Mock(asynq.GetMaxRetry).Return(10, true).Build()

				err := runHandler(Args{
					AppID: app.ID, ConfigName: config.Name, EnvName: environment.Name,
				})
				Expect(err).To(HaveOccurred())
				Expect(stderrors.Is(err, asynq.SkipRetry)).To(BeFalse())

				stored, getErr := store.Get(ctx, app.ID, config.Name)
				Expect(getErr).NotTo(HaveOccurred())
				Expect(stored.GetEnvState(environment.Name).LastError).To(And(
					ContainSubstring("test discovery error"),
					ContainSubstring("(retry 3/11)"),
				))
			})
		})

		It("should clear lastError when a later retry succeeds", func() {
			config := createDeployedConfig("cfg-retry-success", []string{environment.Name})
			Expect(appModelStore.CreateAppModel(ctx, &appmodel.AppModel{AppID: app.ID})).To(Succeed())
			DeferCleanup(func() { _ = appModelStore.DeleteAppModel(ctx, app.ID) })
			Expect(envStateManager.RecordDynamicApplyResult(
				ctx, app.ID, config.Name, environment.Name, config.UpdatedAt, errors.New("previous apply error"),
			)).To(Succeed())

			mockey.PatchConvey("apply succeeds on retry", GinkgoT(), func() {
				mockey.Mock((*polaris.CRApplier).Apply).Return(nil).Build()

				err := runHandler(Args{
					AppID: app.ID, ConfigName: config.Name, EnvName: environment.Name,
				})
				Expect(err).NotTo(HaveOccurred())

				stored, getErr := store.Get(ctx, app.ID, config.Name)
				Expect(getErr).NotTo(HaveOccurred())
				Expect(stored.GetEnvState(environment.Name).LastError).To(BeEmpty())
			})
		})

		It("should retry without overwriting a newer result when config changes during apply", func() {
			config := createDeployedConfig("cfg-changed-during-apply", []string{environment.Name})
			Expect(appModelStore.CreateAppModel(ctx, &appmodel.AppModel{AppID: app.ID})).To(Succeed())
			DeferCleanup(func() { _ = appModelStore.DeleteAppModel(ctx, app.ID) })

			mockey.PatchConvey("config changes while applying", GinkgoT(), func() {
				mockey.Mock((*polaris.CRApplier).Apply).To(func(
					_ *polaris.CRApplier,
					_ context.Context,
					_ *bkmsapp.Application,
					_ *bkmsenv.Environment,
					_ *polaris.PolarisConfig,
					_ map[string]string,
				) error {
					time.Sleep(5 * time.Millisecond)
					Expect(store.Update(ctx, app.ID, config.Name, &polaris.ConfigUpdateData{
						Direct: lo.ToPtr(true),
					})).To(Succeed())
					latest, err := store.Get(ctx, app.ID, config.Name)
					Expect(err).NotTo(HaveOccurred())
					Expect(envStateManager.RecordDynamicApplyResult(
						ctx,
						app.ID,
						config.Name,
						environment.Name,
						latest.UpdatedAt,
						errors.New("newer task error"),
					)).To(Succeed())
					return nil
				}).Build()

				err := runHandler(Args{
					AppID: app.ID, ConfigName: config.Name, EnvName: environment.Name,
				})
				Expect(err).To(MatchError(ContainSubstring("changed during dynamic apply")))
				Expect(stderrors.Is(err, asynq.SkipRetry)).To(BeFalse())
				Expect(stderrors.Is(err, polaris.ErrDynamicApplyConfigChanged)).To(BeTrue())

				stored, getErr := store.Get(ctx, app.ID, config.Name)
				Expect(getErr).NotTo(HaveOccurred())
				Expect(stored.GetEnvState(environment.Name).LastError).To(Equal("newer task error"))
			})
		})

		It("should mark lastError as exhausted when retries are exhausted", func() {
			config := createDeployedConfig("cfg-retry-exhausted", []string{environment.Name})
			Expect(appModelStore.CreateAppModel(ctx, &appmodel.AppModel{AppID: app.ID})).To(Succeed())
			DeferCleanup(func() { _ = appModelStore.DeleteAppModel(ctx, app.ID) })

			mockey.PatchConvey("retries exhausted", GinkgoT(), func() {
				mockey.Mock((*polaris.CRApplier).Apply).Return(errors.New("upsert polaris CR failed")).Build()
				mockey.Mock(asynq.GetRetryCount).Return(10, true).Build()
				mockey.Mock(asynq.GetMaxRetry).Return(10, true).Build()

				err := runHandler(Args{
					AppID: app.ID, ConfigName: config.Name, EnvName: environment.Name,
				})
				Expect(err).To(MatchError("upsert polaris CR failed"))

				stored, getErr := store.Get(ctx, app.ID, config.Name)
				Expect(getErr).NotTo(HaveOccurred())
				Expect(stored.GetEnvState(environment.Name).LastError).To(Equal(
					"upsert polaris CR failed (retry 11/11, retries exhausted)",
				))
			})
		})

		It("should keep applying other envs when one env task fails", func() {
			config := createDeployedConfig("cfg-partial", []string{environment.Name, otherEnvironment.Name})
			Expect(appModelStore.CreateAppModel(ctx, &appmodel.AppModel{AppID: app.ID})).To(Succeed())
			DeferCleanup(func() { _ = appModelStore.DeleteAppModel(ctx, app.ID) })
			Expect(envStore.Delete(ctx, environment.ID)).To(Succeed())

			err := runHandler(Args{
				AppID: app.ID, ConfigName: config.Name, EnvName: environment.Name,
			})
			Expect(err).To(HaveOccurred())
			partial, getErr := store.Get(ctx, app.ID, config.Name)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(partial.GetEnvState(environment.Name).LastError).NotTo(BeEmpty())
			Expect(partial.GetEnvState(otherEnvironment.Name).LastError).To(BeEmpty())

			mockey.PatchConvey("remaining env still records its own result", GinkgoT(), func() {
				mockPolarisDiscoveryFailure()

				err = runHandler(Args{
					AppID: app.ID, ConfigName: config.Name, EnvName: otherEnvironment.Name,
				})
				Expect(err).To(HaveOccurred())

				stored, getErr := store.Get(ctx, app.ID, config.Name)
				Expect(getErr).NotTo(HaveOccurred())
				Expect(stored.GetEnvState(environment.Name).LastError).To(ContainSubstring("get env"))
				Expect(stored.GetEnvState(otherEnvironment.Name).LastError).To(
					ContainSubstring("test discovery error"),
				)
			})
		})
	})
})

func TestFormatLastError(t *testing.T) {
	err := stderrors.New("upsert polaris CR failed")
	tests := []struct {
		name      string
		exhausted bool
		attempt   int
		total     int
		want      string
	}{
		{"first failed attempt", false, 1, 11, "upsert polaris CR failed (retry 1/11)"},
		{"later failed attempt", false, 3, 11, "upsert polaris CR failed (retry 3/11)"},
		{"retries exhausted", true, 11, 11, "upsert polaris CR failed (retry 11/11, retries exhausted)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatLastError(err, tt.attempt, tt.total, tt.exhausted)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func mockPolarisDiscoveryFailure() {
	mockey.Mock(cluster.NewConfig).Return(&cluster.Config{ClusterID: "test-cluster"}).Build()
	mockey.Mock(discovery.GetGroupVersionResource).Return(nil, stderrors.New("test discovery error")).Build()
}

func newTestConfig(
	appID, name string,
	scopeEnvNames []string,
	envStates map[string]polaris.PolarisEnvState,
) *polaris.PolarisConfig {
	return &polaris.PolarisConfig{
		AppID: appID,
		Name:  name,
		Properties: polaris.Properties{
			InstanceKey:      "k1",
			PolarisName:      "polaris-service",
			PolarisNamespace: "Test",
			PolarisToken:     "t1",
			ServicePort:      8080,
		},
		ScopeEnvNames: scopeEnvNames,
		EnvStates:     envStates,
	}
}

func redeployFields(instanceKey, token string, servicePort int32) *polaris.RedeployRequiredFields {
	return &polaris.RedeployRequiredFields{
		InstanceKey:  instanceKey,
		PolarisToken: token,
		ServicePort:  servicePort,
	}
}

func envState(appliedFields *polaris.RedeployRequiredFields) polaris.PolarisEnvState {
	return polaris.PolarisEnvState{AppliedFields: appliedFields}
}
