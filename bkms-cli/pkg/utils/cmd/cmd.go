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

// Package cmd provides some helper functions for cobra commands.
package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/config"
	apphandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/app"
)

// SkipAuthAnnotationKey 允许在 cmd 注解中设置为 "true" 以跳过认证
const SkipAuthAnnotationKey = "skip-auth"

// IsAuthRequired 判断当前命令是否需要认证
func IsAuthRequired(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	// 特定的命令不需要认证
	if cmd.Name() == "bkms-cli" || cmd.Name() == "help" {
		return false
	}
	// 通过注解跳过认证的
	if cmd.Annotations != nil && cmd.Annotations[SkipAuthAnnotationKey] == "true" {
		return false
	}
	// 默认都要鉴权
	return true
}

// CommonPreRun 通用 PreRun
// 当配置无默认 workspace 时将 --workspace 标记为必填。
func CommonPreRun(cmd *cobra.Command, _ []string) {
	requireWorkspace(cmd)
}

// GetWorkspaceID 返回 flag 值，为空时回退到配置默认值。
func GetWorkspaceID(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return config.G.Defaults.WorkspaceID
}

// ResolveAppPreRunE workspace 必填检查 + 将 --app 的 ID/name 解析为确定的 app ID。
// 通过 cmd.Flags().Set 回写，StringVar 绑定的变量自动更新； 无 app 参数时仅做 workspace 检查。
func ResolveAppPreRunE(cmd *cobra.Command, _ []string) error {
	requireWorkspace(cmd)
	appFlag := cmd.Flags().Lookup("app")
	if appFlag == nil || appFlag.Value.String() == "" {
		return nil
	}
	wsID := ""
	if wsFlag := cmd.Flags().Lookup("workspace"); wsFlag != nil {
		wsID = wsFlag.Value.String()
	}
	wsID = GetWorkspaceID(wsID)

	resolved, err := apphandler.ResolveAppID(cmd.Context(), wsID, appFlag.Value.String())
	if err != nil {
		return errors.Wrap(err, "resolve app")
	}

	return cmd.Flags().Set("app", resolved)
}

// requireWorkspace 当配置中未设置默认 WorkspaceID 时，将 --workspace flag 标记为必填。
func requireWorkspace(cmd *cobra.Command) {
	if cmd.Flags().Lookup("workspace") == nil {
		return
	}
	if config.G.Defaults.WorkspaceID == "" {
		_ = cmd.MarkFlagRequired("workspace")
	}
}

// AddAppFlags 注册 --app 和 --workspace flag
// workspaceID 仅供 ResolveAppPreRunE 使用，不绑定变量。
func AddAppFlags(cmd *cobra.Command, appID *string) {
	cmd.Flags().StringVar(appID, "app", "", "application ID or name")
	cmd.Flags().String("workspace", "", "workspace ID")
}

// AddWorkspaceFlag 注册 --workspace flag 并绑定变量（供 RunE 中直接使用 workspaceID 的场景）。
func AddWorkspaceFlag(cmd *cobra.Command, workspaceID *string) {
	cmd.Flags().StringVar(workspaceID, "workspace", "", "workspace ID")
}
