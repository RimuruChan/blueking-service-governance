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

package hostport

import (
	"context"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InjectPodAnnotationsFromStore loads declared HostPort mappings and merges BCS
// random HostPort webhook annotations into pod template metadata.
//
// Non-federated environments are a no-op.
func InjectPodAnnotationsFromStore(
	ctx context.Context,
	store HostPortStore,
	appID string,
	isFederation bool,
	meta *metav1.ObjectMeta,
) error {
	if !isFederation || meta == nil {
		return nil
	}

	ports, err := store.ListPorts(ctx, appID)
	if err != nil {
		return errors.Wrap(err, "listing hostport mappings")
	}
	for k, v := range BuildPodAnnotations(ports) {
		if meta.Annotations == nil {
			meta.Annotations = make(map[string]string)
		}
		meta.Annotations[k] = v
	}
	return nil
}
