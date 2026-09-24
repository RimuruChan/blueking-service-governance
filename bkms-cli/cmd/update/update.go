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

// Package update provides the bkms-cli self-update command.
package update

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/updater"
	cmdutil "github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/utils/console"
	"github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/version"
)

const (
	updateCheckTimeout = 15 * time.Second
	npmUpgradeCommand  = "npm i -g @blueking/bkms-cli@latest"
	goUpgradeCommand   = "go install github.com/RimuruChan/blueking-service-governance/bkms-cli@latest"
)

// NewCmd creates the self-update command.
func NewCmd() *cobra.Command {
	var checkOnly, force bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for and install bkms-cli updates.",
		Example: "  bkms-cli update --check\n" +
			"  bkms-cli update\n" +
			"  bkms-cli update --force",
		Args: cobra.NoArgs,
		Annotations: map[string]string{
			cmdutil.SkipAuthAnnotationKey: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd.Context(), checkOnly, force)
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "check for updates without installing")
	cmd.Flags().BoolVar(&force, "force", false, "allow replacing npm or Go builds from the configured update source")
	return cmd
}

func runUpdate(ctx context.Context, checkOnly, force bool) error {
	upgradeCommand := managedUpgradeCommand(updater.InstalledViaNPM(), version.BuildChannel)
	if upgradeCommand != "" && !checkOnly && !force {
		console.Info("Upgrade this installation with:")
		console.Info("  %s", upgradeCommand)
		console.Info("Or use --force to replace the binary from the configured update source.")
		return nil
	}
	if version.Version == "dev" {
		return fmt.Errorf("development build has no release version; install a release with: %s", goUpgradeCommand)
	}
	if checkOnly {
		return checkUpdate(ctx, upgradeCommand)
	}

	info, err := updater.Update(ctx)
	if err != nil {
		return err
	}
	if info.Available {
		console.Info("bkms-cli updated from %s to %s", info.CurrentVersion, info.LatestVersion)
	} else {
		console.Info("bkms-cli %s is up to date", info.CurrentVersion)
	}
	return nil
}

func checkUpdate(ctx context.Context, upgradeCommand string) error {
	ctx, cancel := context.WithTimeout(ctx, updateCheckTimeout)
	defer cancel()

	info, err := updater.Check(ctx)
	if err != nil {
		return err
	}
	if !info.Available {
		console.Info("bkms-cli %s is up to date", info.CurrentVersion)
		return nil
	}

	console.Info("bkms-cli %s is available (current: %s)", info.LatestVersion, info.CurrentVersion)
	if upgradeCommand != "" {
		console.Info("Upgrade with: %s", upgradeCommand)
	}
	return nil
}

func managedUpgradeCommand(viaNPM bool, channel string) string {
	switch {
	case viaNPM:
		return npmUpgradeCommand
	case channel == "release":
		return ""
	default:
		return goUpgradeCommand
	}
}
