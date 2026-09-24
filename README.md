![img](./docs/resource/img/blueking_service_governance.png)

---

[![license](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat)](LICENSE) [![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/RimuruChan/blueking-service-governance/pulls)

简体中文 | [English](README_EN.md)

> 注意：主干分支在开发过程中可能处于不可用状态。
> 请通过稳定版本或发布分支获取可用于生产环境的代码。

蓝鲸服务治理平台面向游戏开发者、SRE 提供一站式应用全生命周期管理服务。

本平台围绕应用托管、制品交付、开发联调、应用观测、管理策略和持续部署等场景，帮助业务团队以更低成本完成微服务应用的构建、运行、发布和治理。

## 架构设计

蓝鲸服务治理采用前后端分离与多模块协作的工程结构：

- `bkms-server`：服务端核心服务，提供空间、环境、应用、组件、部署等领域能力，并作为前端对接的主要 API 入口。
- `bkms-ui`：Web 前端项目，提供服务治理平台的产品化交互界面。
- `bkms-cli`：命令行工具，支持应用信息查看、构建、部署、发布和部署结果查询等能力。
- `bkms-dockerfile-generator`：镜像构建流程中的 Dockerfile 生成工具，基于流水线配置生成平台默认 Dockerfile。
- `libs/bkms-adapter`：公共适配层模块，用于封装与外部系统或基础设施的对接逻辑。
- `charts`：Helm Chart 部署清单，包含服务端和前端的部署资源。

## 功能特性

针对游戏开发者，主要提供以下核心功能：

- 制品托管：基于代码提供标准化 CI 制品构建；基于制品探测实现事件驱动交付能力。
- 应用托管：根据业务场景提供应用定义，构建应用架构并简化底层运行时配置，实现业务应用托管服务。
- 开发联调：基于应用服务架构，为开发者提供一特性一环境的个人调试环境和团队调试环境。
- 应用观测：提供日志采集与查询服务、应用基础质量监测、自定义指标采集和监控告警能力。
- 管理策略：提供面向业务场景的精简管理策略，简化应用弹性伸缩、容灾调度等复杂场景。
- 应用发布：基于单应用快速完成服务部署、滚动更新和灰度更新。

针对 SRE，主要提供以下核心功能：

- 环境管理：提供基础资源管理与引用能力，为业务环境筹备合适的运行时。
- 组件管理：基于业务场景构建组件实例，降低开发者配置和接入成本。
- 流程管理：基于应用制品晋级等流程实现审核校验。
- 持续部署：基于单应用、应用组实现高效持续部署。

## 代码目录说明

- `bkms-server`：蓝鲸服务治理服务端，提供 REST API、领域服务、异步任务和数据访问等能力。
- `bkms-ui`：蓝鲸服务治理前端工程，基于 Vue 3 构建。
- `bkms-cli`：蓝鲸服务治理命令行工具，提供应用、环境、工作空间、构建和部署相关命令。
- `bkms-dockerfile-generator`：Dockerfile 生成工具，用于在镜像构建流程中生成平台默认 Dockerfile。
- `libs/bkms-adapter`：公共适配层模块，用于封装与外部系统或基础设施的对接逻辑。
- `charts/bkms-server`：服务端 Helm Chart 部署清单。
- `charts/bkms-ui`：前端 Helm Chart 部署清单。

## 快速开始

- [本地开发部署指引](docs/DEVELOP_GUIDE.md)
- [服务端开发指引](./bkms-server/README.md)
- [前端开发指引](./bkms-ui/README.md)
- [命令行工具开发指引](./bkms-cli/README.md)
- [Dockerfile 生成工具说明](./bkms-dockerfile-generator/README.md)

## 支持

- [蓝鲸智云 - 学习社区](https://bk.tencent.com/s-mart/community)
- [蓝鲸 DevOps 在线视频教程](https://bk.tencent.com/s-mart/video)
- [蓝鲸智云官网](https://bk.tencent.com/)

## 蓝鲸社区

- [BK-PaaS](https://github.com/TencentBlueKing/blueking-paas)：蓝鲸 PaaS 平台是开放式的开发平台，让开发者可以方便快捷地创建、开发、部署和管理
  SaaS 应用。
- [BK-APIGW](https://github.com/TencentBlueKing/blueking-apigateway)：蓝鲸 API 网关是高性能，高可用的 API
  托管服务，帮助开发者创建、发布、维护、监控和保护 API。
- [BK-CI](https://github.com/TencentBlueKing/bk-ci)：蓝鲸持续集成平台是一个开源的持续集成和持续交付系统，可以轻松将你的研发流程呈现到你面前。
- [BK-BCS](https://github.com/TencentBlueKing/bk-bcs)：蓝鲸容器管理平台是以容器技术为基础，为微服务业务提供编排管理的基础服务平台。
- [BK-SOPS](https://github.com/TencentBlueKing/bk-sops)：标准运维（SOPS）是通过可视化的图形界面进行任务流程编排和执行的系统。
- [BK-JOB](https://github.com/TencentBlueKing/bk-job)：蓝鲸作业平台（Job）是一套运维脚本管理系统，具备海量任务并发处理能力。
- [BK-CMDB](https://github.com/TencentBlueKing/bk-cmdb)：蓝鲸配置平台是一个面向资产及应用的企业级配置管理平台。

## 贡献

如果你有好的意见或建议，欢迎通过 Issues 或 Pull Requests 参与项目共建，为蓝鲸开源社区贡献力量。

[腾讯开源激励计划](https://opensource.tencent.com/contribution) 鼓励开发者的参与和贡献，期待你的加入。

## 协议

本项目基于 MIT 协议，详细请参考 [LICENSE](LICENSE)
