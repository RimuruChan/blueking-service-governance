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

package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/config"
	"github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/version"
)

var _ = Describe("Updater", func() {
	BeforeEach(func() {
		oldVersion, oldConfig := version.Version, config.G
		DeferCleanup(func() { version.Version, config.G = oldVersion, oldConfig })
		version.Version = "1.2.0"
		config.G = &config.Config{}
	})

	It("uses the default pair for an unconfigured installation", func() {
		c, err := newClient()
		Expect(err).NotTo(HaveOccurred())
		Expect(c.source.LatestVersionURL).To(Equal(config.DefaultUpdateLatestURL))
		Expect(c.source.DownloadURLTemplate).To(Equal(config.DefaultUpdateDownloadURLTemplate))
	})

	It("rejects a partly configured source instead of mixing it with public defaults", func() {
		config.G.Update.LatestVersionURL = "https://internal/latest.txt"
		_, err := newClient()
		Expect(errors.Is(err, ErrUpdateNotConfigured)).To(BeTrue())
	})

	DescribeTable("detects npm installs",
		func(path string, expected bool) {
			Expect(pathLooksLikeNPMInstall(path)).To(Equal(expected))
		},
		Entry("npm", "/usr/lib/node_modules/@blueking/bkms-cli/bin/bkms-cli", true),
		Entry("Windows", `C:\Users\me\npm\node_modules\@blueking\bkms-cli\bin\bkms-cli.exe`, true),
		Entry("standalone", "/usr/local/bin/bkms-cli", false),
	)

	DescribeTable("parses current versions",
		func(value, expected string) {
			v, err := parseVersion(value)
			if expected == "" {
				Expect(errors.Is(err, ErrInvalidVersion)).To(BeTrue())
				return
			}
			Expect(err).NotTo(HaveOccurred())
			Expect(v.String()).To(Equal(expected))
		},
		Entry("plain", "1.2.3", "1.2.3"),
		Entry("tag", " bkms-cli/v1.2.3\n", "1.2.3"),
		Entry("pseudo", "v1.2.4-0.20260910000000-abcdef123456", "1.2.4-0.20260910000000-abcdef123456"),
		Entry("dev", "dev", ""),
		Entry("empty", "", ""),
		Entry("partial", "v1.2", ""),
		Entry("leading zero", "v01.2.3", ""),
		Entry("other product", "bkms-server/v1.2.3", ""),
	)

	Describe("static version and release downloads", func() {
		var c *client
		var server *httptest.Server
		var responses map[string][]byte
		var requests []string
		const latestPath = "/bkms-cli/latest.txt"
		const releasePath = "/generic/bkms/public-assets/bkms-cli/releases/v1.3.0/"

		BeforeEach(func() {
			requests = nil
			responses = map[string][]byte{latestPath: []byte("1.3.0\n")}
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.URL.RequestURI())
				if r.URL.Path == "/oversize" {
					w.(http.Flusher).Flush()
					_, _ = w.Write([]byte("too large"))
					return
				}
				if r.URL.Path == "/forbidden" {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				data, ok := responses[r.URL.Path]
				if !ok {
					http.NotFound(w, r)
					return
				}
				_, _ = w.Write(data)
			}))
			DeferCleanup(server.Close)
			config.G.Update = config.UpdateSource{
				LatestVersionURL:    server.URL + latestPath,
				DownloadURLTemplate: server.URL + "/generic/bkms/public-assets/bkms-cli/releases/v{version}/{archive}",
			}
			var err error
			c, err = newClient()
			Expect(err).NotTo(HaveOccurred())
		})

		It("checks only the static version file, without querying GitHub API or npm", func() {
			info, err := c.check(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(info).To(Equal(Info{CurrentVersion: "1.2.0", LatestVersion: "1.3.0", Available: true}))
			Expect(requests).To(Equal([]string{latestPath}))
		})

		DescribeTable("compares stable versions",
			func(latest string, available bool) {
				responses[latestPath] = []byte(latest)
				info, err := c.check(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(info.Available).To(Equal(available))
			},
			Entry("same", "1.2.0", false),
			Entry("older", "1.1.0", false),
			Entry("newer with CRLF", "1.3.0\r\n", true),
		)

		DescribeTable("rejects invalid stable pointers",
			func(latest string) {
				responses[latestPath] = []byte(latest)
				_, err := c.check(context.Background())
				Expect(errors.Is(err, ErrInvalidVersion)).To(BeTrue())
			},
			Entry("empty", ""),
			Entry("prerelease", "1.3.0-rc.1"),
			Entry("metadata", "1.3.0+build"),
			Entry("path", "../1.3.0"),
			Entry("HTML", "<html>error</html>"),
			Entry("multiple lines", "1.3.0\n1.4.0"),
		)

		It("honors cancellation", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err := c.check(ctx)
			Expect(errors.Is(err, context.Canceled)).To(BeTrue())
		})

		It("reports a missing version file without falling back to another source", func() {
			delete(responses, latestPath)
			_, err := c.check(context.Background())
			Expect(errors.Is(err, ErrNoRelease)).To(BeTrue())
		})

		It("reports HTTP errors", func() {
			_, err := c.download(context.Background(), server.URL+"/forbidden", 128)
			Expect(err).To(MatchError(ContainSubstring("HTTP 403")))
		})

		It("limits downloads without Content-Length", func() {
			_, err := c.download(context.Background(), server.URL+"/oversize", 4)
			Expect(errors.Is(err, ErrDownloadTooLarge)).To(BeTrue())
		})

		It("limits the version file size", func() {
			responses[latestPath] = []byte(strings.Repeat("1", 129))
			_, err := c.check(context.Background())
			Expect(errors.Is(err, ErrDownloadTooLarge)).To(BeTrue())
		})

		DescribeTable("validates configured distribution assets before replacing the executable",
			func(scenario string, succeeds bool) {
				binary := []byte("new executable")
				archive := releaseArchive(binary)
				asset := archiveName("1.3.0")
				checksum := fmt.Sprintf("%x  %s\n", sha256.Sum256(archive), asset)
				switch scenario {
				case "wrong hash":
					checksum = fmt.Sprintf("%064d  %s\n", 0, asset)
				case "missing entry":
					checksum = fmt.Sprintf("%x  unrelated.tar.gz\n", sha256.Sum256(archive))
				}
				if scenario != "missing checksums" {
					responses[releasePath+"checksums.txt"] = []byte(checksum)
				}
				if scenario != "missing archive" {
					responses[releasePath+asset] = archive
				}
				executable := filepath.Join(GinkgoT().TempDir(), "bkms-cli")
				Expect(os.WriteFile(executable, []byte("old executable"), 0o755)).To(Succeed())

				err := c.install(context.Background(), "1.3.0", executable)
				installed, readErr := os.ReadFile(executable)
				Expect(readErr).NotTo(HaveOccurred())
				if succeeds {
					Expect(err).NotTo(HaveOccurred())
					Expect(installed).To(Equal(binary))
					Expect(requests).To(Equal([]string{
						"/generic/bkms/public-assets/bkms-cli/releases/v1.3.0/checksums.txt",
						"/generic/bkms/public-assets/bkms-cli/releases/v1.3.0/" + asset,
					}))
				} else {
					Expect(err).To(HaveOccurred())
					Expect(string(installed)).To(Equal("old executable"))
				}
			},
			Entry("verified asset", "valid", true),
			Entry("mismatched hash", "wrong hash", false),
			Entry("missing checksum entry", "missing entry", false),
			Entry("missing checksum file", "missing checksums", false),
			Entry("missing archive", "missing archive", false),
		)
	})
})

func releaseArchive(binary []byte) []byte {
	var buf bytes.Buffer
	if runtime.GOOS == "windows" {
		archive := zip.NewWriter(&buf)
		file, err := archive.Create("bkms-cli.exe")
		Expect(err).NotTo(HaveOccurred())
		_, err = file.Write(binary)
		Expect(err).NotTo(HaveOccurred())
		Expect(archive.Close()).To(Succeed())
	} else {
		compressed := gzip.NewWriter(&buf)
		archive := tar.NewWriter(compressed)
		Expect(archive.WriteHeader(&tar.Header{Name: "bkms-cli", Mode: 0o755, Size: int64(len(binary))})).To(Succeed())
		_, err := archive.Write(binary)
		Expect(err).NotTo(HaveOccurred())
		Expect(archive.Close()).To(Succeed())
		Expect(compressed.Close()).To(Succeed())
	}
	return buf.Bytes()
}
