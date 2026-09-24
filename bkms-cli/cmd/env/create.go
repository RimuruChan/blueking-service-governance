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
	"github.com/spf13/cobra"

	envhandler "github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/handler/env"
	cmdutil "github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewCreateCmd returns a Command instance for 'env create' sub command
func NewCreateCmd() *cobra.Command {
	var workspaceID, specFile string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new environment",
		Long: `Create a new environment in a workspace from a YAML spec file.

Note: name and type are immutable after creation.
Valid types: development | test | staging | production`,
		Example: `  # Create from YAML spec file
  bkms-cli env create --workspace ws1 -f env.yaml

  # Example env.yaml:
  #   name: staging
  #   displayName: Staging Env
  #   type: staging
  #   description: Staging environment
  #   cluster:
  #     clusterID: BCS-K8S-12345
  #     clusterType: shared
  #     namespace: bkms-staging`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := envhandler.CreateEnv(cmd.Context(), cmdutil.GetWorkspaceID(workspaceID), specFile)
			if err != nil {
				return err
			}
			console.Info("✓ Environment created successfully")
			console.Info("  ID:          %s", env.ID)
			console.Info("  Name:        %s", env.Name)
			console.Info("  DisplayName: %s", env.DisplayName)
			console.Info("  Type:        %s", env.Type)
			if env.Cluster != nil {
				console.Info("  Cluster:     %s / %s", env.Cluster.ClusterID, env.Cluster.Namespace)
			}
			return nil
		},
	}

	cmdutil.AddWorkspaceFlag(cmd, &workspaceID)
	cmd.Flags().StringVarP(&specFile, "file", "f", "", "YAML spec file path (required)")

	_ = cmd.MarkFlagRequired("file")

	return cmd
}
