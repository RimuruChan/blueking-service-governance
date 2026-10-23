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

package e2escript_test

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rogpeppe/go-internal/testscript"
)

// customCmds 返回脚本中可用的自定义命令映射。
// 各命令均读取上一条 exec 的输出；加 ! 前缀为反向断言。
func customCmds() map[string]func(ts *testscript.TestScript, neg bool, args []string) {
	return map[string]func(ts *testscript.TestScript, neg bool, args []string){
		"jsonhas":   cmdJSONHas,
		"jsonfield": cmdJSONField,
	}
}

// cmdJSONHas 断言 stdout 是 JSON 对象且含指定顶层键：jsonhas <key>...
func cmdJSONHas(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) == 0 {
		ts.Fatalf("jsonhas: at least one key required")
	}

	raw := ts.ReadFile("stdout")
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		ts.Fatalf("jsonhas: stdout is not a JSON object: %v\nstdout: %s", err, raw)
	}

	for _, key := range args {
		_, has := m[key]
		if neg && has {
			ts.Fatalf("jsonhas: key %q unexpectedly present in JSON output", key)
		}
		if !neg && !has {
			ts.Fatalf("jsonhas: key %q not found in JSON output\nstdout: %s", key, raw)
		}
	}
}

// cmdJSONField 断言 stdout 中指定字段的值与期望相符：jsonfield <key>=<expected>
func cmdJSONField(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 1 {
		ts.Fatalf("jsonfield: exactly one key=value argument required")
	}

	parts := strings.SplitN(args[0], "=", 2)
	if len(parts) != 2 {
		ts.Fatalf("jsonfield: argument must be key=value, got %q", args[0])
	}
	key, expected := parts[0], parts[1]

	raw := ts.ReadFile("stdout")
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		ts.Fatalf("jsonfield: stdout is not a JSON object: %v\nstdout: %s", err, raw)
	}

	val, has := m[key]
	if !has {
		ts.Fatalf("jsonfield: key %q not found in JSON output\nstdout: %s", key, raw)
	}

	actual := fmt.Sprintf("%v", val)
	match := actual == expected
	if neg && match {
		ts.Fatalf("jsonfield: field %q unexpectedly equals %q", key, expected)
	}
	if !neg && !match {
		ts.Fatalf("jsonfield: field %q: got %q, want %q\nstdout: %s", key, actual, expected, raw)
	}
}
