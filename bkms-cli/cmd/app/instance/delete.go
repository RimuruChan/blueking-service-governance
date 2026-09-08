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

// NewInstanceDeleteCmd returns a Command instance for 'app instance delete' sub command
func NewInstanceDeleteCmd() *cobra.Command {
	var appID, envName, instanceIDsStr string
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete (terminate) one or more running instances",
		Long: `Delete running Pod instances for an application in an environment.

WARNING: Deleting running pods also decrements the replica count permanently.
The fleet will have fewer instances than expected until you manually restore replicas.`,
		Example: `  # Delete specific instances
  bkms-cli app instance delete --app myapp --env prod --instance-ids pod1,pod2

  # Delete without confirmation prompt
  bkms-cli app instance delete --app myapp --env prod --instance-ids pod1,pod2 --yes`,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstanceDelete(cmd, appID, envName, instanceIDsStr, yes)
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVar(&envName, "env", "", "environment name (required)")
	cmd.Flags().StringVar(&instanceIDsStr, "instance-ids", "", "comma-separated instance IDs (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("env")
	_ = cmd.MarkFlagRequired("instance-ids")

	return cmd
}

func runInstanceDelete(cmd *cobra.Command, appID, envName, instanceIDsStr string, yes bool) error {
	instanceIDs := params.NormalizeInstIDs(instanceIDsStr, ",")

	cli := client.New()
	instances, err := instancehandler.ListInstances(
		cmd.Context(), cli, appID, envName, instancehandler.ListInstancesOptions{},
	)
	if err != nil {
		return errors.Wrap(err, "list instances")
	}

	printInstanceDeleteConfirmInfo(appID, envName, instanceIDs, instances)

	confirmed, confirmErr := cmdutil.PromptConfirm("Confirm deletion? (yes/no): ", yes)
	if confirmErr != nil {
		return errors.Wrap(confirmErr, "read confirmation")
	}
	if !confirmed {
		return errors.New("deletion cancelled")
	}

	if err = instancehandler.DeleteInstances(cmd.Context(), cli, appID, envName, instanceIDs); err != nil {
		return err
	}

	console.Info("✓ Deleted instances: %s", strings.Join(instanceIDs, ", "))
	console.Warn("WARNING: Deleting running pods also decrements the replica count.")
	console.Warn("Use 'app instance list' to verify, and 'app deploy create' to restore replicas if needed.")
	return nil
}

func printInstanceDeleteConfirmInfo(appID, envName string, instanceIDs []string, instances []client.Instance) {
	statusMap := make(map[string]string, len(instances))
	for _, inst := range instances {
		statusMap[inst.ID] = inst.Status
	}

	console.Warn("WARNING: Deleting running Pods PERMANENTLY decrements the replica count.")
	console.Warn("The fleet may have fewer instances than expected after this operation.")
	console.Info("")
	console.Info("  App: %s", appID)
	console.Info("  Env: %s", envName)
	console.Info("")
	console.Info("  Instances to delete:")
	for _, id := range instanceIDs {
		status := statusMap[id]
		if status == "" {
			status = "unknown"
		}
		console.Info("    %s (%s)", id, status)
	}
	console.Info("")
}
