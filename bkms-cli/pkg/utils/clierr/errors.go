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

// Package clierr distinguishes usage errors, reported failures, and cancellations.
package clierr

import "github.com/pkg/errors"

// ErrCancelled 表示用户取消操作，保留非零退出码但不作为执行错误展示。
var ErrCancelled = errors.New("operation cancelled")

// UsageError 表示参数或输入格式错误，需要附带命令用法。
type UsageError struct{ Err error }

func (e *UsageError) Error() string { return e.Err.Error() }
func (e *UsageError) Unwrap() error { return e.Err }

// Usage 标记参数错误，供统一错误出口识别。
func Usage(err error) error { return &UsageError{Err: err} }

// Usagef 创建参数错误，支持格式化消息。
func Usagef(format string, args ...any) error {
	return Usage(errors.Errorf(format, args...))
}

// ReportedError 表示失败明细已经输出，仅需返回非零退出码。
type ReportedError struct{ Err error }

func (e *ReportedError) Error() string { return e.Err.Error() }
func (e *ReportedError) Unwrap() error { return e.Err }

// Reported 标记已输出明细的失败，保留错误链而不重复打印。
func Reported(err error) error { return &ReportedError{Err: err} }

// Reportedf 创建已输出明细的失败，支持格式化消息。
func Reportedf(format string, args ...any) error {
	return Reported(errors.Errorf(format, args...))
}
