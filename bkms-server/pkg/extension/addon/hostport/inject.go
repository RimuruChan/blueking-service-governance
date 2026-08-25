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
	"fmt"
	"maps"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InjectFromStore loads declared HostPort mappings and, for federated environments:
//  1. merges BCS random HostPort webhook annotations into pod template metadata
//  2. upserts matching containerPorts on the main container
//
// The webhook only allocates HostPort / injects BCS_RANDHOSTPORT_* when the
// container declares the corresponding containerPort (same idea as the former
// Polaris health-check port injection).
//
// Returns the ports snapshot used for this inject (empty when none / non-federation),
// so deploy reconcile can record the same applied set without a second ListPorts.
//
// Non-federated environments are a no-op.
func InjectFromStore(
	ctx context.Context,
	store HostPortStore,
	appID string,
	isFederation bool,
	meta *metav1.ObjectMeta,
	containers []corev1.Container,
	mainContainerName string,
) ([]int32, error) {
	if !isFederation {
		return nil, nil
	}

	ports, err := store.ListPorts(ctx, appID)
	if err != nil {
		return nil, errors.Wrap(err, "listing hostport mappings")
	}
	ports = NormalizePorts(ports)
	if len(ports) == 0 {
		return ports, nil
	}

	if meta != nil {
		if meta.Annotations == nil {
			meta.Annotations = make(map[string]string)
		}
		maps.Copy(meta.Annotations, BuildPodAnnotations(ports))
	}

	for i := range containers {
		if containers[i].Name != mainContainerName {
			continue
		}
		injectContainerPorts(ports, &containers[i])
		break
	}
	return ports, nil
}

// injectContainerPorts upserts HostPort-declared ports onto the container,
// mirroring the former Polaris injectContainerPorts merge-by-ContainerPort behavior.
func injectContainerPorts(ports []int32, container *corev1.Container) {
	for _, port := range NormalizePorts(ports) {
		upsertContainerPort(&container.Ports, corev1.ContainerPort{
			Name:          fmt.Sprintf("hostport-%d", port),
			ContainerPort: port,
			Protocol:      corev1.ProtocolTCP,
		})
	}
}

func upsertContainerPort(items *[]corev1.ContainerPort, value corev1.ContainerPort) {
	for idx := range *items {
		if (*items)[idx].ContainerPort == value.ContainerPort {
			(*items)[idx] = value
			return
		}
	}
	*items = append(*items, value)
}
