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

// Package apm 提供 OpenTelemetry Trace + Log 统一接入及 Gin 可观测性中间件
package apm

import (
	"cmp"
	"context"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"

	"github.com/RimuruChan/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/RimuruChan/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/RimuruChan/blueking-service-governance/bkms-server/pkg/common/version"
)

const (
	// defaultAppName 与历史 tRPC server.app 配置保持一致
	defaultAppName = "bkms"
	// defaultServerName 与历史 tRPC server.server 配置保持一致，当调用方未传入 serverRole 时使用
	defaultServerName = "bkmsserver"
	// defaultTenantID 与原内部 SDK 的默认租户保持一致
	defaultTenantID = "default"
	// tenantHeaderKey 与原内部 SDK 发送给蓝鲸监控采集端的租户 Header 保持一致
	tenantHeaderKey = "x-bk-token"
	// legacyTenantIDAttributeKey 保留原内部 SDK 上报的租户资源属性
	legacyTenantIDAttributeKey = "tps.tenant.id"
	// legacyTelemetrySDKName 保留原内部 SDK 上报的 telemetry SDK 名称
	legacyTelemetrySDKName = "opentelemetry"
	// legacyServiceNameAttributeKey 保留历史 oteltrpc attributes 中的 service_name 属性
	legacyServiceNameAttributeKey = "service_name"
	// grpcCompressorName 与原内部 SDK 的 gRPC gzip 压缩配置保持一致
	grpcCompressorName = "gzip"
	// maxSendMessageSize 与原内部 SDK 的 gRPC 单次发送大小限制保持一致
	maxSendMessageSize = 4194304
	// exporterRetryInitialInterval 与官方 exporter 默认重试初始间隔一致
	exporterRetryInitialInterval = 5 * time.Second
	// exporterRetryMaxInterval 与官方 exporter 默认重试最大间隔一致
	exporterRetryMaxInterval = 30 * time.Second
	// exporterRetryMaxElapsedTime 与官方 exporter 默认重试总耗时一致
	exporterRetryMaxElapsedTime = time.Minute
)

// setupConfig 是 trace/log provider 共用的初始化参数
type setupConfig struct {
	Endpoint    string
	HTTPEnabled bool
	TenantID    string
	ServiceName string
}

// Setup 初始化全局 OTel Provider（Trace + Log），失败时仅记录日志不阻断服务。
// 返回统一的 shutdown 函数，配置不可用时返回空实现。
func Setup(ctx context.Context, cfg config.BkMonitorConfig, serverRole string) func(context.Context) error {
	setupCfg := resolveSetupConfig(ctx, cfg, serverRole)
	if setupCfg.Endpoint == "" {
		log.Warn(ctx, "bk monitor APM endpoint is empty, skip APM setup")
		return noopShutdown
	}

	// 剥离 cancel 传播，避免进程退出时 exporter 连接被提前关闭
	exporterCtx := context.WithoutCancel(ctx)

	// 1. 初始化 Log Provider
	shutdownLog, err := initLogProvider(exporterCtx, setupCfg)
	if err != nil {
		log.Warnf(ctx, "failed to setup OTel log provider, skip log export: %v", err)
	}

	// 2. 初始化 Trace Provider
	shutdownTrace, err := setupTraceProvider(exporterCtx, setupCfg)
	if err != nil {
		log.Warnf(ctx, "failed to setup bk monitor APM, skip APM setup: %v", err)
		return shutdownAll(shutdownLog, nil)
	}

	return shutdownAll(shutdownLog, shutdownTrace)
}

// ServiceName 返回 APM 服务名。优先使用 APMServiceName 配置，未配置时按 bkms.${serverRole} 拼接。
func ServiceName(cfg config.BkMonitorConfig, serverRole string) string {
	if cfg.APMServiceName != "" {
		return cfg.APMServiceName
	}
	return defaultAppName + "." + cmp.Or(serverRole, defaultServerName)
}

// shutdownAll 组合 shutdown 函数，按 log → trace 顺序关闭。
func shutdownAll(shutdownLog, shutdownTrace func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		if shutdownLog != nil {
			if err := shutdownLog(ctx); err != nil {
				log.Errorf(ctx, "shutdown APM log provider: %v", err)
			}
		}
		if shutdownTrace != nil {
			return shutdownTrace(ctx)
		}
		return nil
	}
}

// resolveSetupConfig 配置解析
func resolveSetupConfig(ctx context.Context, cfg config.BkMonitorConfig, serverRole string) setupConfig {
	endpoint, httpEnabled := resolveEndpoint(cfg)
	serviceName := ServiceName(cfg, serverRole)
	tenantID := resolveTenantID(ctx, cfg)

	return setupConfig{
		Endpoint:    endpoint,
		HTTPEnabled: httpEnabled,
		TenantID:    tenantID,
		ServiceName: serviceName,
	}
}

// resolveTenantID 解析租户 ID，APMToken 为空时使用默认租户
func resolveTenantID(ctx context.Context, cfg config.BkMonitorConfig) string {
	tenantID := cmp.Or(strings.TrimSpace(cfg.APMToken), defaultTenantID)
	if cfg.APMToken == "" {
		log.Warn(ctx, "bk monitor APM token is empty, use default tenant id")
	}
	return tenantID
}

// resolveEndpoint 返回上报地址及是否走 HTTP exporter。
// 优先使用 APMHttpEndpoint，其次 APMEndpoint（按 scheme 判断协议）。
func resolveEndpoint(cfg config.BkMonitorConfig) (string, bool) {
	httpEndpoint := strings.TrimSpace(cfg.APMHttpEndpoint)
	if httpEndpoint != "" {
		return httpEndpoint, true
	}

	endpoint := strings.TrimSpace(cfg.APMEndpoint)
	if endpoint == "" {
		return "", false
	}
	return endpoint, strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://")
}

// newResource 构建 OTel Resource（服务名、版本、租户等属性）。
func newResource(ctx context.Context, cfg setupConfig) (*resource.Resource, error) {
	extraRes, err := resource.New(
		ctx,
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithHost(),
		resource.WithAttributes(
			attribute.String(legacyTenantIDAttributeKey, cfg.TenantID),
			semconv.TelemetrySDKLanguageGo,
			semconv.TelemetrySDKNameKey.String(legacyTelemetrySDKName),
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(version.Version),
			attribute.String(legacyServiceNameAttributeKey, cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, err
	}

	return resource.Merge(resource.Default(), extraRes)
}

func noopShutdown(context.Context) error {
	return nil
}
