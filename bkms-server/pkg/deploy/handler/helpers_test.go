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

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
)

var _ = Describe("AppModel deployment error responses", func() {
	DescribeTable("aborts the request with the appropriate code and component details",
		func(cause error, status int, code bkerrs.ErrCode, detailCode bkerrs.ErrDetailCode, modules []string) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(bkerrs.ErrorHandler())
			router.GET("/deploy", func(c *gin.Context) {
				abortWithAppModelDeployError(c, errors.Wrap(cause, "deploy service"), "cluster-1", "deploy app")
				Expect(c.IsAborted()).To(BeTrue())
				Expect(errors.Is(c.Errors.Last().Err, cause)).To(BeTrue())
			}, func(_ *gin.Context) {
				Fail("the next handler must not run")
			})
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/deploy", nil))

			Expect(rec.Code).To(Equal(status))
			var output bkerrs.GinErrorOutput
			Expect(json.Unmarshal(rec.Body.Bytes(), &output)).To(Succeed())
			Expect(output.Error.Code).To(Equal(code))
			Expect(output.Error.Message).To(ContainSubstring(cause.Error()))
			Expect(output.Error.Details).To(HaveLen(len(modules)))
			for i, module := range modules {
				Expect(output.Error.Details[i]["module"]).To(Equal(module))
				Expect(output.Error.Details[i]["code"]).To(Equal(string(detailCode)))
			}
		},
		Entry(
			"missing addons retain separate stable component identifiers",
			&clusteraddon.RequiredAddonsNotInstalledError{Missing: []clusteraddon.AddonReference{
				{Name: "game", DisplayName: "Operator"}, {Name: "hook", DisplayName: "Operator"},
			}},
			http.StatusNotFound,
			bkerrs.ErrCodeNotFound,
			bkerrs.ErrDetailCodeComponentNotInstalled,
			[]string{"game", "hook"},
		),
		Entry("query failures use the default internal error response",
			errors.Wrap(errors.New("cluster unavailable"), "query addon game status"),
			http.StatusInternalServerError, bkerrs.ErrCodeInternalServerError,
			bkerrs.ErrDetailCode(""), []string{}),
		Entry("ordinary failures use the default internal error response",
			errors.New("database unavailable"), http.StatusInternalServerError, bkerrs.ErrCodeInternalServerError,
			bkerrs.ErrDetailCode(""), []string{}),
	)
})
