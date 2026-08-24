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

package serializer

import "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/hostport"

// AppURIInput binds the appID path parameter.
type AppURIInput struct {
	AppID string `uri:"appID" binding:"required"`
}

// DeleteHostPortURIInput binds appID and containerPort path parameters.
type DeleteHostPortURIInput struct {
	AppID         string `uri:"appID" binding:"required"`
	ContainerPort int32  `uri:"containerPort" binding:"required"`
}

// CreateHostPortInput is the request body for adding a hostport.
type CreateHostPortInput struct {
	ContainerPort int32 `json:"containerPort" binding:"required"`
}

// HostPortEnvStateOutput is one federated environment's HostPort status.
type HostPortEnvStateOutput struct {
	AppliedPorts       []int32 `json:"appliedPorts"`
	PendingAddPorts    []int32 `json:"pendingAddPorts"`
	PendingRemovePorts []int32 `json:"pendingRemovePorts"`
}

// HostPortsOutput is the response for listing hostports (ports + federated env states).
type HostPortsOutput struct {
	Ports     []int32                           `json:"ports"`
	EnvStates map[string]HostPortEnvStateOutput `json:"envStates"`
}

// FromPorts builds HostPortsOutput with ports only (envStates empty).
func (o *HostPortsOutput) FromPorts(ports []int32) *HostPortsOutput {
	if ports == nil {
		ports = []int32{}
	}
	o.Ports = ports
	o.EnvStates = map[string]HostPortEnvStateOutput{}
	return o
}

// FromPortsAndViews builds HostPortsOutput with ports and env states.
func (o *HostPortsOutput) FromPortsAndViews(
	ports []int32,
	views map[string]hostport.EnvStateView,
) *HostPortsOutput {
	if ports == nil {
		ports = []int32{}
	}
	o.Ports = ports
	o.EnvStates = make(map[string]HostPortEnvStateOutput, len(views))
	for name, view := range views {
		o.EnvStates[name] = HostPortEnvStateOutput{
			AppliedPorts:       nonNilPorts(view.AppliedPorts),
			PendingAddPorts:    nonNilPorts(view.PendingAddPorts),
			PendingRemovePorts: nonNilPorts(view.PendingRemovePorts),
		}
	}
	return o
}

func nonNilPorts(ports []int32) []int32 {
	if ports == nil {
		return []int32{}
	}
	return ports
}
