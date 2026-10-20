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

// Package bcstoken 提供 BCS 用户 Token 的获取能力
package bcstoken

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
)

// ErrNoActiveUserToken 用户没有有效的 BCS Token
var ErrNoActiveUserToken = errors.New("no active BCS user token")

// FetchActiveUserToken 获取当前用户在指定 BCS 项目下状态为 active 的 Token
func FetchActiveUserToken(ctx context.Context, bcsProjectCode string) (string, error) {
	user, err := auth.GetUser(ctx)
	if err != nil {
		return "", errors.Wrap(err, "get auth user")
	}

	client, err := bcs.New(user, bcs.ClientOption{UserMode: true})
	if err != nil {
		return "", errors.Wrap(err, "init BCS client")
	}
	tokens, err := client.ListUserTokens(ctx)
	if err != nil {
		return "", errors.Wrap(err, "list BCS user tokens")
	}

	active, found := lo.Find(tokens, func(token bcs.UserToken) bool {
		return token.Status == bcs.UserTokenStatusActive
	})
	if !found {
		return "", errors.Wrapf(
			ErrNoActiveUserToken, "please create a token in BCS at %s", UserTokenURL(bcsProjectCode),
		)
	}
	return active.Token, nil
}

// UserTokenURL 返回 BCS 用户 Token 创建页地址
func UserTokenURL(bcsProjectCode string) string {
	return fmt.Sprintf(
		"%s/bcs/projects/%s/user-token",
		strings.TrimRight(config.G.BCS.BaseUrl, "/"),
		bcsProjectCode,
	)
}
