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

package config

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("Update source configuration", func() {
	var internal UpdateSource

	BeforeEach(func() {
		oldPath, oldConfig := cfgFilePath, G
		DeferCleanup(func() { cfgFilePath, G = oldPath, oldConfig })
		cfgFilePath = filepath.Join(GinkgoT().TempDir(), "config.yaml")
		G = &Config{}
		internal = UpdateSource{
			LatestVersionURL:    "https://artifacts.example.com/bkms-cli/releases/latest.txt",
			DownloadURLTemplate: "https://artifacts.example.com/bkms-cli/releases/v{version}/{archive}",
		}
	})

	It("uses official defaults without persisting them", func() {
		source, err := G.UpdateSource()
		Expect(err).NotTo(HaveOccurred())
		Expect(source.LatestVersionURL).To(Equal(DefaultUpdateLatestURL))
		Expect(source.DownloadURLTemplate).To(Equal(DefaultUpdateDownloadURLTemplate))
		Expect(G.Update.IsZero()).To(BeTrue())
		data, err := yaml.Marshal(G)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).NotTo(ContainSubstring("update:"))
	})

	It("persists and reloads the internal source as a pair", func() {
		updated, err := G.SetEndpoints("", internal, true)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).To(BeTrue())
		loaded, err := (&Config{}).Load()
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded.Update).To(Equal(internal))
		source, err := loaded.UpdateSource()
		Expect(err).NotTo(HaveOccurred())
		Expect(source.DownloadURL("1.0.2", "checksums.txt")).To(Equal(
			"https://artifacts.example.com/bkms-cli/releases/v1.0.2/checksums.txt"))
		Expect(loaded.String()).To(ContainSubstring("latestVersionUrl: " + internal.LatestVersionURL))
	})

	It("preserves an existing source when ifUnset is explicitly requested", func() {
		G.Update = internal
		updated, err := G.SetEndpoints("", UpdateSource{
			LatestVersionURL:    DefaultUpdateLatestURL,
			DownloadURLTemplate: DefaultUpdateDownloadURLTemplate,
		}, true)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).To(BeFalse())
		Expect(G.Update).To(Equal(internal))
	})

	It("never fills a partly configured source from another channel", func() {
		G.Update.LatestVersionURL = "https://old.internal/latest.txt"
		updated, err := G.SetEndpoints("https://bkms.example.com", internal, true)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).To(BeTrue())
		Expect(G.Update.LatestVersionURL).To(Equal("https://old.internal/latest.txt"))
		Expect(G.Update.DownloadURLTemplate).To(BeEmpty())
		_, err = G.UpdateSource()
		Expect(err).To(HaveOccurred())
	})

	It("does not save the service URL when the update pair is invalid", func() {
		G.BkmsBaseURL = "https://old.example.com"
		Expect(G.Dump()).To(Succeed())
		before, err := os.ReadFile(cfgFilePath)
		Expect(err).NotTo(HaveOccurred())
		_, err = G.SetEndpoints(
			"https://new.example.com",
			UpdateSource{LatestVersionURL: internal.LatestVersionURL},
			false,
		)
		Expect(err).To(HaveOccurred())
		after, err := os.ReadFile(cfgFilePath)
		Expect(err).NotTo(HaveOccurred())
		Expect(after).To(Equal(before))
		Expect(G.BkmsBaseURL).To(Equal("https://old.example.com"))
	})

	It("allows an explicit source change and preserves unrelated configuration", func() {
		G.Update = internal
		G.BkmsBaseURL = "https://existing.example.com"
		G.Username, G.AccessToken = "user", "token"
		replacement := UpdateSource{
			LatestVersionURL:    "http://mirror/latest.txt",
			DownloadURLTemplate: "http://mirror/{version}/{archive}",
		}
		_, err := G.SetEndpoints("", replacement, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(G.Update).To(Equal(replacement))
		Expect(G.BkmsBaseURL).To(Equal("https://existing.example.com"))
		Expect(G.Username).To(Equal("user"))
		Expect(G.AccessToken).To(Equal("token"))
	})

	It("overwrites supplied service and update URLs together", func() {
		G.BkmsBaseURL = "https://old.example.com"
		G.Update = internal
		official := UpdateSource{
			LatestVersionURL:    DefaultUpdateLatestURL,
			DownloadURLTemplate: DefaultUpdateDownloadURLTemplate,
		}
		updated, err := G.SetEndpoints("https://new.example.com/", official, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).To(BeTrue())
		loaded, err := (&Config{}).Load()
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded.BkmsBaseURL).To(Equal("https://new.example.com"))
		Expect(loaded.Update).To(Equal(official))
	})

	It("does not change in-memory config when persistence fails", func() {
		cfgFilePath = filepath.Join(GinkgoT().TempDir(), "missing", "config.yaml")
		_, err := G.SetEndpoints("https://bkms.example.com", internal, false)
		Expect(err).To(HaveOccurred())
		Expect(G.Update.IsZero()).To(BeTrue())
		Expect(G.BkmsBaseURL).To(BeEmpty())
	})

	DescribeTable(
		"validates template syntax and HTTP endpoints",
		func(latest, template string, valid bool) {
			source := UpdateSource{LatestVersionURL: latest, DownloadURLTemplate: template}
			if valid {
				Expect(source.Validate()).To(Succeed())
			} else {
				Expect(source.Validate()).NotTo(Succeed())
			}
		},
		Entry("HTTPS", "https://example.com/latest.txt", "https://example.com/v{version}/{archive}", true),
		Entry("internal HTTP", "http://mirror/latest.txt", "http://mirror/{archive}", true),
		Entry(
			"query parameters",
			"https://example.com/latest",
			"https://example.com/download?version={version}&file={archive}",
			true,
		),
		Entry("partial", "https://example.com/latest", "", false),
		Entry("file URL", "file:///latest.txt", "https://example.com/{archive}", false),
		Entry("missing archive", "https://example.com/latest", "https://example.com/{version}/binary", false),
		Entry("unknown placeholder", "https://example.com/latest", "https://example.com/{os}/{archive}", false),
		Entry("fragment", "https://example.com/latest#fragment", "https://example.com/{archive}", false),
	)
})
