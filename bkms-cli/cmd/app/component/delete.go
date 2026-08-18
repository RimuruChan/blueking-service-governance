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

// NewDeleteCmd returns a Command instance for 'app component delete' sub command
func NewDeleteCmd() *cobra.Command {
	var appID, compName string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Remove an application component",
		Long: `Remove a component from an application by its app-local name.

--name is the component name returned by list, not the workspace component
name unless they happen to be the same. This only removes the app-side
attachment; the workspace component itself is not deleted.

After deleting a reference, trigger a deployment for the change to take
effect. Only trpc and taf apps are supported.`,
		Example: `  # Remove a referenced component from an application
  bkms-cli app component delete --app my-app --name my-limits`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := handler.DeleteAppComponent(cmd.Context(), client.New(), appID, compName); err != nil {
				return errors.Wrap(err, "delete app component")
			}

			console.Info("✓ App component deleted successfully")
			console.Info("  Name: %s", compName)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID")
	cmd.Flags().StringVar(&compName, "name", "", "app-local component name from list")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}
