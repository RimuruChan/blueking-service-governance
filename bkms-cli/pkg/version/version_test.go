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

package version

import (
	"fmt"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("UserAgent", func() {
	var originalVersion string

	BeforeEach(func() {
		originalVersion = Version
		DeferCleanup(func() { Version = originalVersion })
	})

	DescribeTable("assembles product/version (os/arch)",
		func(version, wantVersion string) {
			Version = version
			Expect(
				UserAgent(),
			).To(Equal(fmt.Sprintf("bkms-cli/%s (%s/%s)", wantVersion, runtime.GOOS, runtime.GOARCH)))
		},
		Entry("release version", "1.2.3", "1.2.3"),
		Entry("strips leading v", "v1.2.3", "1.2.3"),
		Entry("empty version uses dev", "", "dev"),
	)
})
