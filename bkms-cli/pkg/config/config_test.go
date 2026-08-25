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

var _ = Describe("Config", func() {
	var (
		tmpDir     string
		cfgFile    string
		origConfig *Config
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "bkms-cli-config-test-*")
		Expect(err).ToNot(HaveOccurred())

		// Use a test-only config path to avoid clobbering a real local config.
		cfgFile = filepath.Join(tmpDir, ".bkms", "test-config.yaml")
		cfgFilePath = cfgFile

		origConfig = G
		G = &Config{}
	})

	AfterEach(func() {
		G = origConfig
		os.RemoveAll(tmpDir)
	})

	Describe("Load", func() {
		It("loads an existing file and trims trailing slashes", func() {
			Expect(os.MkdirAll(filepath.Dir(cfgFile), 0o755)).To(Succeed())

			cfg := &Config{
				BkmsBaseURL: "http://existing.example.com/",
				Username:    "testuser",
				AccessToken: "test-token",
				Defaults:    Defaults{WorkspaceID: "ws-123"},
			}
			data, err := yaml.Marshal(cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(os.WriteFile(cfgFile, data, 0o600)).To(Succeed())

			conf, err := G.Load()
			Expect(err).ToNot(HaveOccurred())
			Expect(conf.BkmsBaseURL).To(Equal("http://existing.example.com"))
			Expect(conf.Username).To(Equal("testuser"))
			Expect(conf.AccessToken).To(Equal("test-token"))
			Expect(conf.Defaults.WorkspaceID).To(Equal("ws-123"))
			Expect(G).To(Equal(conf))
		})

		It("creates an empty config when the file is missing", func() {
			conf, err := G.Load()
			Expect(err).ToNot(HaveOccurred())
			Expect(conf.BkmsBaseURL).To(Equal(""))
			Expect(G).To(Equal(conf))

			_, statErr := os.Stat(cfgFile)
			Expect(statErr).ToNot(HaveOccurred())
		})

		It("returns an error for invalid YAML", func() {
			Expect(os.MkdirAll(filepath.Dir(cfgFile), 0o755)).To(Succeed())
			Expect(os.WriteFile(cfgFile, []byte("invalid: [yaml: content"), 0o600)).To(Succeed())

			_, err := G.Load()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Dump", func() {
		It("round-trips config through Dump and Load", func() {
			Expect(os.MkdirAll(filepath.Dir(cfgFile), 0o755)).To(Succeed())

			G = &Config{
				BkmsBaseURL: "http://dump.example.com",
				Username:    "dumpuser",
				AccessToken: "dump-token",
				Defaults:    Defaults{WorkspaceID: "ws-dump"},
			}
			Expect(G.Dump()).To(Succeed())

			conf, err := G.Load()
			Expect(err).ToNot(HaveOccurred())
			Expect(conf.BkmsBaseURL).To(Equal("http://dump.example.com"))
			Expect(conf.Username).To(Equal("dumpuser"))
			Expect(conf.AccessToken).To(Equal("dump-token"))
			Expect(conf.Defaults.WorkspaceID).To(Equal("ws-dump"))
		})
	})

	Describe("String", func() {
		It("shows normal fields and redacts secrets", func() {
			G = &Config{
				BkmsBaseURL: "http://string-test.example.com",
				Username:    "stringuser",
				AccessToken: "secret-token",
			}

			s := G.String()
			Expect(s).To(ContainSubstring("http://string-test.example.com"))
			Expect(s).To(ContainSubstring("stringuser"))
			Expect(s).ToNot(ContainSubstring("secret-token"))
			Expect(s).To(ContainSubstring("[REDACTED]"))
		})
	})

	Describe("SetBkmsBaseURL", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(filepath.Dir(cfgFile), 0o755)).To(Succeed())
			G = &Config{}
		})

		It("writes the URL and persists it", func() {
			updated, err := G.SetBkmsBaseURL("http://bkms.example.com/", false)
			Expect(err).ToNot(HaveOccurred())
			Expect(updated).To(BeTrue())
			Expect(G.BkmsBaseURL).To(Equal("http://bkms.example.com"))

			conf, err := (&Config{}).Load()
			Expect(err).ToNot(HaveOccurred())
			Expect(conf.BkmsBaseURL).To(Equal("http://bkms.example.com"))
		})

		It("does not overwrite an existing value with ifUnset", func() {
			G.BkmsBaseURL = "http://existing.example.com"
			Expect(G.Dump()).To(Succeed())

			updated, err := G.SetBkmsBaseURL("http://new.example.com", true)
			Expect(err).ToNot(HaveOccurred())
			Expect(updated).To(BeFalse())
			Expect(G.BkmsBaseURL).To(Equal("http://existing.example.com"))
		})

		It("rejects an empty URL", func() {
			_, err := G.SetBkmsBaseURL("  ", false)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("RequireBkmsBaseURL", func() {
		It("returns setup guidance when unset", func() {
			G = &Config{}
			err := G.RequireBkmsBaseURL()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("bkms-cli config set"))
		})

		It("passes when the URL is configured", func() {
			G = &Config{BkmsBaseURL: "https://bkms.example.com"}
			Expect(G.RequireBkmsBaseURL()).To(Succeed())
			Expect(G.HasBkmsBaseURL()).To(BeTrue())
		})
	})
})
