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

// Package handler contains Gin handlers for HostPort APIs.
package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/hostport"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/hostport/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// Handler handles Gin HostPort API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

func (h *Handler) service() *hostport.Service {
	return hostport.NewService(h.registry.HostPortStore, h.registry.EnvStore)
}

// ListHostPorts 获取应用 HostPort 列表及联邦环境待部署状态。
//
//	@ID			ListHostPorts
//	@Summary	获取应用 HostPort 列表及联邦环境待部署状态
//	@Tags		hostport
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	serializer.HostPortsOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/hostports [get]
func (h *Handler) ListHostPorts(c *gin.Context) {
	var uriInput serializer.AppURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	result, err := h.service().GetHostPorts(ctx, app)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get hostports"))
		return
	}
	ginutils.OK(c, new(serializer.HostPortsOutput).FromModel(result))
}

// CreateHostPort 新增应用 HostPort。
//
//	@ID			CreateHostPort
//	@Summary	新增应用 HostPort
//	@Tags		hostport
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string							true	"应用 ID"
//	@Param		body	body		serializer.CreateHostPortInput	true	"请求体"
//	@Success	201		{object}	serializer.HostPortsOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/hostports [post]
func (h *Handler) CreateHostPort(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var input serializer.CreateHostPortInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if _, err = h.service().AddPort(ctx, app.ID, input.ContainerPort); err != nil {
		if errors.Is(err, hostport.ErrInvalidPort) {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Errorf(bkerrs.ErrCodeInvalidRequest, "invalid container port: %d", input.ContainerPort),
			)
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create hostport"))
		return
	}
	result, err := h.service().GetHostPorts(ctx, app)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get hostports"))
		return
	}
	ginutils.Created(c, new(serializer.HostPortsOutput).FromModel(result))
}

// DeleteHostPort 删除应用 HostPort。
//
//	@ID			DeleteHostPort
//	@Summary	删除应用 HostPort
//	@Tags		hostport
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID			path		string	true	"应用 ID"
//	@Param		containerPort	path		int		true	"容器端口"
//	@Success	200				{object}	serializer.HostPortsOutput
//	@Failure	400				{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/hostports/{containerPort} [delete]
func (h *Handler) DeleteHostPort(c *gin.Context) {
	var uriInput serializer.DeleteHostPortURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if err = h.service().RemovePort(ctx, app.ID, uriInput.ContainerPort); err != nil {
		if errors.Is(err, hostport.ErrInvalidPort) {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Errorf(bkerrs.ErrCodeInvalidRequest, "invalid container port: %d", uriInput.ContainerPort),
			)
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete hostport"))
		return
	}
	result, err := h.service().GetHostPorts(ctx, app)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get hostports"))
		return
	}
	ginutils.OK(c, new(serializer.HostPortsOutput).FromModel(result))
}
