![img](./docs/resource/img/blueking_service_governance.png)

---

[![license](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat)](LICENSE) [![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/RimuruChan/blueking-service-governance/pulls)

[简体中文](README.md) | English

> Note: The main branch may be unstable during development.
> Please use stable versions or release branches for production-ready code.

BlueKing Service Governance provides game developers and SREs with one-stop application lifecycle management services. 

The platform centered around application hosting, artifact delivery, development debugging, application observability, governance policies, and continuous deployment, it helps business teams build, run, release, and govern service applications at lower cost.

## Architecture

BlueKing Service Governance uses a frontend-backend separated architecture with multiple collaborating modules:

- `bkms-server`: The core backend service, providing domain capabilities such as workspaces, environments, applications, components, and deployments. It also serves as the main API entry point for the frontend.
- `bkms-ui`: The web frontend project, providing the product UI for the service governance platform.
- `bkms-cli`: A command-line tool that supports application information query, build, deployment, release, and deployment result query.
- `bkms-dockerfile-generator`: A Dockerfile generation tool used in the image build process to generate platform-default Dockerfiles based on pipeline configuration.
- `libs/bkms-adapter`: A common adapter module that encapsulates integration logic with external systems or infrastructure.
- `charts`: Helm Chart deployment manifests for the backend and frontend services.

## Features

For game developers, the platform provides the following core capabilities:

- Artifact hosting: Provides standardized CI artifact builds based on source code and event-driven delivery based on artifact detection.
- Application hosting: Provides application definitions for business scenarios, builds application architecture, simplifies underlying runtime configuration, and enables managed application hosting.
- Development debugging: Provides per-feature personal debugging environments and team debugging environments based on the application service architecture.
- Application observability: Provides log collection and query, basic application quality monitoring, custom metric collection, and monitoring alerts.
- Governance policies: Provides simplified governance policies for business scenarios, reducing the complexity of application autoscaling, disaster recovery scheduling, and similar scenarios.
- Application release: Supports fast deployment, rolling updates, and canary releases for individual applications.

For SREs, the platform provides the following core capabilities:

- Environment management: Provides basic resource management and referencing capabilities to prepare suitable runtimes for business environments.
- Component management: Builds component instances based on business scenarios to reduce developer configuration and integration costs.
- Process management: Implements review and validation based on processes such as application artifact promotion.
- Continuous deployment: Enables efficient continuous deployment based on individual applications and application groups.

## Code Directory

- `bkms-server`: The BlueKing Service Governance backend service, providing REST APIs, domain services, asynchronous tasks, and data access capabilities.
- `bkms-ui`: The BlueKing Service Governance frontend project, built with Vue 3.
- `bkms-cli`: The BlueKing Service Governance command-line tool, providing commands related to applications, environments, workspaces, builds, and deployments.
- `bkms-dockerfile-generator`: A Dockerfile generation tool used to generate platform-default Dockerfiles in the image build process.
- `libs/bkms-adapter`: A common adapter module that encapsulates integration logic with external systems or infrastructure.
- `charts/bkms-server`: Helm Chart deployment manifests for the backend service.
- `charts/bkms-ui`: Helm Chart deployment manifests for the frontend service.

## Getting Started

- [Local Development Guide](docs/DEVELOP_GUIDE.md)
- [Backend Development Guide](./bkms-server/README.md)
- [Frontend Development Guide](./bkms-ui/README.md)
- [CLI Development Guide](./bkms-cli/README.md)
- [Dockerfile Generator Guide](./bkms-dockerfile-generator/README.md)

## Support

- [BlueKing Learning Community](https://bk.tencent.com/s-mart/community)
- [BlueKing DevOps Video Tutorials](https://bk.tencent.com/s-mart/video)
- [BlueKing Official Website](https://bk.tencent.com/)

## BlueKing Community

- [BK-PaaS](https://github.com/TencentBlueKing/blueking-paas): Open development platform for creating, developing, deploying, and managing SaaS applications.
- [BK-APIGW](https://github.com/TencentBlueKing/blueking-apigateway): High-performance and highly available API gateway for creating, publishing, maintaining, monitoring, and securing APIs.
- [BK-CI](https://github.com/TencentBlueKing/bk-ci): Open-source CI/CD system for streamlining development workflows.
- [BK-BCS](https://github.com/TencentBlueKing/bk-bcs): Container management platform for microservice orchestration.
- [BK-SOPS](https://github.com/TencentBlueKing/bk-sops): Visual task orchestration and execution system.
- [BK-JOB](https://github.com/TencentBlueKing/bk-job): Script management system for large-scale task processing.
- [BK-CMDB](https://github.com/TencentBlueKing/bk-cmdb): Enterprise-grade configuration management platform for assets and applications.

## Contribution

We welcome contributions via Issues or Pull Requests to help build the project and contribute to the BlueKing open-source community.

Join the [Tencent OpenSource Plan](https://opensource.tencent.com/contribution) to participate in open-source development.

## License

This project is based on the MIT License. For details, see [LICENSE](LICENSE).
