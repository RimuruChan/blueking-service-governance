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

package env

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	envhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/env"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewDeleteCmd returns a Command instance for 'env delete' sub command
func NewDeleteCmd() *cobra.Command {
	var workspaceID, envName string
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an environment",
		Long: `Delete an environment permanently.

WARNING: This operation is irreversible. Ensure all deployments in this environment
have been removed before deleting.`,
		Example: `  # Delete an environment
  bkms-cli env delete --workspace ws1 --env staging

  # Delete without confirmation prompt
  bkms-cli env delete --workspace ws1 --env staging --yes`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			env, err := envhandler.ResolveEnvByName(cmd.Context(), client.New(), workspaceID, envName)
			if err != nil {
				return errors.Wrap(err, "resolve env")
			}

			printEnvDeleteConfirmInfo(env)

			confirmed, confirmErr := cmdutil.PromptConfirm("Confirm deletion? (yes/no): ", yes)
			if confirmErr != nil {
				return errors.Wrap(confirmErr, "read confirmation")
			}
			if !confirmed {
				return errors.New("deletion cancelled")
			}

			if err = client.New().DeleteEnv(cmd.Context(), env.ID); err != nil {
				return errors.Wrap(err, "delete env")
			}

			console.Info("✓ Environment %s (%s) deleted successfully", env.Name, env.ID)
			return nil
		},
	}

	cmdutil.AddWorkspaceFlag(cmd, &workspaceID)
	cmd.Flags().StringVar(&envName, "env", "", "environment name (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")

	_ = cmd.MarkFlagRequired("env")

	return cmd
}

func printEnvDeleteConfirmInfo(env *client.Env) {
	console.Warn("WARNING: This operation is IRREVERSIBLE.")
	console.Warn("Ensure all deployments in this environment have been removed first.")
	console.Info("")
	console.Info("  Environment: %s (%s)", env.Name, env.ID)
	console.Info("  Type:        %s", env.Type)
	if env.Cluster != nil {
		console.Info("  Cluster:     %s / %s", env.Cluster.ClusterID, env.Cluster.Namespace)
	}
	console.Info("")
}
