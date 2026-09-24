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

package env

import (
	"context"

	"github.com/pkg/errors"

	"github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// UpdateEnv 通过环境名称解析 ID，更新环境基本信息（displayName / type）。
func UpdateEnv(ctx context.Context, workspaceID, envName string, body client.UpdateEnvBasicInfoBody) error {
	cli := client.New()

	envID, err := ResolveEnvIDByName(ctx, cli, workspaceID, envName)
	if err != nil {
		return err
	}

	if err = cli.UpdateEnvBasicInfo(ctx, envID, body); err != nil {
		return errors.Wrap(err, "update env basic info")
	}

	console.Info("✓ Environment %s updated successfully", envName)
	return nil
}
