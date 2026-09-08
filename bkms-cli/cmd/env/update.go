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

package env

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	envhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/env"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewUpdateCmd returns a Command instance for 'env update' sub command
func NewUpdateCmd() *cobra.Command {
	var workspaceID, envName, displayName, envType string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update environment basic info",
		Long: `Update the display name or type of an environment.

Note: name is immutable after creation and cannot be changed.
Valid types: development | test | staging | production`,
		Example: `  # Update display name
  bkms-cli env update --workspace ws1 --env staging --display-name "My Env"

  # Change environment type
  bkms-cli env update --workspace ws1 --env staging --type production`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			if displayName == "" && envType == "" {
				return errors.New("at least one of --display-name or --type must be specified")
			}

			body := client.UpdateEnvBasicInfoBody{
				DisplayName: displayName,
				Type:        envType,
			}
			return envhandler.UpdateEnv(cmd.Context(), workspaceID, envName, body)
		},
	}

	cmdutil.AddWorkspaceFlag(cmd, &workspaceID)
	cmd.Flags().StringVar(&envName, "env", "", "environment name (required)")
	cmd.Flags().StringVar(&displayName, "display-name", "", "new display name")
	cmd.Flags().StringVar(&envType, "type", "", "new environment type: development | test | staging | production")

	_ = cmd.MarkFlagRequired("env")

	return cmd
}
