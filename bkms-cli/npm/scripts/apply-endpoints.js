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

// 这个文件负责「装完 npm 包之后，把默认 API 地址写进本地配置」。
//
// 典型用法：分发包在 postinstall 里调用本脚本。脚本会读当前正在安装的那个
// package.json（优先 npm 注入的 npm_package_json，否则读本包自己的）里的
// bkmsCli.bkmsBaseUrl。字段为空就跳过；有值则调用本机已下载好的
// bkms-cli，执行 `config set --if-unset --bkms-base-url ...`。
//
// 用 --if-unset 是为了：用户已经改过地址时，npm 重装/升级不要把它盖掉。

const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");

const NAME = "bkms-cli";

function packageJSONPath() {
  if (process.env.npm_package_json && fs.existsSync(process.env.npm_package_json)) {
    return process.env.npm_package_json;
  }
  return path.join(__dirname, "..", "package.json");
}

function binaryPath() {
  const ext = process.platform === "win32" ? ".exe" : "";
  return path.join(__dirname, "..", "bin", NAME + ext);
}

function applyEndpoints() {
  const pkg = JSON.parse(fs.readFileSync(packageJSONPath(), "utf8"));
  const endpoints = pkg.bkmsCli || {};
  const bkmsBaseUrl = String(endpoints.bkmsBaseUrl || "").trim();
  if (!bkmsBaseUrl) {
    return;
  }

  const bin = binaryPath();
  if (!fs.existsSync(bin)) {
    throw new Error(`bkms-cli binary not found at ${bin}; install the binary first`);
  }

  execFileSync(bin, ["config", "set", "--if-unset", "--bkms-base-url", bkmsBaseUrl], {
    stdio: "inherit",
  });
}

module.exports = { applyEndpoints, packageJSONPath, binaryPath };

if (require.main === module) {
  try {
    applyEndpoints();
  } catch (err) {
    console.error(`Failed to apply bkmsCli endpoints:`, err.message);
    process.exit(1);
  }
}
