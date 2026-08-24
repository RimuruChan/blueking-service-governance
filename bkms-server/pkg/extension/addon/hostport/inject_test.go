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

package hostport_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/hostport"
)

var _ = Describe("InjectPodAnnotationsFromStore", func() {
	var (
		ctx       context.Context
		diApp     *fxtest.App
		store     hostport.HostPortStore
		testAppID string
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			hostport.FxModule,
			fx.Populate(&store),
		)
		diApp.RequireStart()
		testAppID = "hostport-inject-" + stringx.Random(6)
	})

	AfterEach(func() {
		_ = store.DeleteByApp(ctx, testAppID)
		diApp.RequireStop()
	})

	It("is a no-op for non-federation environments", func() {
		_, err := store.AddPort(ctx, testAppID, 80)
		Expect(err).NotTo(HaveOccurred())

		meta := &metav1.ObjectMeta{}
		err = hostport.InjectPodAnnotationsFromStore(ctx, store, testAppID, false, meta)
		Expect(err).NotTo(HaveOccurred())
		Expect(meta.Annotations).To(BeNil())
	})

	It("merges webhook annotations for federation environments", func() {
		_, err := store.AddPort(ctx, testAppID, 8080)
		Expect(err).NotTo(HaveOccurred())
		_, err = store.AddPort(ctx, testAppID, 80)
		Expect(err).NotTo(HaveOccurred())

		meta := &metav1.ObjectMeta{Annotations: map[string]string{"keep": "1"}}
		err = hostport.InjectPodAnnotationsFromStore(ctx, store, testAppID, true, meta)
		Expect(err).NotTo(HaveOccurred())
		Expect(meta.Annotations).To(Equal(map[string]string{
			"keep":                        "1",
			hostport.AnnotationEnabledKey: hostport.AnnotationEnabledVal,
			hostport.AnnotationPortsKey:   "80,8080",
		}))
	})

	It("writes nothing when federation env has no declared ports", func() {
		meta := &metav1.ObjectMeta{}
		err := hostport.InjectPodAnnotationsFromStore(ctx, store, testAppID, true, meta)
		Expect(err).NotTo(HaveOccurred())
		Expect(meta.Annotations).To(BeNil())
	})
})
