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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewListCmd returns a Command instance for 'app component list' sub command
func NewListCmd() *cobra.Command {
	var appID, kind, outputFormat string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List application component instances",
		Long: `List component instances attached to an application.

kind=ref is a reference to a workspace component instance.
kind=inst is a component instance created on the application.
Use --kind to filter. Only trpc and taf apps are supported.`,
		Example: `  # List all component instances for an application
  bkms-cli app component list --app my-app

  # List only workspace component references
  bkms-cli app component list --app my-app --kind ref

  # Output in JSON format
  bkms-cli app component list --app my-app -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			comps, err := handler.ListAppComponents(cmd.Context(), client.New(), appID, kind)
			if err != nil {
				return errors.Wrap(err, "list app components")
			}

			formatted, err := output.FormatData(cmd.Context(), comps, outputFormat)
			if err != nil {
				return errors.Wrap(err, "format output")
			}
			console.Info("%s", formatted)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID")
	cmd.Flags().StringVar(&kind, "kind", "", "filter by kind: ref | inst")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("app")

	return cmd
}
