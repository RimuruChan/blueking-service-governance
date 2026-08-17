package appdefaults

import (
	"github.com/pkg/errors"
	"github.com/samber/lo"

	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// ValidateRule requires at least one valid environment type and exactly one
// complete, supported AppSpec section.
func ValidateRule(rule *Rule) error {
	if rule.ConfigType != appspec.AppSpecSectionResources && rule.ConfigType != appspec.AppSpecSectionDevMode {
		return errors.Wrapf(ErrInvalidRule, "unsupported config type %q", rule.ConfigType)
	}
	rule.EnvTypes = lo.Uniq(rule.EnvTypes)
	if len(rule.EnvTypes) == 0 {
		return errors.Wrapf(ErrInvalidRule, "envTypes must contain at least one environment type")
	}
	for _, envType := range rule.EnvTypes {
		if !bkmsenv.IsValidEnvType(envType) {
			return errors.Wrapf(ErrInvalidRule, "envTypes must contain only valid environment types")
		}
	}
	if rule.Spec == nil {
		return errors.Wrapf(ErrInvalidRule, "spec is required")
	}
	if rule.Spec.AppID != "" || rule.Spec.EnvName != "" {
		return errors.Wrapf(ErrInvalidRule, "spec identity must be empty in a workspace rule")
	}

	sections := configuredSections(rule.Spec)
	if len(sections) != 1 || sections[0] != rule.ConfigType {
		return errors.Wrapf(ErrInvalidRule, "spec must contain only the %s section", rule.ConfigType)
	}

	if err := validateSpecCompleteness(rule.ConfigType, rule.Spec); err != nil {
		return err
	}

	// Delegate to appspec's own validator for cross-field checks
	// (e.g. request ≤ limit).
	validationSpec := appspec.Clone(rule.Spec)
	validationSpec.AppID = "rule-validation"
	if err := appspec.Validate(validationSpec); err != nil {
		return errors.Wrapf(ErrInvalidRule, "invalid %s configuration: %v", rule.ConfigType, err)
	}
	return nil
}

func validateSpecCompleteness(configType ConfigType, spec *appspec.AppSpec) error {
	switch configType {
	case appspec.AppSpecSectionResources:
		r := spec.Resources
		if r.Replicas == nil || r.CPURequests == nil ||
			r.CPULimits == nil || r.MemoryRequests == nil ||
			r.MemoryLimits == nil {
			return errors.Wrapf(ErrInvalidRule, "resources rule must contain all fields")
		}
	case appspec.AppSpecSectionDevMode:
		if spec.DevMode.Enabled == nil {
			return errors.Wrapf(ErrInvalidRule, "devMode rule must contain enabled")
		}
	}
	return nil
}

func configuredSections(spec *appspec.AppSpec) []ConfigType {
	var sections []ConfigType
	if spec.Resources != nil {
		sections = append(sections, appspec.AppSpecSectionResources)
	}
	if spec.UpdateStrategy != nil {
		sections = append(sections, appspec.AppSpecSectionUpdateStrategy)
	}
	if spec.DevMode != nil {
		sections = append(sections, appspec.AppSpecSectionDevMode)
	}
	if spec.Lifecycle != nil {
		sections = append(sections, appspec.AppSpecSectionLifecycle)
	}
	if spec.Probes != nil {
		sections = append(sections, appspec.AppSpecSectionProbe)
	}
	if spec.Labels != nil {
		sections = append(sections, appspec.AppSpecSectionLabels)
	}
	if spec.Annotations != nil {
		sections = append(sections, appspec.AppSpecSectionAnnotations)
	}
	if spec.TkeRouteEni != nil {
		sections = append(sections, appspec.AppSpecSectionTkeRouteEni)
	}
	return sections
}
