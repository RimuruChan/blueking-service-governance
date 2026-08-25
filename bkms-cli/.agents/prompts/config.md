# bkms-cli config Reference

管理本地配置文件（`${HOME}/.bkms/config.yaml`）。

## 查看配置

```bash
bkms-cli config view
```

## 设置 API 地址

```bash
bkms-cli config set --bkms-base-url https://bkms.example.com
bkms-cli config set --if-unset --bkms-base-url https://bkms.example.com
```

`--bkms-base-url` 必填。写入前会去掉 URL 尾部 `/`。

`--if-unset`：仅当配置文件中该字段为空时才写入（npm postinstall 用这个，避免更新时覆盖已有配置）。

`bkmsBaseUrl` 未配置时，`bkms-cli`（无参）、`login` 以及需鉴权的业务命令会提示先执行 `config set`。
