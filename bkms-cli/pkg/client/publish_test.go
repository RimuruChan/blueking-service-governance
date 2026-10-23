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

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DevMode publish record queries", func() {
	var cli *SvcBasedClient
	var server *httptest.Server
	var query url.Values

	BeforeEach(func() {
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			query = r.URL.Query()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"count":"2","results":[]}}`))
		}))
		cli = &SvcBasedClient{cli: resty.New().SetBaseURL(server.URL)}
	})
	AfterEach(func() { server.Close() })

	It("sends list conditions as repeated query parameters", func() {
		_, count, err := cli.ListDevModePublishRecords(context.Background(), "app-1", DevModePublishRecordsQuery{
			EnvNames:  []string{"stag", "test"},
			RecordIDs: []string{"id-1", "id-2"},
			Statuses:  []string{"pending", "running"},
			Keyword:   "server",
			Page:      2,
			PageSize:  100,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(Equal(int64(2)))
		Expect(query["envNames"]).To(Equal([]string{"stag", "test"}))
		Expect(query["recordIDs"]).To(Equal([]string{"id-1", "id-2"}))
		Expect(query["statuses"]).To(Equal([]string{"pending", "running"}))
		Expect(query.Get("keyword")).To(Equal("server"))
		Expect(query.Get("page")).To(Equal("2"))
		Expect(query.Get("pageSize")).To(Equal("100"))
	})
})
