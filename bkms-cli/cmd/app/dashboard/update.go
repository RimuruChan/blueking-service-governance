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

// NewUpdateCmd returns a Command instance for 'app dashboard update' sub command
func NewUpdateCmd() *cobra.Command {
	var appID, uid, newUID string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Re-point an application dashboard binding",
		Long: `Re-point a dashboard binding to another dashboard.

--uid identifies the binding to update. --new-uid is the uid of the dashboard
to bind instead.

Use 'workspace dashboard list' to confirm the new uid exists before re-pointing.`,
		Example: `  # Re-point the binding to another dashboard
  bkms-cli app dashboard update --app my-app --uid <uid> --new-uid <uid>`,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := client.UpdateAppDashboardOptions{NewUID: newUID}
			if err := client.New().UpdateAppDashboard(cmd.Context(), appID, uid, opts); err != nil {
				return errors.Wrap(err, "update app dashboard")
			}

			console.Info("✓ App dashboard updated successfully")
			return nil
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVar(&uid, "uid", "", "dashboard uid to update")
	cmd.Flags().StringVar(&newUID, "new-uid", "", "new dashboard uid")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("uid")
	_ = cmd.MarkFlagRequired("new-uid")

	return cmd
}
