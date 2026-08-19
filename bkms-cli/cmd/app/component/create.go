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

package component

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewCreateCmd returns a Command instance for 'app component create' sub command
func NewCreateCmd() *cobra.Command {
	var appID, refName, compName string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Reference a workspace component instance",
		Long: `Attach a workspace component instance to an application by name.

The application does not copy properties; deploy uses the workspace component
instance's values. Referenced instances cannot be edited. Only trpc and taf
apps are supported.

--name is optional. If omitted, the server generates a name.

The change takes effect after the next deployment.`,
		Example: `  # Reference a workspace component instance
  bkms-cli app component create --app my-app --ref shared-limits

  # Reference with an explicit application-local name
  bkms-cli app component create --app my-app --ref shared-limits --name my-limits`,
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := handler.CreateAppComponentRef(cmd.Context(), client.New(), appID, refName, compName)
			if err != nil {
				return errors.Wrap(err, "create app component reference")
			}

			console.Info("✓ App component referenced successfully")
			console.Info("  Name: %s", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID")
	cmd.Flags().StringVar(&refName, "ref", "", "workspace component instance name")
	cmd.Flags().StringVar(&compName, "name", "", "application-local component instance name")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("ref")

	return cmd
}
