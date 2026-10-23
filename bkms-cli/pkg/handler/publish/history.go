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

package publish

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// actionRestart 进程重启记录的操作类型
const actionRestart = "restart"

// PublishRecordRow 发布记录展示行
type PublishRecordRow struct {
	// Action 操作类型：publish / restart
	Action     string `json:"action"`
	EnvName    string `json:"envName"`
	Instance   string `json:"instance"`
	BinaryName string `json:"binaryName"`
	FileSize   string `json:"fileSize"`
	MD5        string `json:"md5"`
	Status     string `json:"status"`
	// Message 失败原因等附加信息，仅在 JSON/YAML 输出中保留，表格不展示
	Message   string `json:"message" table:"-"`
	Operator  string `json:"operator"`
	UpdatedAt string `json:"updatedAt"`
}

// ListHistory 获取开发模式发布历史记录（展示格式）。
// envName 与 envTypes 二选一：指定 envName 时只查该环境，否则按环境类型跨环境查询。
func ListHistory(
	ctx context.Context,
	cli client.Client,
	appID, envName string, envTypes []string, keyword string,
) ([]PublishRecordRow, error) {
	records, err := listRecords(ctx, cli, appID, envName, envTypes, keyword)
	if err != nil {
		return nil, err
	}

	rows := lo.Map(records, func(record client.DevModePublishRecord, _ int) PublishRecordRow {
		// restart 记录不涉及二进制文件，不展示文件大小
		fileSize := lo.Ternary(record.Action == actionRestart, "", formatFileSize(record.FileSize))
		return PublishRecordRow{
			Action:     lo.CoalesceOrEmpty(record.Action, "publish"),
			EnvName:    record.EnvName,
			Instance:   record.Instance,
			BinaryName: record.BinaryName,
			FileSize:   fileSize,
			MD5:        record.MD5,
			Status:     record.Status,
			Message:    record.Message,
			Operator:   record.Operator,
			UpdatedAt:  record.UpdatedAt,
		}
	})
	return rows, nil
}

// listRecords 按环境或环境类型查询发布记录
func listRecords(
	ctx context.Context,
	cli client.Client,
	appID, envName string, envTypes []string, keyword string,
) ([]client.DevModePublishRecord, error) {
	query := client.DevModePublishRecordsQuery{Keyword: keyword, Page: 1, PageSize: 100}
	if envName != "" {
		query.EnvNames = []string{envName}
	} else {
		query.EnvTypes = envTypes
	}
	records, _, err := cli.ListDevModePublishRecords(ctx, appID, query)
	return records, errors.Wrapf(err, "list dev mode publish records (env %q, env types %v)", envName, envTypes)
}

// formatFileSize 将字节数格式化为人类可读字符串（B / KB / MB / GB ...）。
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
