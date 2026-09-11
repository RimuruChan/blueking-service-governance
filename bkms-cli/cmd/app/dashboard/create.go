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

package dashboard

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewCreateCmd returns a Command instance for 'app dashboard create' sub command
func NewCreateCmd() *cobra.Command {
	var appID, uid string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Bind a dashboard to an application",
		Long: `Bind an existing BlueKing Monitor dashboard to an application.

The dashboard uid must already exist in the workspace's BlueKing Monitor
project. The dashboard title is fetched automatically. Use 'workspace
dashboard list' to browse available dashboards.`,
		Example: `  # Bind a dashboard to an application
  bkms-cli app dashboard create --app my-app --uid <uid>`,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := client.New().CreateAppDashboard(cmd.Context(), appID, uid); err != nil {
				return errors.Wrap(err, "create app dashboard")
			}

			console.Info("✓ App dashboard bound successfully")
			console.Info("  UID: %s", uid)
			return nil
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVar(&uid, "uid", "", "dashboard uid")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("uid")

	return cmd
}
