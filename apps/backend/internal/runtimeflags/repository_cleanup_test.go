package runtimeflags

import (
	"testing"

	"github.com/kandev/kandev/internal/common/config"
	"github.com/kandev/kandev/internal/profiles"
	"github.com/stretchr/testify/require"
)

func TestRepositoryCleanupFlagContract(t *testing.T) {
	const key = "features.repositoryCleanup"
	def, ok := DefinitionByKey(key)
	require.True(t, ok, "repository cleanup registration must exist")
	require.Equal(t, "KANDEV_FEATURES_REPOSITORY_CLEANUP", def.EnvVar)
	require.Equal(t, KindFeature, def.Kind)
	require.True(t, def.RestartRequired)
	require.True(t, def.Mutable)
	require.NotEmpty(t, def.RiskDescription)

	defaults, err := profiles.FeatureFlagDefaults()
	require.NoError(t, err)
	require.Equal(t, "false", defaults["repository_cleanup"])

	cfg := &config.Config{}
	require.False(t, ValuesFromConfig(cfg)[key])
	ApplyStatesToConfig(cfg, []RuntimeFlagState{{Key: key, EffectiveValue: true}})
	require.True(t, cfg.Features.RepositoryCleanup)
	require.True(t, ValuesFromConfig(cfg)[key])
}
