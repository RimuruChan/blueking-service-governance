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

// Package hostport manages app-level random HostPort port mappings for federated clusters.
package hostport

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
)

const (
	// AnnotationEnabledKey enables BCS random HostPort webhook injection.
	AnnotationEnabledKey = "randhostport.webhook.bkbcs.tencent.com"
	// AnnotationEnabledVal is the value that enables the webhook.
	AnnotationEnabledVal = "true"
	// AnnotationPortsKey declares container ports that need random HostPort mapping.
	AnnotationPortsKey = "ports.randhostport.webhook.bkbcs.tencent.com"
)

// HostPortConfig is the app-level HostPort mapping document (1:1 with app).
type HostPortConfig struct {
	AppID     string                      `bson:"appID"`
	Ports     []int32                     `bson:"ports"`
	EnvStates map[string]HostPortEnvState `bson:"envStates"`
	CreatedAt time.Time                   `bson:"createdAt"`
	UpdatedAt time.Time                   `bson:"updatedAt"`
}

// HostPortEnvState records the last successfully deployed HostPort ports for one env.
type HostPortEnvState struct {
	AppliedPorts []int32   `bson:"appliedPorts"`
	UpdatedAt    time.Time `bson:"updatedAt"`
}

// EnvStateView is the computed pending-deploy view for one federated environment.
type EnvStateView struct {
	AppliedPorts       []int32
	PendingAddPorts    []int32
	PendingRemovePorts []int32
}

// HostPorts is the API-facing aggregate of declared ports and federated env pending state.
type HostPorts struct {
	Ports     []int32
	EnvStates map[string]EnvStateView
}

// NormalizePorts returns a sorted unique copy of ports.
func NormalizePorts(ports []int32) []int32 {
	if len(ports) == 0 {
		return []int32{}
	}
	uniq := lo.Uniq(ports)
	slices.Sort(uniq)
	return uniq
}

// ValidateContainerPort checks that port is in [1, 65535].
func ValidateContainerPort(port int32) bool {
	return port >= 1 && port <= 65535
}

// DiffPorts returns ports in desired but not applied (add) and in applied but not desired (remove).
func DiffPorts(desired, applied []int32) (add, remove []int32) {
	desiredSet := lo.SliceToMap(NormalizePorts(desired), func(p int32) (int32, struct{}) {
		return p, struct{}{}
	})
	appliedSet := lo.SliceToMap(NormalizePorts(applied), func(p int32) (int32, struct{}) {
		return p, struct{}{}
	})

	for p := range desiredSet {
		if _, ok := appliedSet[p]; !ok {
			add = append(add, p)
		}
	}
	for p := range appliedSet {
		if _, ok := desiredSet[p]; !ok {
			remove = append(remove, p)
		}
	}
	return NormalizePorts(add), NormalizePorts(remove)
}

// ComputeEnvState builds the pending-deploy view for one env.
// state == nil means the env has never been reconciled after a HostPort-aware deploy.
func ComputeEnvState(desired []int32, state *HostPortEnvState) EnvStateView {
	desired = NormalizePorts(desired)
	if state == nil {
		if len(desired) == 0 {
			return EnvStateView{
				AppliedPorts:       []int32{},
				PendingAddPorts:    []int32{},
				PendingRemovePorts: []int32{},
			}
		}
		return EnvStateView{
			AppliedPorts:       []int32{},
			PendingAddPorts:    desired,
			PendingRemovePorts: []int32{},
		}
	}

	applied := NormalizePorts(state.AppliedPorts)
	add, remove := DiffPorts(desired, applied)
	return EnvStateView{
		AppliedPorts:       applied,
		PendingAddPorts:    add,
		PendingRemovePorts: remove,
	}
}

// FormatPortsAnnotationValue joins ports as a comma-separated ascending string.
func FormatPortsAnnotationValue(ports []int32) string {
	ports = NormalizePorts(ports)
	parts := make([]string, 0, len(ports))
	for _, p := range ports {
		parts = append(parts, strconv.FormatInt(int64(p), 10))
	}
	return strings.Join(parts, ",")
}

// BuildPodAnnotations returns HostPort webhook annotations for the given ports.
// Returns nil when ports is empty.
func BuildPodAnnotations(ports []int32) map[string]string {
	ports = NormalizePorts(ports)
	if len(ports) == 0 {
		return nil
	}
	return map[string]string{
		AnnotationEnabledKey: AnnotationEnabledVal,
		AnnotationPortsKey:   FormatPortsAnnotationValue(ports),
	}
}
