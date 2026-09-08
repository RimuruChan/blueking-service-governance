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

package polaris

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewWeightCmd returns a Command instance for 'app polaris weight' sub command
func NewWeightCmd() *cobra.Command {
	var appID, configName, envName string
	var weight int32

	cmd := &cobra.Command{
		Use:   "weight",
		Short: "Set global Polaris weight for all instances in an environment",
		Long: `Update the global default weight for all instances registered in a Polaris
service configuration for a specific environment.

This affects all instances in the environment uniformly (unlike 'app instance polaris'
which targets specific Pod instances).

Weight range: 0-10000 (0 = drain all traffic, 100 = normal weight).`,
		Example: `  # Drain all traffic for an environment
  bkms-cli app polaris weight --app myapp --config myconfig --env test --weight 0

  # Restore normal weight
  bkms-cli app polaris weight --app myapp --config myconfig --env test --weight 100`,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			if weight < 0 || weight > 10000 {
				return errors.New("--weight must be in range 0-10000")
			}

			if err := client.New().UpdatePolarisConfigEnvWeight(cmd.Context(), appID, configName, envName, weight); err != nil {
				return errors.Wrap(err, "update polaris config env weight")
			}

			console.Info("✓ Polaris global weight updated to %d for config %s in env %s", weight, configName, envName)
			if weight == 0 {
				console.Tips("All traffic has been drained. Remember to restore the weight when done.")
			}
			return nil
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVar(&configName, "config", "", "Polaris config name (required)")
	cmd.Flags().StringVar(&envName, "env", "", "environment name (required)")
	cmd.Flags().Int32Var(&weight, "weight", 0, "global weight for all instances (0-10000, required)")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("config")
	_ = cmd.MarkFlagRequired("env")
	_ = cmd.MarkFlagRequired("weight")

	return cmd
}
