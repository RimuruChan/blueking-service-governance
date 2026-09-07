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

package cmd

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/clierr"
)

// Execute 统一输出命令错误并返回退出码；Cobra 不再重复打印错误和 usage。
func Execute(ctx context.Context, root *cobra.Command, args []string) int {
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return clierr.Usage(err) })
	markArgumentErrors(root)
	root.SetArgs(args)
	command, err := root.ExecuteContextC(ctx)
	if err == nil {
		return 0
	}

	var reported *clierr.ReportedError
	if errors.As(err, &reported) {
		return 1
	}
	if errors.Is(err, clierr.ErrCancelled) || errors.Is(err, context.Canceled) {
		command.PrintErrln(err)
		return 1
	}

	command.PrintErrln("Error:", err)
	var usage *clierr.UsageError
	_, _, findErr := root.Find(args)
	if errors.As(err, &usage) || findErr != nil ||
		command.ValidateRequiredFlags() != nil || command.ValidateFlagGroups() != nil {
		command.PrintErrln(command.UsageString())
	}
	return 1
}

// 将 Cobra 的位置参数校验错误标记为用法错误，不影响运行和 PreRun 错误。
func markArgumentErrors(command *cobra.Command) {
	if validate := command.Args; validate != nil {
		command.Args = func(cmd *cobra.Command, args []string) error {
			if err := validate(cmd, args); err != nil {
				return clierr.Usage(err)
			}
			return nil
		}
	}
	for _, child := range command.Commands() {
		markArgumentErrors(child)
	}
}
