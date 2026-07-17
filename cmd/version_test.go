package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildVersionNeverEmpty(t *testing.T) {
	assert.NotEmpty(t, buildVersion())
}

func TestVersionCommandPrintsVersion(t *testing.T) {
	cmd, out := newTestCommandOutput()

	require.NoError(t, versionCmd.RunE(cmd, nil))
	assert.Equal(t, buildVersion()+"\n", out.String())
}
