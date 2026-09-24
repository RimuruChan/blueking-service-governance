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

package deploy

import (
	"context"

	"github.com/pkg/errors"

	"github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/constant"
	"github.com/RimuruChan/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// DeleteDeploy 根据应用类型路由到对应的删除接口，执行部署删除并打印成功消息。
// appType 由调用方传入（cmd 层已为确认提示取过 app 信息，避免重复 API 调用）。
func DeleteDeploy(ctx context.Context, appType, appID, envName, deployID string) error {
	cli := client.New()

	var err error
	switch appType {
	case constant.AppTypeHelm:
		if deployID == "" {
			return errors.New("--deploy-id is required for helm applications (use 'deploy list' to get deploy IDs)")
		}
		err = cli.DeleteHelmDeploy(ctx, appID, envName, deployID)
	case constant.AppTypeTrpc:
		err = cli.DeleteTrpcDeploy(ctx, appID, envName)
	case constant.AppTypeTaf:
		err = cli.DeleteTafDeploy(ctx, appID, envName)
	default:
		return errors.Errorf("unsupported app type '%s' for deploy delete", appType)
	}
	if err != nil {
		return errors.Wrap(err, "delete deploy")
	}

	console.Info("✓ Deployment deleted for app %s in env %s", appID, envName)
	return nil
}
