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

package cmd_test

import (
	"bytes"
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/clierr"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

var _ = Describe("Execute", func() {
	var root, command *cobra.Command
	var stdout, stderr bytes.Buffer
	var runErr, preRunErr error
	BeforeEach(func() {
		stdout.Reset()
		stderr.Reset()
		runErr, preRunErr = nil, nil
		root = &cobra.Command{Use: "test-cli"}
		command = &cobra.Command{
			Use:     "check",
			Args:    cobra.NoArgs,
			PreRunE: func(*cobra.Command, []string) error { return preRunErr },
			RunE: func(c *cobra.Command, _ []string) error {
				var reported *clierr.ReportedError
				if errors.As(runErr, &reported) {
					c.Println("failure details")
				}
				return runErr
			},
		}
		command.Flags().String("env", "", "environment")
		Expect(command.MarkFlagRequired("env")).To(Succeed())
		command.Flags().Bool("first", false, "")
		command.Flags().Bool("second", false, "")
		command.MarkFlagsMutuallyExclusive("first", "second")
		root.AddCommand(command)
		root.SetOut(&stdout)
		root.SetErr(&stderr)
	})

	DescribeTable(
		"prints usage only for argument errors",
		func(args []string, failure error, wantUsage bool) {
			runErr = failure
			Expect(cmdutil.Execute(context.Background(), root, args)).To(Equal(1))
			Expect(strings.Count(stderr.String(), "Error:")).To(Equal(1))
			Expect(strings.Contains(stderr.String(), "Usage:")).To(Equal(wantUsage))
			Expect(stdout.String()).To(BeEmpty())
		},
		Entry("unknown flag", []string{"check", "--unknown"}, nil, true),
		Entry("invalid flag value", []string{"check", "--first=invalid"}, nil, true),
		Entry("missing required flag", []string{"check"}, nil, true),
		Entry("conflicting flags", []string{"check", "--env", "test", "--first", "--second"}, nil, true),
		Entry("unexpected positional argument", []string{"check", "--env", "test", "extra"}, nil, true),
		Entry("unknown command", []string{"unknown"}, nil, true),
		Entry(
			"custom argument error",
			[]string{"check", "--env", "test"},
			errors.Wrap(clierr.Usagef("invalid options"), "configure"),
			true,
		),
		Entry("API failure", []string{"check", "--env", "test"}, errors.New("environment tes not found"), false),
	)

	It("prints formatted usage errors with usage", func() {
		runErr = clierr.Usagef("unsupported mode: %s", "test")
		Expect(cmdutil.Execute(context.Background(), root, []string{"check", "--env", "test"})).To(Equal(1))
		Expect(stderr.String()).To(HavePrefix("Error: unsupported mode: test\n"))
		Expect(stderr.String()).To(ContainSubstring("Usage:"))
	})

	It("prints authentication errors once without usage", func() {
		preRunErr = errors.New("authentication failed")
		Expect(cmdutil.Execute(context.Background(), root, []string{"check", "--env", "test"})).To(Equal(1))
		Expect(stderr.String()).To(Equal("Error: authentication failed\n"))
	})

	It("does not repeat reported failures even through wrapping", func() {
		runErr = errors.Wrap(clierr.Reported(errors.New("check failed")), "command")
		Expect(cmdutil.Execute(context.Background(), root, []string{"check", "--env", "test"})).To(Equal(1))
		Expect(stdout.String()).To(Equal("failure details\n"))
		Expect(stderr.String()).To(BeEmpty())
	})

	It("reports cancellation without an error prefix or usage", func() {
		runErr = errors.Wrap(clierr.ErrCancelled, "delete")
		Expect(cmdutil.Execute(context.Background(), root, []string{"check", "--env", "test"})).To(Equal(1))
		Expect(stderr.String()).To(Equal("delete: operation cancelled\n"))
	})

	It("returns success without extra output", func() {
		Expect(cmdutil.Execute(context.Background(), root, []string{"check", "--env", "test"})).To(BeZero())
		Expect(stderr.String()).To(BeEmpty())
		Expect(stdout.String()).To(BeEmpty())
	})

	It("preserves explicit help", func() {
		Expect(cmdutil.Execute(context.Background(), root, []string{"check", "--help"})).To(BeZero())
		Expect(stdout.String()).To(ContainSubstring("Usage:"))
		Expect(stderr.String()).To(BeEmpty())
	})
})
