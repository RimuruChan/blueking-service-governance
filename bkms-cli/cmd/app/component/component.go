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

// Package component provides app component command group
package component

import "github.com/spf13/cobra"

// NewCmd returns a Command instance for 'app component' command group
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "component",
		Short: "Manage application component instances",
		Long: `Manage application component instances, including references to
workspace component instances.

Use this command to list application component instances, reference a
workspace component instance, or remove an application component instance
by name.

Referencing a workspace component instance does not copy its properties. The
app uses the workspace instance's values at deploy time. Only trpc and taf
apps are supported.`,
		DisableFlagsInUseLine: true,
	}

	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewCreateCmd())
	cmd.AddCommand(NewDeleteCmd())

	return cmd
}
