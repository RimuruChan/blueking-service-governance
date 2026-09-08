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

package app

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewDeleteCmd returns a Command instance for 'app delete' sub command
func NewDeleteCmd() *cobra.Command {
	var appID string
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an application",
		Long: `Delete an application and all its configurations permanently.

WARNING: This operation is irreversible. The application and all its configurations
(build config, deploy history, config files, etc.) will be permanently deleted.`,
		Example: `  # Delete an application
  bkms-cli app delete --app myapp

  # Delete without confirmation prompt
  bkms-cli app delete --app myapp --yes`,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := client.New().GetApp(cmd.Context(), appID)
			if err != nil {
				return errors.Wrap(err, "get app")
			}

			printAppDeleteConfirmInfo(app)

			if !yes {
				input, inputErr := cmdutil.PromptInput("Type the app name '" + app.Name + "' to confirm deletion: ")
				if inputErr != nil {
					return errors.Wrap(inputErr, "read confirmation")
				}
				if input != app.Name {
					return errors.New("deletion cancelled: input does not match app name")
				}
			}

			if err = client.New().DeleteApp(cmd.Context(), appID); err != nil {
				return errors.Wrap(err, "delete app")
			}

			console.Info("✓ App %s (%s) deleted successfully", app.Name, app.ID)
			return nil
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	_ = cmd.MarkFlagRequired("app")

	return cmd
}

func printAppDeleteConfirmInfo(app *client.AppFull) {
	console.Warn("⚠️  WARNING: This operation is IRREVERSIBLE.")
	console.Warn("   The application and ALL its configurations will be permanently deleted.")
	console.Info("")
	console.Info("  App ID:      %s", app.ID)
	console.Info("  App Name:    %s", app.Name)
	console.Info("  DisplayName: %s", app.DisplayName)
	console.Info("  Type:        %s", app.Type)
	console.Info("")
}
