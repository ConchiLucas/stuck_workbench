package seed

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnglishCatalogHasReviewedMeaningForEveryWord(t *testing.T) {
	modules := englishModules()
	require.Len(t, modules, 20)

	count := 0
	for _, module := range modules {
		require.Len(t, module.Kps, 10, module.Code)
		for _, kp := range module.Kps {
			var payload englishPayload
			require.NoError(t, json.Unmarshal([]byte(kp.Payload), &payload), kp.Title)
			require.NotEmpty(t, payload.MeaningZh, kp.Title)
			count++
		}
	}
	require.Equal(t, 200, count)
}
