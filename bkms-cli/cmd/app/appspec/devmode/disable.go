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

package devmode

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/handler/appspec"
	cmdutil "github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewDisableCmd returns a Command instance for 'appspec dev-mode disable' sub command.
func NewDisableCmd() *cobra.Command {
	var appID, envName string

	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable dev-mode for an environment",
		Long: `Disable the development mode for an application environment.

After disabling, the application will need to be redeployed to take effect.`,
		Example: `  # Disable dev-mode for an environment
  bkms-cli app appspec dev-mode disable --app my-app --env prod`,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := appspec.SetEnabledHandler(
				cmd.Context(), appID, envName, client.AppSpecSectionDevMode, false,
			); err != nil {
				return errors.Wrap(err, "disable dev-mode")
			}

			fmt.Printf("Successfully disabled dev-mode for app %s in env %s\n", appID, envName)
			return nil
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVar(&envName, "env", "", "environment name (required)")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("env")

	return cmd
}
