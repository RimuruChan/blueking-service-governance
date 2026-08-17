// Package serializer defines Gin serializers for workspace application defaults.
package serializer

import (
	"time"

	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
)

// WorkspaceAppTypeURIInput binds a workspace + app-type scoped path.
type WorkspaceAppTypeURIInput struct {
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
	AppType     string `uri:"appType" binding:"required"`
}

// RuleURIInput binds a workspace rule path.
type RuleURIInput struct {
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
	AppType     string `uri:"appType" binding:"required"`
	RuleID      string `uri:"ruleID" binding:"required,mongodb"`
}

// RuleOutputFields contains fields shared by every section rule response.
type RuleOutputFields struct {
	ID        string    `json:"id"`
	EnvTypes  []string  `json:"envTypes"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (output *RuleOutputFields) fromModel(rule appdefaults.Rule) {
	*output = RuleOutputFields{
		ID:        rule.ID.Hex(),
		EnvTypes:  rule.EnvTypes,
		CreatedAt: rule.CreatedAt,
		UpdatedAt: rule.UpdatedAt,
	}
}

// EmptyOutput is an empty successful response.
type EmptyOutput struct{}
