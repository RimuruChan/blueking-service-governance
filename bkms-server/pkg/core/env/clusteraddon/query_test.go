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

package clusteraddon_test

import (
	"context"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	helmrelease "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/storage/driver"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	clusteraddon "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
)

var _ = Describe("Query", func() {
	Describe("GetSupportedActions", func() {
		DescribeTable("should return correct actions for each status",
			func(status helmrelease.Status, expected []string) {
				Expect(clusteraddon.GetSupportedActions(status)).To(Equal(expected))
			},
			Entry("empty status (not installed)", helmrelease.Status(""), []string{"install"}),
			Entry("uninstalled", helm.StatusUninstalled, []string{"install"}),
			Entry("deployed", helm.StatusDeployed, []string{"upgrade", "uninstall"}),
			Entry("failed", helm.StatusFailed, []string{"install", "uninstall"}),
		)

		It("should return nil for pending status", func() {
			Expect(clusteraddon.GetSupportedActions(helm.StatusPendingInstall)).To(BeNil())
			Expect(clusteraddon.GetSupportedActions(helm.StatusPendingUpgrade)).To(BeNil())
			Expect(clusteraddon.GetSupportedActions(helm.StatusPendingRollback)).To(BeNil())
		})
	})

	Describe("BuildAddonInfoList", func() {
		var (
			repoIndex  *clusteraddon.RepoIndex
			mocker     *mockey.Mocker
			clusterCfg *cluster.Config
			ctx        context.Context
		)

		BeforeEach(func() {
			var err error
			clusterCfg, err = testutil.TestClusterConfig("")
			if errors.Is(err, testutil.ErrKubeConfigNotFound) {
				Skip(err.Error())
			}
			Expect(err).NotTo(HaveOccurred())
			mocker = mockey.Mock(cluster.NewConfig).Return(clusterCfg).Build()
			indexFile := &repo.IndexFile{
				Entries: map[string]repo.ChartVersions{
					"chart-a": {
						{Metadata: &chart.Metadata{Version: "2.0.0"}},
						{Metadata: &chart.Metadata{Version: "1.0.0"}},
					},
					"chart-b": {
						{Metadata: &chart.Metadata{Version: "3.0.0"}},
					},
				},
			}
			repoIndex = clusteraddon.NewRepoIndex(indexFile)

			ctx = context.Background()
		})

		AfterEach(func() {
			if mocker != nil {
				mocker.Release()
			}
		})

		It("should fill available versions from repo index", func() {
			addonDefs := []*clusteraddon.ClusterAddonDef{
				{
					Name: "addon-a",
					ChartInfo: clusteraddon.HelmChartInfo{
						ChartName:           "chart-a",
						DefaultChartVersion: "0.9.0",
						DefaultNamespace:    "ns-a",
					},
				},
			}

			// 注意：BuildAddonInfoList 内部会调用 FillAddonStatusFromCluster，
			// 该函数依赖真实集群连接，在无集群环境中会走错误分支（返回 install action）
			addons := clusteraddon.BuildAddonInfoList(
				ctx,
				addonDefs,
				addonEnvironment("fake-cluster", false),
				"",
				repoIndex,
			)

			Expect(addons).To(HaveLen(1))
			Expect(addons[0].ChartInfo.AvailableVersions).To(Equal([]string{"2.0.0", "1.0.0"}))
			// 仓库最新版本应覆盖定义中的默认版本
			Expect(addons[0].ChartInfo.DefaultChartVersion).To(Equal("2.0.0"))
			Expect(addons[0].InstallInfo.Namespace).To(Equal("ns-a"))
		})

		It("should keep default version when chart not found in repo", func() {
			addonDefs := []*clusteraddon.ClusterAddonDef{
				{
					Name: "addon-x",
					ChartInfo: clusteraddon.HelmChartInfo{
						ChartName:           "non-existent",
						DefaultChartVersion: "0.5.0",
					},
				},
			}

			addons := clusteraddon.BuildAddonInfoList(
				ctx,
				addonDefs,
				addonEnvironment("fake-cluster", false),
				"override-ns",
				repoIndex,
			)

			Expect(addons).To(HaveLen(1))
			Expect(addons[0].ChartInfo.AvailableVersions).To(BeNil())
			Expect(addons[0].ChartInfo.DefaultChartVersion).To(Equal("0.5.0"))
			Expect(addons[0].InstallInfo.Namespace).To(Equal("override-ns"))
		})

		It("should use addon default namespace when request namespace is empty", func() {
			addonDefs := []*clusteraddon.ClusterAddonDef{
				{
					Name: "addon-b",
					ChartInfo: clusteraddon.HelmChartInfo{
						ChartName:        "chart-b",
						DefaultNamespace: "custom-ns",
					},
				},
			}

			addons := clusteraddon.BuildAddonInfoList(
				ctx,
				addonDefs,
				addonEnvironment("fake-cluster", false),
				"",
				repoIndex,
			)

			Expect(addons).To(HaveLen(1))
			Expect(addons[0].InstallInfo.Namespace).To(Equal("custom-ns"))
		})
	})

	Describe("BuildAddonInfoList applicability", func() {
		DescribeTable("queries and returns only applicable addons",
			func(isFederation bool, expected []string) {
				var queried []string
				configMock := mockey.Mock(helm.NewActionConfiguration).To(
					func(_, namespace string, _ action.DebugLog) (*action.Configuration, error) {
						queried = append(queried, namespace)
						return &action.Configuration{}, nil
					},
				).Build()
				defer configMock.UnPatch()
				releaseMock := mockey.Mock(helm.GetReleaseStatus).Return(&helm.Release{
					DeployResult: helm.DeployResult{Status: helm.StatusDeployed},
				}, nil).Build()
				defer releaseMock.UnPatch()
				valuesMock := mockey.Mock(helm.GetReleaseValues).Return(nil, nil).Build()
				defer valuesMock.UnPatch()
				defs := []*clusteraddon.ClusterAddonDef{
					{Name: "supported", ChartInfo: clusteraddon.HelmChartInfo{DefaultNamespace: "supported"}},
					{
						Name: "unsupported", UnsupportedOnFederation: true,
						ChartInfo: clusteraddon.HelmChartInfo{DefaultNamespace: "unsupported"},
					},
				}
				addons := clusteraddon.BuildAddonInfoList(
					context.Background(),
					defs,
					addonEnvironment("cluster", isFederation),
					"",
					clusteraddon.NewRepoIndex(&repo.IndexFile{}),
				)
				var names []string
				for _, addon := range addons {
					names = append(names, addon.Name)
				}
				Expect(names).To(ConsistOf(expected))
				Expect(queried).To(ConsistOf(expected))
			},
			Entry("regular cluster", false, []string{"supported", "unsupported"}),
			Entry("federation cluster", true, []string{"supported"}),
		)
	})

	Describe("QueryAddonStatus", func() {
		var configMock, releaseMock *mockey.Mocker
		var def *clusteraddon.ClusterAddonDef
		BeforeEach(func() {
			def = &clusteraddon.ClusterAddonDef{Name: "test-addon"}
			configMock = mockey.Mock(helm.NewActionConfiguration).Return(&action.Configuration{}, nil).Build()
			releaseMock = mockey.Mock(helm.GetReleaseStatus).Return(&helm.Release{
				DeployResult: helm.DeployResult{Status: helm.StatusDeployed},
			}, nil).Build()
		})
		AfterEach(func() {
			releaseMock.UnPatch()
			configMock.UnPatch()
		})

		It("returns the status without reading release values", func() {
			def.ChartInfo.ReleaseName = "custom-release"
			cfg := &action.Configuration{}
			configMock.To(func(clusterID, namespace string, _ action.DebugLog) (*action.Configuration, error) {
				Expect(clusterID).To(Equal("target-cluster"))
				Expect(namespace).To(Equal("operator-ns"))
				return cfg, nil
			})
			releaseMock.To(func(actualCfg *action.Configuration, releaseName string) (*helm.Release, error) {
				Expect(actualCfg).To(BeIdenticalTo(cfg))
				Expect(releaseName).To(Equal("custom-release"))
				return &helm.Release{DeployResult: helm.DeployResult{Status: helm.StatusDeployed}}, nil
			})
			valuesMock := mockey.Mock(helm.GetReleaseValues).Return(nil, errors.New("values must not be read")).Build()
			defer valuesMock.UnPatch()
			status, err := clusteraddon.QueryAddonStatus(context.Background(), "target-cluster", "operator-ns", def)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(helm.StatusDeployed))
			Expect(valuesMock.Times()).To(BeZero())
		})
		It("reports a missing release as not found", func() {
			releaseMock.Return(nil, errors.Wrap(driver.ErrReleaseNotFound, "lookup"))
			status, err := clusteraddon.QueryAddonStatus(context.Background(), "cluster", "bcs-system", def)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(helm.StatusNotFound))
		})
		It("preserves release query errors and their cluster context", func() {
			cause := errors.New("access denied")
			releaseMock.Return(nil, cause)
			_, err := clusteraddon.QueryAddonStatus(context.Background(), "cluster", "bcs-system", def)
			Expect(errors.Is(err, cause)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring(def.Name))
			Expect(err.Error()).To(ContainSubstring("cluster"))
			Expect(err.Error()).To(ContainSubstring("bcs-system"))
		})
		It("reports an unknown release state as a query error", func() {
			releaseMock.Return(&helm.Release{DeployResult: helm.DeployResult{Status: helm.StatusUnknown}}, nil)
			_, err := clusteraddon.QueryAddonStatus(context.Background(), "cluster", "bcs-system", def)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(def.Name))
			Expect(err).To(MatchError(ContainSubstring("release status is unknown")))
		})
		It("does not query releases when configuration fails", func() {
			cause := errors.New("invalid cluster configuration")
			configMock.Return(nil, cause)
			_, err := clusteraddon.QueryAddonStatus(context.Background(), "cluster", "bcs-system", def)
			Expect(errors.Is(err, cause)).To(BeTrue())
			Expect(releaseMock.Times()).To(BeZero())
		})
	})

	Describe("InspectRequiredAddons", func() {
		var statusMock *mockey.Mocker
		var defs []*clusteraddon.ClusterAddonDef

		BeforeEach(func() {
			defs = []*clusteraddon.ClusterAddonDef{gameDeployDef(), hookOperatorDef(), agonesDef()}
			statusMock = mockey.Mock(clusteraddon.QueryAddonStatus).Return(helm.StatusDeployed, nil).Build()
		})
		AfterEach(func() { statusMock.UnPatch() })

		It("checks supported required addons while skipping unsupported ones in federation clusters", func() {
			defs[1].UnsupportedOnFederation = false
			statusMock.To(
				func(_ context.Context, _, _ string, def *clusteraddon.ClusterAddonDef) (clusteraddon.AddonStatus, error) {
					Expect(def.Name).To(Equal("bcs-hook-operator"))
					return helm.StatusNotFound, nil
				},
			)
			missing, err := clusteraddon.InspectRequiredAddons(
				context.Background(),
				defs,
				bkmsapp.AppTypeTRPC,
				addonEnvironment("cluster", true),
				"",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(
				missing,
			).To(ConsistOf(clusteraddon.AddonReference{Name: "bcs-hook-operator", DisplayName: "Hook-operator"}))
			Expect(statusMock.Times()).To(Equal(1))
		})

		It("queries only required addons in their configured namespaces", func() {
			defs[0].ChartInfo.DefaultNamespace = "game-ns"
			defs[1].ChartInfo.DefaultNamespace = "hook-ns"
			statusMock.To(
				func(_ context.Context, clusterID, namespace string, def *clusteraddon.ClusterAddonDef) (clusteraddon.AddonStatus, error) {
					Expect(clusterID).To(Equal("cluster"))
					if def.Name == "bcs-hook-operator" {
						Expect(namespace).To(Equal("hook-ns"))
						return helm.StatusNotFound, nil
					}
					Expect(def.Name).To(Equal("bcs-gamedeployment-operator"))
					Expect(namespace).To(Equal("game-ns"))
					return helm.StatusDeployed, nil
				},
			)
			missing, err := clusteraddon.InspectRequiredAddons(
				context.Background(),
				defs,
				bkmsapp.AppTypeTRPC,
				addonEnvironment("cluster", false),
				"",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(
				missing,
			).To(Equal([]clusteraddon.AddonReference{{Name: "bcs-hook-operator", DisplayName: "Hook-operator"}}))
			Expect(statusMock.Times()).To(Equal(2))
		})

		DescribeTable("classifies release states",
			func(status clusteraddon.AddonStatus, isMissing bool) {
				statusMock.Return(status, nil)
				missing, err := clusteraddon.InspectRequiredAddons(
					context.Background(),
					defs,
					bkmsapp.AppTypeTAF,
					addonEnvironment("cluster", false),
					"",
				)
				Expect(err).NotTo(HaveOccurred())
				if isMissing {
					Expect(missing).To(ConsistOf(
						clusteraddon.AddonReference{Name: "bcs-gamedeployment-operator", DisplayName: "Gamedeploy"},
						clusteraddon.AddonReference{Name: "bcs-hook-operator", DisplayName: "Hook-operator"},
					))
				} else {
					Expect(missing).To(BeEmpty())
				}
			},
			Entry("deployed", helm.StatusDeployed, false),
			Entry("not found", helm.StatusNotFound, true),
			Entry("failed", helm.StatusFailed, true),
			Entry("pending upgrade", helm.StatusPendingUpgrade, true),
		)

		It("preserves the original query error", func() {
			cause := errors.New("cluster access denied")
			statusMock.Return(helm.StatusUnknown, cause)
			_, err := clusteraddon.InspectRequiredAddons(
				context.Background(),
				defs,
				bkmsapp.AppTypeTRPC,
				addonEnvironment("cluster", false),
				"",
			)
			Expect(errors.Is(err, cause)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("cluster access denied"))
		})

		It("does not query optional addons", func() {
			defs[0].OptionalForAppTypes = []string{bkmsapp.AppTypeHelm}
			statusMock.Return(helm.StatusNotFound, nil)
			missing, err := clusteraddon.InspectRequiredAddons(
				context.Background(),
				defs,
				bkmsapp.AppTypeHelm,
				addonEnvironment("cluster", false),
				"",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(missing).To(BeEmpty())
			Expect(statusMock.Times()).To(BeZero())
		})
	})

	Describe("ApplicableAddonDefs", func() {
		DescribeTable("filters definitions by cluster applicability",
			func(isFederation bool, expectedNames []string) {
				defs := []*clusteraddon.ClusterAddonDef{gameDeployDef(), hookOperatorDef(), agonesDef()}
				applicable := clusteraddon.ApplicableAddonDefs(defs, addonEnvironment("cluster", isFederation))
				names := make([]string, 0, len(applicable))
				for _, def := range applicable {
					names = append(names, def.Name)
				}
				Expect(names).To(Equal(expectedNames))
			},
			Entry("regular clusters", false, []string{"bcs-gamedeployment-operator", "bcs-hook-operator", "agones"}),
			Entry("federation clusters", true, []string{"agones"}),
		)
		It("accepts an empty definition list", func() {
			Expect(clusteraddon.ApplicableAddonDefs(nil, addonEnvironment("cluster", true))).To(BeEmpty())
		})
	})
})

func gameDeployDef() *clusteraddon.ClusterAddonDef {
	return &clusteraddon.ClusterAddonDef{
		Name:                    "bcs-gamedeployment-operator",
		DisplayName:             "Gamedeploy",
		UnsupportedOnFederation: true,
		RequiredForAppTypes:     []string{bkmsapp.AppTypeTRPC, bkmsapp.AppTypeTAF},
	}
}

func hookOperatorDef() *clusteraddon.ClusterAddonDef {
	return &clusteraddon.ClusterAddonDef{
		Name:                    "bcs-hook-operator",
		DisplayName:             "Hook-operator",
		UnsupportedOnFederation: true,
		RequiredForAppTypes:     []string{bkmsapp.AppTypeTRPC, bkmsapp.AppTypeTAF},
	}
}

func agonesDef() *clusteraddon.ClusterAddonDef {
	return &clusteraddon.ClusterAddonDef{
		Name:                "agones",
		DisplayName:         "agones",
		RequiredForAppTypes: []string{bkmsapp.AppTypeAgones},
	}
}

var _ = Describe("RequiredAddonsNotInstalledError", func() {
	It("identifies every missing addon by its stable name", func() {
		err := &clusteraddon.RequiredAddonsNotInstalledError{Missing: []clusteraddon.AddonReference{
			{Name: "game", DisplayName: "Same display name"},
			{Name: "hook", DisplayName: "Same display name"},
		}}
		Expect(err).To(MatchError("required cluster addons not installed: game, hook"))
	})
})

// addonEnvironment 构造组件测试所需的内存环境对象，仅设置集群信息，不写入数据库。
func addonEnvironment(clusterID string, isFederation bool) *envmodel.Environment {
	return &envmodel.Environment{Cluster: envmodel.BizCluster{ClusterID: clusterID, IsFederation: isFederation}}
}
