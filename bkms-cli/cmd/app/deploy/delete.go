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

package deploy

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"
	deployhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/deploy"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewDeployDeleteCmd returns a Command instance for 'app deploy delete' sub command
func NewDeployDeleteCmd() *cobra.Command {
	var appID, envName, deployID string
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete (undeploy) an application from an environment",
		Long: `Remove the deployment of an application from an environment.

For helm applications, you must specify --deploy-id (from 'deploy list').
For trpc and taf applications, the entire environment deployment is removed.`,
		Example: `  # Delete trpc/taf deployment
  bkms-cli app deploy delete --app myapp --env test

  # Delete helm deployment (deploy-id required)
  bkms-cli app deploy delete --app myapp --env test --deploy-id deploy1

  # Delete without confirmation prompt
  bkms-cli app deploy delete --app myapp --env test --yes`,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := client.New().GetApp(cmd.Context(), appID)
			if err != nil {
				return errors.Wrap(err, "get app")
			}

			if app.Type == constant.AppTypeHelm && deployID == "" {
				return errors.New("--deploy-id is required for helm applications (see 'app deploy list')")
			}

			printDeployDeleteConfirmInfo(app, envName, deployID)

			confirmed, confirmErr := cmdutil.PromptConfirm("Confirm delete deployment? (yes/no): ", yes)
			if confirmErr != nil {
				return errors.Wrap(confirmErr, "read confirmation")
			}
			if !confirmed {
				return errors.New("deletion cancelled")
			}

			return deployhandler.DeleteDeploy(cmd.Context(), app.Type, appID, envName, deployID)
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVar(&envName, "env", "", "environment name (required)")
	cmd.Flags().StringVar(&deployID, "deploy-id", "", "deploy record ID (required for helm apps)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("env")

	return cmd
}

func printDeployDeleteConfirmInfo(app *client.AppFull, envName, deployID string) {
	console.Info("  App:  %s (%s)", app.Name, app.ID)
	console.Info("  Env:  %s", envName)
	console.Info("  Type: %s", app.Type)
	if deployID != "" {
		console.Info("  DeployID: %s", deployID)
	}
	console.Info("")
}
