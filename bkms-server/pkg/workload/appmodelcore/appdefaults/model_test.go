package appdefaults_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

var _ = Describe("Application default rule model", func() {
	DescribeTable(
		"accepts supported AppSpec sections",
		func(configType appdefaults.ConfigType, spec *appspec.AppSpec) {
			rule := &appdefaults.Rule{
				WorkspaceID: "workspace-supported-sections",
				ConfigType:  configType,
				EnvTypes:    []string{"staging"},
				Spec:        spec,
			}
			Expect(appdefaults.ValidateRule(rule)).To(Succeed())
		},
		Entry(
			"resources",
			appspec.AppSpecSectionResources,
			&appspec.AppSpec{Resources: resourcesSpec(1, "1", "2", "2Gi", "4Gi")},
		),
		Entry(
			"dev mode with an explicit false value",
			appspec.AppSpecSectionDevMode,
			&appspec.AppSpec{DevMode: &appspec.DevModeSpec{Enabled: lo.ToPtr(false)}},
		),
	)

	It("rejects dev-mode rules targeting production environments", func() {
		rule := &appdefaults.Rule{
			WorkspaceID: "workspace-production-dev-mode",
			ConfigType:  appspec.AppSpecSectionDevMode,
			EnvTypes:    []string{"production"},
			Spec:        &appspec.AppSpec{DevMode: &appspec.DevModeSpec{Enabled: lo.ToPtr(false)}},
		}
		Expect(errors.Is(appdefaults.ValidateRule(rule), appdefaults.ErrInvalidRule)).To(BeTrue())
	})

	It("rejects unsupported sections", func() {
		rule := &appdefaults.Rule{
			WorkspaceID: "workspace-unsupported-section",
			ConfigType:  appspec.AppSpecSectionLabels,
			EnvTypes:    []string{"production"},
			Spec: &appspec.AppSpec{
				Labels: &appspec.LabelsSpec{Labels: map[string]string{"team": "platform"}},
			},
		}
		Expect(errors.Is(
			appdefaults.ValidateRule(rule),
			appdefaults.ErrInvalidRule,
		)).To(BeTrue())
	})

	It("rejects a mismatched or additional section", func() {
		rule := &appdefaults.Rule{
			WorkspaceID: "workspace-section-mismatch",
			ConfigType:  appspec.AppSpecSectionResources,
			EnvTypes:    []string{"production"},
			Spec: &appspec.AppSpec{
				Resources: resourcesSpec(1, "1", "2", "2Gi", "4Gi"),
				DevMode:   &appspec.DevModeSpec{Enabled: lo.ToPtr(true)},
			},
		}
		Expect(errors.Is(
			appdefaults.ValidateRule(rule),
			appdefaults.ErrInvalidRule,
		)).To(BeTrue())
	})

	It("requires every resources field", func() {
		resources := resourcesSpec(1, "1", "2", "2Gi", "4Gi")
		resources.CPURequests = nil
		rule := &appdefaults.Rule{
			WorkspaceID: "workspace-incomplete-resources",
			ConfigType:  appspec.AppSpecSectionResources,
			EnvTypes:    []string{"production"},
			Spec:        &appspec.AppSpec{Resources: resources},
		}
		Expect(errors.Is(
			appdefaults.ValidateRule(rule),
			appdefaults.ErrInvalidRule,
		)).To(BeTrue())
	})

	It("requires an explicit dev-mode value", func() {
		rule := &appdefaults.Rule{
			WorkspaceID: "workspace-incomplete-dev-mode",
			ConfigType:  appspec.AppSpecSectionDevMode,
			EnvTypes:    []string{"production"},
			Spec:        &appspec.AppSpec{DevMode: &appspec.DevModeSpec{}},
		}
		Expect(errors.Is(
			appdefaults.ValidateRule(rule),
			appdefaults.ErrInvalidRule,
		)).To(BeTrue())
	})

	It("requires environment types and empty AppSpec identity", func() {
		rule := &appdefaults.Rule{
			WorkspaceID: "workspace-invalid-identity",
			ConfigType:  appspec.AppSpecSectionResources,
			Spec: &appspec.AppSpec{
				Resources: resourcesSpec(1, "1", "2", "2Gi", "4Gi"),
			},
		}
		Expect(errors.Is(
			appdefaults.ValidateRule(rule),
			appdefaults.ErrInvalidRule,
		)).To(BeTrue())

		rule.EnvTypes = []string{"production"}
		rule.Spec = &appspec.AppSpec{
			AppID:     "application",
			Resources: resourcesSpec(1, "1", "2", "2Gi", "4Gi"),
		}
		Expect(errors.Is(
			appdefaults.ValidateRule(rule),
			appdefaults.ErrInvalidRule,
		)).To(BeTrue())
	})

	It("accepts multiple environment types and deduplicates in first-seen order", func() {
		rule := &appdefaults.Rule{
			WorkspaceID: "workspace-multiple-env-types",
			ConfigType:  appspec.AppSpecSectionResources,
			EnvTypes:    []string{"production", "production", "staging"},
			Spec:        &appspec.AppSpec{Resources: resourcesSpec(1, "1", "2", "2Gi", "4Gi")},
		}
		Expect(appdefaults.ValidateRule(rule)).To(Succeed())
		Expect(rule.EnvTypes).To(Equal([]string{"production", "staging"}))
	})

	It("rejects an invalid environment type in the list", func() {
		rule := &appdefaults.Rule{
			WorkspaceID: "workspace-invalid-env-type",
			ConfigType:  appspec.AppSpecSectionResources,
			EnvTypes:    []string{"production", "prod"},
			Spec:        &appspec.AppSpec{Resources: resourcesSpec(1, "1", "2", "2Gi", "4Gi")},
		}
		Expect(errors.Is(
			appdefaults.ValidateRule(rule),
			appdefaults.ErrInvalidRule,
		)).To(BeTrue())
	})
})

func resourcesSpec(
	replicas int32,
	cpuRequests, cpuLimits, memoryRequests, memoryLimits string,
) *appspec.ResourcesSpec {
	return &appspec.ResourcesSpec{
		Replicas:       lo.ToPtr(replicas),
		CPURequests:    lo.ToPtr(cpuRequests),
		CPULimits:      lo.ToPtr(cpuLimits),
		MemoryRequests: lo.ToPtr(memoryRequests),
		MemoryLimits:   lo.ToPtr(memoryLimits),
	}
}
