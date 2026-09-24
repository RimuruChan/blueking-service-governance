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

package app

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/client"
)

// UpdateBuildConfig 读取 YAML spec 文件并更新应用构建配置。
func UpdateBuildConfig(ctx context.Context, appID, specFile string) error {
	data, err := os.ReadFile(specFile)
	if err != nil {
		return errors.Wrap(err, "read spec file")
	}

	var body client.AppBuildConfigUpdateBody
	if err = yaml.Unmarshal(data, &body); err != nil {
		return errors.Wrap(err, "parse spec file")
	}

	return errors.Wrap(client.New().UpdateAppBuildConfig(ctx, appID, body), "update build config")
}
