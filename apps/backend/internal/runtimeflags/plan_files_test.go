package runtimeflags

import (
	"testing"

	"github.com/kandev/kandev/internal/common/config"
	"github.com/kandev/kandev/internal/profiles"
	"github.com/stretchr/testify/require"
)

// @covers AC-TASKS-PLAN-FILES-005.6
func TestPlanFilesFlagContract(t *testing.T) {
	const key = "features.planFiles"
	def, ok := DefinitionByKey(key)
	require.True(t, ok, "plan files registration must exist")
	require.Equal(t, "KANDEV_FEATURES_PLAN_FILES", def.EnvVar)
	require.Equal(t, KindFeature, def.Kind)
	require.True(t, def.RestartRequired)
	require.True(t, def.Mutable)
	require.NotEmpty(t, def.RiskDescription)

	defaults, err := profiles.FeatureFlagDefaults()
	require.NoError(t, err)
	require.Equal(t, "false", defaults["plan_files"])

	cfg := &config.Config{}
	require.False(t, ValuesFromConfig(cfg)[key])
	ApplyStatesToConfig(cfg, []RuntimeFlagState{{Key: key, EffectiveValue: true}})
	require.True(t, cfg.Features.PlanFiles)
	require.True(t, ValuesFromConfig(cfg)[key])
}
