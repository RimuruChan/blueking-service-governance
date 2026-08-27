# @blueking/bkms-cli

npm 代理包：`postinstall` 用 `curl` 从 GitHub Releases 下载对应平台的 `bkms-cli` 二进制。

## 安装

```bash
npm i -g @blueking/bkms-cli
```

要求：Node.js >= 18、本机有 `curl`，以及解压工具（macOS/Linux：`tar`；Windows：PowerShell）。

## 升级

```bash
npm i -g @blueking/bkms-cli@latest
```

若通过本包安装，默认请用 npm 升级，不要直接 `bkms-cli update`（命令会提示改走 npm）。需要强制从 GitHub Releases 替换二进制时可用 `bkms-cli update --force`。

手动设置 API 地址：

```bash
bkms-cli config set --bkms-base-url https://bkms.example.com
```

## `bkmsCli` 配置

本包 `package.json`：

```json
"bkmsCli": {
  "bkmsBaseUrl": "",
  "releaseUrl": "https://github.com/TencentBlueKing/blueking-service-governance/releases/download/bkms-cli%2Fv{version}/{archive}"
}
```

- `releaseUrl`：二进制下载地址模板，支持 `{version}`、`{archive}`
- `bkmsBaseUrl`：默认为空；非空时 `postinstall` 会 `config set --if-unset`

分发包可依赖本包并只覆盖 endpoint，例如：

```json
{
  "name": "@example/bkms-cli",
  "version": "1.0.0",
  "dependencies": {
    "@blueking/bkms-cli": "1.0.0"
  },
  "bin": {
    "bkms-cli": "bin/bkms-cli.js"
  },
  "scripts": {
    "postinstall": "node node_modules/@blueking/bkms-cli/scripts/apply-endpoints.js"
  },
  "bkmsCli": {
    "bkmsBaseUrl": "https://bkms.example.com"
  }
}
```

`bin/bkms-cli.js`：

```js
#!/usr/bin/env node
require("@blueking/bkms-cli/scripts/bkms-cli.js");
```

说明：

- 安装分发包时会先安装 `@blueking/bkms-cli`（按本包 `releaseUrl` 下载二进制）
- 分发包 `postinstall` 调用 `apply-endpoints.js`；该脚本通过 `npm_package_json` 读取分发包的 `bkmsCli`，再执行 `bkms-cli config set --if-unset`
- `--if-unset`：仅当配置文件里对应字段为空时写入，因此 `npm update` / 重装不会覆盖用户已改过的地址
