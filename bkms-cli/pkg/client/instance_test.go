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

var _ = Describe("Instance APIs", func() {
	var cli *SvcBasedClient
	var server *httptest.Server
	var requestPath string
	var requestQuery url.Values

	BeforeEach(func() {
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestPath, requestQuery = r.URL.Path, r.URL.Query()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"count":"0","results":[]}}`))
		}))
		cli = &SvcBasedClient{cli: resty.New().SetBaseURL(server.URL)}
	})
	AfterEach(func() { server.Close() })

	It("sends pagination without IP filters by default", func() {
		_, err := cli.ListAppInstances(context.Background(), "app-1", "test", ListAppInstancesOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(requestPath).To(Equal("/bkms/v1/bkms-server/apps/app-1/envs/test/instances"))
		Expect(requestQuery.Get("page")).To(Equal("1"))
		Expect(requestQuery.Get("pageSize")).To(Equal("100"))
		Expect(requestQuery).NotTo(HaveKey("podIP"))
		Expect(requestQuery).NotTo(HaveKey("nodeIP"))
	})

	It("sends each IP filter value as a repeated query param", func() {
		_, err := cli.ListAppInstances(context.Background(), "app-1", "test", ListAppInstancesOptions{
			Page:    2,
			PodIPs:  []string{"10.0.0.1", "10.0.0.2"},
			NodeIPs: []string{"192.168.0.1"},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(requestQuery.Get("page")).To(Equal("2"))
		Expect(requestQuery["podIP"]).To(Equal([]string{"10.0.0.1", "10.0.0.2"}))
		Expect(requestQuery["nodeIP"]).To(Equal([]string{"192.168.0.1"}))
	})
})
