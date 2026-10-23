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

// Package instance provides instance list command
package instance

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/instance"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewListCmd returns a Command instance for 'app instance list' sub command
func NewListCmd() *cobra.Command {
	var appID, envName, outputFormat string
	var podIPs, nodeIPs []string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List application instances",
		Long: `List running instances for an application in a specific environment.

This command retrieves all running instances for the specified application
and environment, including instance ID, IP, image, status, and age.

Use --pod-ip / --node-ip to filter instances by exact IP match. Both flags can be
repeated or comma-separated, and all given values are ORed together, across the
two flags as well.`,
		Example: `  # List all instances for an application
  bkms-cli app instance list --app demo --env test

  # Output in JSON format
  bkms-cli app instance list --app demo --env test -o json

  # Filter instances by pod IP
  bkms-cli app instance list --app demo --env test --pod-ip 10.0.0.1 --pod-ip 10.0.0.2

  # Filter instances running on given nodes
  bkms-cli app instance list --app demo --env test --node-ip 192.168.0.1,192.168.0.2`,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			instances, err := handler.ListInstances(
				cmd.Context(),
				client.New(),
				appID,
				envName,
				handler.ListInstancesOptions{PodIPs: podIPs, NodeIPs: nodeIPs},
			)
			if err != nil {
				return errors.Wrap(err, "list app instances")
			}

			formatted, err := output.FormatData(cmd.Context(), instances, outputFormat)
			if err != nil {
				return errors.Wrap(err, "format output")
			}
			console.Info("%s", formatted)
			return nil
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVar(&envName, "env", "", "environment name")
	cmd.Flags().StringSliceVar(&podIPs, "pod-ip", nil, "filter by pod IP (exact match, repeatable)")
	cmd.Flags().StringSliceVar(&nodeIPs, "node-ip", nil, "filter by node IP (exact match, repeatable)")
	output.AddFormatFlag(cmd, &outputFormat)

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("env")

	return cmd
}
