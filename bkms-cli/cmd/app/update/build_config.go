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

package update

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	apphandler "github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/handler/app"
	cmdutil "github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewBuildConfigCmd returns a Command instance for 'app update build-config' sub command
func NewBuildConfigCmd() *cobra.Command {
	var appID, specFile string

	cmd := &cobra.Command{
		Use:   "build-config",
		Short: "Update application build configuration",
		Long: `Update the build configuration for an application from a YAML spec file.

The spec file must contain sourceType and the matching sub-config (codeRepo / image / pipeline).
Use 'app get --app myapp -o yaml' to view the current config as a reference.`,
		Example: `  # Update build config from a YAML file
  bkms-cli app update build-config --app myapp -f build_config.yaml

  # codeRepository with Dockerfile (build_config.yaml):
  #   sourceType: codeRepository
  #   tagConfig:
  #     type: custom
  #     customOpts:
  #       withBuildTime: true
  #   codeRepo:
  #     type: TGit
  #     repoAlias: myteam/myapp
  #     repoURL: https://git.woa.com/myteam/myapp.git
  #     defaultBranch: main
  #     imageBuildMode: repositoryDockerfile
  #     dockerfile: ./Dockerfile

  # codeRepository with platform build:
  #   sourceType: codeRepository
  #   codeRepo:
  #     type: TGit
  #     repoAlias: myteam/myapp
  #     repoURL: https://git.woa.com/myteam/myapp.git
  #     defaultBranch: main
  #     imageBuildMode: platform
  #     platformBuildConfig:
  #       builderImage: mirrors.tencent.com/bkms/golang:1.26.5-alpine3.24
  #       runnerImage: mirrors.tencent.com/bkms/alpine:3.24

  # imageRegistry:
  #   sourceType: imageRegistry
  #   image:
  #     name: mirrors.tencent.com/myteam/myapp

  # pipeline:
  #   sourceType: pipeline
  #   pipeline:
  #     pipelineID: p-abc123
  #     params:
  #       BKMS_IMAGE_TAG: latest`,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAppUpdateBuildConfig(cmd, appID, specFile)
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVarP(&specFile, "file", "f", "", "YAML spec file path (required)")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func runAppUpdateBuildConfig(cmd *cobra.Command, appID, specFile string) error {
	if err := apphandler.UpdateBuildConfig(cmd.Context(), appID, specFile); err != nil {
		return errors.Wrap(err, "update build config")
	}

	console.Info("✓ Build config updated for app %s", appID)
	return nil
}
