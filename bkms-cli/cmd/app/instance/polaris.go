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

package instance

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	instancehandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/instance"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/params"
)

// NewPolarisCmd returns a Command instance for 'app instance polaris' sub command
func NewPolarisCmd() *cobra.Command {
	var appID, envName, instanceIDsStr string
	var weight int
	var isolate bool
	var weightSet, isolateSet bool

	cmd := &cobra.Command{
		Use:   "polaris",
		Short: "Update Polaris (service registry) weight or isolation for instances",
		Long: `Adjust the Polaris traffic weight or isolation status for one or more instances.

Use --weight to control traffic share (0 = drain traffic, 100 = normal).
Use --isolate to set isolation status (true = isolate, false = restore).
At least one of --weight or --isolate must be specified.`,
		Example: `  # Drain traffic from an instance (before maintenance)
  bkms-cli app instance polaris --app myapp --env prod --instance-ids pod1 --weight 0

  # Restore traffic weight
  bkms-cli app instance polaris --app myapp --env prod --instance-ids pod1 --weight 100`,
		PreRunE: func(c *cobra.Command, a []string) error {
			if err := cmdutil.ResolveAppPreRunE(c, a); err != nil {
				return err
			}
			weightSet = c.Flags().Changed("weight")
			isolateSet = c.Flags().Changed("isolate")
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstancePolaris(cmd, appID, envName, instanceIDsStr, weight, isolate, weightSet, isolateSet)
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVar(&envName, "env", "", "environment name (required)")
	cmd.Flags().StringVar(&instanceIDsStr, "instance-ids", "", "comma-separated instance IDs (required)")
	cmd.Flags().IntVar(&weight, "weight", 0, "target Polaris traffic weight (0=drain, 100=normal)")
	cmd.Flags().BoolVar(&isolate, "isolate", false, "Polaris isolation status")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("env")
	_ = cmd.MarkFlagRequired("instance-ids")

	return cmd
}

func runInstancePolaris(
	cmd *cobra.Command,
	appID, envName, instanceIDsStr string,
	weight int,
	isolate, weightSet, isolateSet bool,
) error {
	if !weightSet && !isolateSet {
		return errors.New("at least one of --weight or --isolate must be specified")
	}

	instanceIDs := params.NormalizeInstIDs(instanceIDsStr, ",")

	opts := buildPolarisOpts(instanceIDs, weight, isolate, weightSet, isolateSet)

	if err := instancehandler.UpdatePolaris(cmd.Context(), client.New(), appID, envName, opts); err != nil {
		return err
	}

	printPolarisResult(instanceIDs, weight, isolate, weightSet, isolateSet)
	return nil
}

func buildPolarisOpts(
	instanceIDs []string,
	weight int,
	isolate, weightSet, isolateSet bool,
) client.UpdateInstancePolarisOptions {
	opts := client.UpdateInstancePolarisOptions{InstanceIDs: instanceIDs}
	if weightSet {
		opts.Weight = &weight
	}
	if isolateSet {
		opts.Isolate = &isolate
	}
	return opts
}

func printPolarisResult(instanceIDs []string, weight int, isolate, weightSet, isolateSet bool) {
	if weightSet {
		console.Info("✓ Updated polaris weight to %d for instances: %s",
			weight, strings.Join(instanceIDs, ", "))
		if weight == 0 {
			console.Tips("Remember to restore the weight when maintenance is complete.")
		}
	}
	if isolateSet {
		console.Info("✓ Updated polaris isolate to %v for instances: %s",
			isolate, strings.Join(instanceIDs, ", "))
	}
}
