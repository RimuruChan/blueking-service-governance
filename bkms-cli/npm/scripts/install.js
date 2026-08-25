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

// 这个文件是 npm 的 postinstall 脚本：负责把对应平台的 bkms-cli 二进制下载到本机。
//
// 流程大致是：看当前系统是 mac/linux/windows、cpu 是 amd64 还是 arm64，再结合
// package.json 的 version，拼出要下的压缩包名字；下载地址来自 bkmsCli.releaseUrl
//（里面的 {version}、{archive} 会被替换）。下完后先对照同目录的 checksums.txt
// 做 SHA-256 校验，再解压，把二进制拷到 npm 包的 bin/ 目录，并 chmod 成可执行。
// 最后再调一下 apply-endpoints.js，如果 package.json 里配了默认地址就写进本地配置。
//
// 下载依赖本机 curl；解压用 tar / Expand-Archive。失败时会提示检查代理 / 网络。

const crypto = require("crypto");
const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");
const os = require("os");

const pkg = require("../package.json");
const VERSION = pkg.version;
const NAME = "bkms-cli";
const CHECKSUMS_NAME = "checksums.txt";

const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const ARCH_MAP = {
  x64: "amd64",
  arm64: "arm64",
};

const platform = PLATFORM_MAP[process.platform];
const arch = ARCH_MAP[process.arch];

if (!platform || !arch) {
  console.error(
    `Unsupported platform: ${process.platform}-${process.arch}`
  );
  process.exit(1);
}

const isWindows = process.platform === "win32";
const ext = isWindows ? ".zip" : ".tar.gz";
// Git tag is bkms-cli/vX.Y.Z; GoReleaser archive names use VERSION without "v".
const archiveName = `${NAME}_${VERSION}_${platform}_${arch}${ext}`;

function resolveReleaseURL(assetName) {
  const template = String((pkg.bkmsCli && pkg.bkmsCli.releaseUrl) || "").trim();
  if (!template) {
    throw new Error("bkmsCli.releaseUrl is required in package.json");
  }
  return template
    .replaceAll("{version}", VERSION)
    .replaceAll("{archive}", assetName);
}

const binDir = path.join(__dirname, "..", "bin");
const dest = path.join(binDir, NAME + (isWindows ? ".exe" : ""));

fs.mkdirSync(binDir, { recursive: true });

function download(url, destPath) {
  // --ssl-revoke-best-effort: on Windows (Schannel), avoid CRYPT_E_REVOCATION_OFFLINE
  // errors when the certificate revocation list server is unreachable
  const args = [
    "--fail",
    "--location",
    "--silent",
    "--show-error",
    "--connect-timeout",
    "10",
    "--max-time",
    "120",
    "--output",
    destPath,
    url,
  ];
  if (isWindows) {
    args.unshift("--ssl-revoke-best-effort");
  }
  execFileSync("curl", args, { stdio: ["ignore", "ignore", "pipe"] });
}

function expectedSHA256(checksumsText, filename) {
  for (const line of checksumsText.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) {
      continue;
    }
    const parts = trimmed.split(/\s+/);
    if (parts.length < 2) {
      continue;
    }
    const hash = parts[0].toLowerCase();
    const name = parts[parts.length - 1].replace(/^\*/, "");
    if (name === filename) {
      return hash;
    }
  }
  throw new Error(`checksum not found for ${filename} in ${CHECKSUMS_NAME}`);
}

function verifySHA256(filePath, expected) {
  const actual = crypto
    .createHash("sha256")
    .update(fs.readFileSync(filePath))
    .digest("hex");
  if (actual !== expected.toLowerCase()) {
    throw new Error(
      `checksum mismatch for ${path.basename(filePath)}: expected ${expected}, got ${actual}`
    );
  }
}

function extractArchive(archivePath, tmpDir) {
  if (isWindows) {
    execFileSync(
      "powershell.exe",
      [
        "-NoProfile",
        "-NonInteractive",
        "-Command",
        "Expand-Archive -LiteralPath $env:BKMS_CLI_ARCHIVE -DestinationPath $env:BKMS_CLI_DEST -Force",
      ],
      {
        env: {
          ...process.env,
          BKMS_CLI_ARCHIVE: archivePath,
          BKMS_CLI_DEST: tmpDir,
        },
        stdio: "ignore",
      }
    );
    return;
  }
  execFileSync("tar", ["-xzf", archivePath, "-C", tmpDir], { stdio: "ignore" });
}

function install() {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "bkms-cli-"));
  const archivePath = path.join(tmpDir, archiveName);
  const checksumsPath = path.join(tmpDir, CHECKSUMS_NAME);

  try {
    download(resolveReleaseURL(archiveName), archivePath);
    download(resolveReleaseURL(CHECKSUMS_NAME), checksumsPath);
    verifySHA256(
      archivePath,
      expectedSHA256(fs.readFileSync(checksumsPath, "utf8"), archiveName)
    );

    extractArchive(archivePath, tmpDir);

    const binaryName = NAME + (isWindows ? ".exe" : "");
    const extractedBinary = path.join(tmpDir, binaryName);

    fs.copyFileSync(extractedBinary, dest);
    fs.chmodSync(dest, 0o755);
    console.log(`${NAME} v${VERSION} installed successfully`);

    const { applyEndpoints } = require("./apply-endpoints");
    applyEndpoints();
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }
}

try {
  install();
} catch (err) {
  console.error(`Failed to install ${NAME}:`, err.message);
  console.error(
    `\nIf you are behind a firewall or in a restricted network, try setting a proxy:\n` +
      `  export https_proxy=http://your-proxy:port\n` +
      `  npm i -g @blueking/bkms-cli`
  );
  process.exit(1);
}
