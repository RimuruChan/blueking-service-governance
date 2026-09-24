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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	log "github.com/RimuruChan/blueking-service-governance/bkms-server/pkg/common/logging"
)

func setupTraceProvider(ctx context.Context, cfg setupConfig) (func(context.Context) error, error) {
	exporter, err := newTraceExporter(ctx, cfg)
	if err != nil {
		return nil, err
	}

	res, err := newResource(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "build otel resource")
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		// TraceContext 支持 W3C traceparent/tracestate Header，otelgin 会自动从请求中提取上游 TraceID 并注入 span
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.Errorf(ctx, "[otel] error: %v", err)
	}))
	log.Infof(ctx, "bk monitor APM trace provider setup completed, serviceName=%s", cfg.ServiceName)

	return provider.Shutdown, nil
}

func newTraceExporter(ctx context.Context, cfg setupConfig) (sdktrace.SpanExporter, error) {
	if cfg.HTTPEnabled {
		return newHTTPTraceExporter(ctx, cfg)
	}
	return newGRPCTraceExporter(ctx, cfg)
}

func newHTTPTraceExporter(ctx context.Context, cfg setupConfig) (sdktrace.SpanExporter, error) {
	// 使用 url.Parse 解析 endpoint，将 Host 与 Path 分开传给 exporter，避免把带 path 的 URL
	// 整段作为 host 传入导致上报地址异常
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "parse APM HTTP endpoint %q", cfg.Endpoint)
	}

	options := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(u.Host),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
		otlptracehttp.WithHeaders(map[string]string{tenantHeaderKey: cfg.TenantID}),
		otlptracehttp.WithRetry(otlptracehttp.RetryConfig{
			Enabled:         true,
			InitialInterval: exporterRetryInitialInterval,
			MaxInterval:     exporterRetryMaxInterval,
			MaxElapsedTime:  exporterRetryMaxElapsedTime,
		}),
	}
	// 仅当 endpoint 显式携带 path 时才透传，避免覆盖 otlptracehttp 默认的 /v1/traces
	if u.Path != "" && u.Path != "/" {
		options = append(options, otlptracehttp.WithURLPath(u.Path))
	}
	// 非 https scheme 视为明文上报，走 WithInsecure
	if u.Scheme != "https" {
		options = append(options, otlptracehttp.WithInsecure())
	}
	return otlptracehttp.New(ctx, options...)
}

func newGRPCTraceExporter(ctx context.Context, cfg setupConfig) (sdktrace.SpanExporter, error) {
	return otlptracegrpc.New(ctx,
		otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()),
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithCompressor(grpcCompressorName),
		otlptracegrpc.WithHeaders(map[string]string{tenantHeaderKey: cfg.TenantID}),
		otlptracegrpc.WithDialOption(grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(maxSendMessageSize))),
		otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{
			Enabled:         true,
			InitialInterval: exporterRetryInitialInterval,
			MaxInterval:     exporterRetryMaxInterval,
			MaxElapsedTime:  exporterRetryMaxElapsedTime,
		}),
	)
}
