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

package apm

import (
	"context"
	"net/url"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	log "github.com/RimuruChan/blueking-service-governance/bkms-server/pkg/common/logging"
)

// initLogProvider 初始化 OTel Log Provider 并注入全局 slog Handler
func initLogProvider(ctx context.Context, setupCfg setupConfig) (func(context.Context) error, error) {
	exporter, err := newLogExporter(ctx, setupCfg)
	if err != nil {
		return nil, errors.Wrap(err, "create log exporter")
	}
	res, err := newResource(ctx, setupCfg)
	if err != nil {
		return nil, errors.Wrap(err, "build otel resource for log provider")
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)
	log.InitOTelHandler(provider)
	log.Infof(ctx, "bk monitor APM log provider setup completed, serviceName=%s", setupCfg.ServiceName)

	return provider.Shutdown, nil
}

// newLogExporter Log Exporter 构建
func newLogExporter(ctx context.Context, cfg setupConfig) (sdklog.Exporter, error) {
	if cfg.HTTPEnabled {
		return newHTTPLogExporter(ctx, cfg)
	}
	return newGRPCLogExporter(ctx, cfg)
}

// newHTTPLogExporter http
func newHTTPLogExporter(ctx context.Context, cfg setupConfig) (sdklog.Exporter, error) {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "parse APM log HTTP endpoint %q", cfg.Endpoint)
	}

	options := []otlploghttp.Option{
		otlploghttp.WithEndpoint(u.Host),
		otlploghttp.WithCompression(otlploghttp.GzipCompression),
		otlploghttp.WithHeaders(map[string]string{tenantHeaderKey: cfg.TenantID}),
		otlploghttp.WithRetry(otlploghttp.RetryConfig{
			Enabled:         true,
			InitialInterval: exporterRetryInitialInterval,
			MaxInterval:     exporterRetryMaxInterval,
			MaxElapsedTime:  exporterRetryMaxElapsedTime,
		}),
	}
	// 仅当 endpoint 显式携带 path 时才透传，避免覆盖 otlploghttp 默认的 /v1/logs
	if u.Path != "" && u.Path != "/" {
		options = append(options, otlploghttp.WithURLPath(u.Path))
	}
	// 非 https scheme 视为明文上报，走 WithInsecure
	if u.Scheme != "https" {
		options = append(options, otlploghttp.WithInsecure())
	}

	return otlploghttp.New(ctx, options...)
}

// newGRPCLogExporter grpc
func newGRPCLogExporter(ctx context.Context, cfg setupConfig) (sdklog.Exporter, error) {
	return otlploggrpc.New(ctx,
		otlploggrpc.WithTLSCredentials(insecure.NewCredentials()),
		otlploggrpc.WithEndpoint(cfg.Endpoint),
		otlploggrpc.WithCompressor(grpcCompressorName),
		otlploggrpc.WithHeaders(map[string]string{tenantHeaderKey: cfg.TenantID}),
		otlploggrpc.WithDialOption(grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(maxSendMessageSize))),
		otlploggrpc.WithRetry(otlploggrpc.RetryConfig{
			Enabled:         true,
			InitialInterval: exporterRetryInitialInterval,
			MaxInterval:     exporterRetryMaxInterval,
			MaxElapsedTime:  exporterRetryMaxElapsedTime,
		}),
	)
}
