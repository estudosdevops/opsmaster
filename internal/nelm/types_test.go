package nelm

import (
	"testing"
	"time"
)

func TestBaseOptions(t *testing.T) {
	tests := []struct {
		name     string
		base     BaseOptions
		expected map[string]any
	}{
		{
			name: "BaseOptions com todos os campos preenchidos",
			base: BaseOptions{
				KubeContext: "test-context",
				ReleaseName: "test-release",
				Namespace:   "test-namespace",
				AutoApprove: true,
				Timeout:     10 * time.Minute,
			},
			expected: map[string]any{
				"kubeContext": "test-context",
				"releaseName": "test-release",
				"namespace":   "test-namespace",
				"autoApprove": true,
				"timeout":     10 * time.Minute,
			},
		},
		{
			name: "BaseOptions com campos mínimos",
			base: BaseOptions{
				KubeContext: "minimal-context",
				ReleaseName: "minimal-release",
			},
			expected: map[string]any{
				"kubeContext": "minimal-context",
				"releaseName": "minimal-release",
				"namespace":   "",
				"autoApprove": false,
				"timeout":     time.Duration(0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Testar métodos getters
			if got := tt.base.GetKubeContext(); got != tt.expected["kubeContext"] {
				t.Errorf("GetKubeContext() = %v, want %v", got, tt.expected["kubeContext"])
			}

			if got := tt.base.GetReleaseName(); got != tt.expected["releaseName"] {
				t.Errorf("GetReleaseName() = %v, want %v", got, tt.expected["releaseName"])
			}

			if got := tt.base.GetNamespace(); got != tt.expected["namespace"] {
				t.Errorf("GetNamespace() = %v, want %v", got, tt.expected["namespace"])
			}

			if got := tt.base.GetAutoApprove(); got != tt.expected["autoApprove"] {
				t.Errorf("GetAutoApprove() = %v, want %v", got, tt.expected["autoApprove"])
			}

			expectedTimeout := tt.expected["timeout"].(time.Duration)
			if got := tt.base.GetTimeout(); got != expectedTimeout {
				t.Errorf("GetTimeout() = %v, want %v", got, expectedTimeout)
			}
		})
	}
}

func TestInstallOptions(t *testing.T) {
	base := BaseOptions{
		KubeContext: "test-context",
		ReleaseName: "test-release",
		Namespace:   "test-namespace",
		AutoApprove: true,
		Timeout:     5 * time.Minute,
	}

	installOpts := InstallOptions{
		BaseOptions:    base,
		Environment:    "stg",
		MaxConcurrency: 3,
	}

	// Testar se os campos da base estão acessíveis
	if installOpts.GetKubeContext() != "test-context" {
		t.Errorf("InstallOptions.GetKubeContext() = %v, want %v", installOpts.GetKubeContext(), "test-context")
	}

	// Testar campos específicos do install
	if installOpts.Environment != "stg" {
		t.Errorf("InstallOptions.Environment = %v, want %v", installOpts.Environment, "stg")
	}

	if installOpts.MaxConcurrency != 3 {
		t.Errorf("InstallOptions.MaxConcurrency = %v, want %v", installOpts.MaxConcurrency, 3)
	}
}

func TestRollbackOptions(t *testing.T) {
	base := BaseOptions{
		KubeContext: "test-context",
		ReleaseName: "test-release",
		Namespace:   "test-namespace",
		AutoApprove: false,
		Timeout:     2 * time.Minute,
	}

	rollbackOpts := RollbackOptions{
		BaseOptions: base,
		Revision:    2,
	}

	// Testar se os campos da base estão acessíveis
	if rollbackOpts.GetReleaseName() != "test-release" {
		t.Errorf("RollbackOptions.GetReleaseName() = %v, want %v", rollbackOpts.GetReleaseName(), "test-release")
	}

	// Testar campo específico do rollback
	if rollbackOpts.Revision != 2 {
		t.Errorf("RollbackOptions.Revision = %v, want %v", rollbackOpts.Revision, 2)
	}
}

func TestUninstallOptions(t *testing.T) {
	base := BaseOptions{
		KubeContext: "test-context",
		ReleaseName: "test-release",
		Namespace:   "test-namespace",
		AutoApprove: true,
		Timeout:     1 * time.Minute,
	}

	uninstallOpts := UninstallOptions{
		BaseOptions: base,
	}

	// Testar se os campos da base estão acessíveis
	if uninstallOpts.GetNamespace() != "test-namespace" {
		t.Errorf("UninstallOptions.GetNamespace() = %v, want %v", uninstallOpts.GetNamespace(), "test-namespace")
	}

	if uninstallOpts.GetAutoApprove() != true {
		t.Errorf("UninstallOptions.GetAutoApprove() = %v, want %v", uninstallOpts.GetAutoApprove(), true)
	}
}

func TestStatusOptions(t *testing.T) {
	base := BaseOptions{
		KubeContext: "test-context",
		ReleaseName: "test-release",
		Namespace:   "test-namespace",
		AutoApprove: false,
		Timeout:     30 * time.Second,
	}

	statusOpts := StatusOptions{
		BaseOptions: base,
	}

	// Testar se os campos da base estão acessíveis
	if statusOpts.GetTimeout() != 30*time.Second {
		t.Errorf("StatusOptions.GetTimeout() = %v, want %v", statusOpts.GetTimeout(), 30*time.Second)
	}
}
