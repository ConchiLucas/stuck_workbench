package literacy

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-content-admin/internal/configclient"
)

func TestListImageProvidersPrefersGeminiOverGrok(t *testing.T) {
	ai := configclient.AIConfiguration{
		Providers: []configclient.AIProvider{
			{
				ID: "sub2api-grok-image", Type: "openai-compatible", Enabled: true,
				BaseURL: "http://grok", Model: "grok-imagine-image",
				Capabilities: []string{"IMAGE_GENERATION"},
			},
			{
				ID: "antigravity-gemini-image", Type: "openai-compatible", Enabled: true,
				BaseURL: "http://gemini", Model: "gemini-3.1-flash-image",
				Capabilities: []string{"IMAGE_GENERATION"},
			},
		},
	}
	got := listImageProviders(ai)
	require.Len(t, got, 1)
	require.Equal(t, "antigravity-gemini-image", got[0].ID)
}
