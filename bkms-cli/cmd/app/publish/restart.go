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

package publish

import (
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	publishhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/publish"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// newRestartCmd 创建 publish restart 子命令
func newRestartCmd() *cobra.Command {
	var appID, envs, envTypes, instances string

	cmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart business processes in dev mode containers (pods are not recreated)",
		Long: `Restart business processes in dev mode containers without recreating pods.

The current binary is not replaced. The request is accepted asynchronously: the
server queues the restart tasks and returns immediately. The CLI then keeps
querying the restart records of the same target condition until every instance
reaches a terminal state (success / failed), then prints the result and exits.

Dev mode must be enabled for the target environments, and production environments
are not allowed.

Targets (exactly one of --env / --env-type is required, multiple values are comma-separated):
  --env           one or more environments, e.g. stage,test; restarts all Running
                  instances in them
  --env-type      one or more environment types, e.g. development,test; restarts all
                  Running instances in matching environments; environments that are
                  not eligible (e.g. dev mode disabled) are skipped
  --instance-ids  one or more instances, e.g. pod1,pod2; only restarts the given
                  instances, requires a single --env

The restart keeps running on the server even if the CLI is interrupted or the wait
times out (3 minutes). Use 'bkms-cli app publish history' to check the final result.`,
		Example: `  # Restart all Running instances in multiple environments
  bkms-cli app publish restart --app myapp --env stage,test

  # Restart all Running instances in environments of the given types
  bkms-cli app publish restart --app myapp --env-type development,test

  # Restart specific instances in one environment
  bkms-cli app publish restart --app myapp --env stage --instance-ids pod1,pod2`,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := publishhandler.BuildRestartOptions(envs, envTypes, instances)
			if err != nil {
				return err
			}
			return publishhandler.Restart(cmd.Context(), client.New(), appID, opts)
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVar(&envs, "env", "", "Environment names, comma-separated")
	cmd.Flags().
		StringVar(&envTypes, "env-type", "", "Environment types, comma-separated: development | test | staging")
	cmd.Flags().
		StringVar(&instances, "instance-ids", "", "Instance IDs (pod names) to restart, comma-separated; requires a single --env")

	cmd.MarkFlagsMutuallyExclusive("env", "env-type")
	cmd.MarkFlagsOneRequired("env", "env-type")

	_ = cmd.MarkFlagRequired("app")

	return cmd
}
