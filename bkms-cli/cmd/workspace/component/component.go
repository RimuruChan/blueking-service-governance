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

// Package component provides workspace component command group
package component

import "github.com/spf13/cobra"

// NewCmd returns a Command instance for 'workspace component' command group
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "component",
		Short: "Manage workspace component instances",
		Long: `List workspace-level component instances that applications can reference.

Workspace component instances are shared presets. Applications attach them
with 'bkms-cli app component create --ref <name>'.`,
		DisableFlagsInUseLine: true,
	}

	cmd.AddCommand(NewListCmd())

	return cmd
}
