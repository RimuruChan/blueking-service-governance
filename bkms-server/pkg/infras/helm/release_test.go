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

package helm

import (
	"io"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
)

var _ = Describe("GetReleaseByChart", func() {
	var cfg *action.Configuration
	var addRelease func(string, string, int, release.Status)
	BeforeEach(func() {
		cfg = &action.Configuration{
			Releases:   storage.Init(driver.NewMemory()),
			KubeClient: &kubefake.PrintingKubeClient{Out: io.Discard},
		}
		addRelease = func(name, chartName string, revision int, status release.Status) {
			Expect(cfg.Releases.Create(&release.Release{
				Name: name, Namespace: "bcs-system", Version: revision,
				Info:   &release.Info{Status: status},
				Config: map[string]any{"replicas": 2},
				Chart:  &chart.Chart{Metadata: &chart.Metadata{Name: chartName, Version: "1.2.3"}},
			})).To(Succeed())
		}
	})

	DescribeTable("uses the latest revision including non-deployed states",
		func(status release.Status) {
			addRelease("custom-hook", "bcs-hook-operator", 1, release.StatusSuperseded)
			addRelease("custom-hook", "bcs-hook-operator", 2, status)
			result, err := GetReleaseByChart(cfg, "bcs-hook-operator", "bcs-hook-operator")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Version).To(Equal("2"))
			Expect(result.DeployResult.Status).To(Equal(status))
		},
		Entry("deployed", release.StatusDeployed),
		Entry("failed", release.StatusFailed),
		Entry("pending", release.StatusPendingUpgrade),
		Entry("unknown", release.StatusUnknown),
	)

	It("does not match an old deployed revision after uninstall", func() {
		addRelease("old-hook", "bcs-hook-operator", 1, release.StatusDeployed)
		addRelease("old-hook", "bcs-hook-operator", 2, release.StatusUninstalled)
		_, err := GetReleaseByChart(cfg, "bcs-hook-operator", "bcs-hook-operator")
		Expect(errors.Is(err, driver.ErrReleaseNotFound)).To(BeTrue())
	})

	It("does not match chart names by prefix or substring", func() {
		addRelease("hook", "bcs-hook-operator", 1, release.StatusDeployed)
		_, err := GetReleaseByChart(cfg, "hook-operator", "hook")
		Expect(errors.Is(err, driver.ErrReleaseNotFound)).To(BeTrue())
	})

	It("prefers a matching release name over lexical order without preferring deployed status", func() {
		addRelease("hook-a", "bcs-hook-operator", 1, release.StatusDeployed)
		addRelease("hook-z", "bcs-hook-operator", 1, release.StatusFailed)
		result, err := GetReleaseByChart(cfg, "bcs-hook-operator", "hook-z")
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Name).To(Equal("hook-z"))
		Expect(result.DeployResult.Status).To(Equal(release.StatusFailed))
	})

	It("filters by chart and selects the first matching name when the preferred release uses another chart", func() {
		addRelease("preferred", "another-chart", 1, release.StatusDeployed)
		addRelease("hook-z", "bcs-hook-operator", 1, release.StatusDeployed)
		addRelease("hook-a", "bcs-hook-operator", 1, release.StatusFailed)
		result, err := GetReleaseByChart(cfg, "bcs-hook-operator", "preferred")
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Name).To(Equal("hook-a"))
		Expect(result.Namespace).To(Equal("bcs-system"))
		Expect(result.Chart.Name).To(Equal("bcs-hook-operator"))
		Expect(result.Chart.Version).To(Equal("1.2.3"))
		Expect(result.Values["replicas"]).To(BeNumerically("==", 2))
		Expect(result.DeployResult.Status).To(Equal(release.StatusFailed))
	})

	It("preserves failures while listing releases", func() {
		mockey.PatchConvey("list failure", GinkgoT(), func() {
			cause := errors.New("access denied")
			mockey.Mock((*action.List).Run).Return(nil, cause).Build()
			_, err := GetReleaseByChart(cfg, "bcs-hook-operator", "bcs-hook-operator")
			Expect(errors.Is(err, cause)).To(BeTrue())
		})
	})
})
