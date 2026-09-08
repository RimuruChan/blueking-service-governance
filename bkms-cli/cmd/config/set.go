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

package config

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/config"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewCmdSet updates API endpoint settings in the local config file.
func NewCmdSet() *cobra.Command {
	var bkmsBaseURL string
	var ifUnset bool

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set bkms-cli API endpoints",
		Long: `Update the bkms service base URL in the local config file.

--bkms-base-url is required.
With --if-unset, the value is written only when it is currently empty.`,
		Example: `  bkms-cli config set --bkms-base-url https://bkms.example.com
  bkms-cli config set --if-unset --bkms-base-url https://bkms.example.com`,
		DisableFlagsInUseLine: true,
		Annotations: map[string]string{
			cmdutil.SkipAuthAnnotationKey: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if bkmsBaseURL == "" {
				return errors.New("--bkms-base-url is required")
			}

			updated, err := config.G.SetBkmsBaseURL(bkmsBaseURL, ifUnset)
			if err != nil {
				return err
			}
			if !updated {
				console.Info("config unchanged (--if-unset and value already set)")
				return nil
			}

			console.Info("config updated")
			console.Info("  bkmsBaseUrl: %s", config.G.BkmsBaseURL)
			return nil
		},
	}

	cmd.Flags().StringVar(&bkmsBaseURL, "bkms-base-url", "", "bkms service base URL")
	cmd.Flags().BoolVar(&ifUnset, "if-unset", false, "only set when currently empty")
	return cmd
}
