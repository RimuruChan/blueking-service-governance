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

// Package e2escript_test 提供基于 testscript 的 bkms-cli 端到端黑盒测试。
// 目录结构、脚本写法与运行方式见 README.md。
package e2escript_test

import (
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

// serialT 包装 *testing.T，把 Parallel 变为空实现。
// testscript 对每个脚本无条件调用 t.Parallel()，但脚本间共享真实后端资源，
// 必须串行。RunT 接受 T 接口而非 *testing.T，故可在此屏蔽。
type serialT struct {
	*testing.T
}

func (serialT) Parallel() {}

// Run 必须一并重写，否则子测试拿到裸 *testing.T，Parallel 会重新生效。
func (t serialT) Run(name string, f func(testscript.T)) {
	t.T.Run(name, func(sub *testing.T) { f(serialT{sub}) })
}

func (serialT) Verbose() bool { return testing.Verbose() }

// runDomain 串行运行 testdata/<domain>/ 下的所有脚本。
func runDomain(t *testing.T, domain string, setupFn func(*testscript.Env) error) {
	t.Helper()

	p := testscript.Params{
		Dir:   filepath.Join("testdata", domain),
		Setup: setupFn,
		Cmds:  customCmds(),
	}
	// deadline 传递是 testscript.Run 的行为，RunT 不做，故在此补上
	if deadline, ok := t.Deadline(); ok {
		p.Deadline = deadline
	}
	testscript.RunT(serialT{t}, p)
}

func TestApp(t *testing.T)       { runDomain(t, "app", setup) }
func TestAppspec(t *testing.T)   { runDomain(t, "appspec", setup) }
func TestBase(t *testing.T)      { runDomain(t, "base", setup) }
func TestDeploy(t *testing.T)    { runDomain(t, "deploy", setup) }
func TestEnvvar(t *testing.T)    { runDomain(t, "envvar", setup) }
func TestExtension(t *testing.T) { runDomain(t, "extension", setup) }
func TestUnauth(t *testing.T)    { runDomain(t, "unauth", setupUnauth) }
