package storage

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_channelToId(t *testing.T) {
	for _, tt := range []struct {
		channel string
		id      string
	}{
		{
			channel: "bus.1.feed",
			id:      "bus.1",
		},
	} {
		t.Run("", func(t *testing.T) {
			require.Equal(t, tt.id, channelToId(tt.channel))
		})
	}
}
