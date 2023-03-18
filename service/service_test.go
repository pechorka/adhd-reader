package service

import (
	"testing"

	"github.com/pechorka/adhd-reader/storage"
	"github.com/stretchr/testify/require"
)

func Test_calculateCompletionPercent(t *testing.T) {
	tests := []struct {
		name         string
		totalChunks  int64
		currentChunk int64
		wantPercent  int
	}{
		{name: "total 0", totalChunks: 0, currentChunk: 0, wantPercent: 0},
		{name: "current 0", totalChunks: 11, currentChunk: 0, wantPercent: 0},
		{name: "current 1", totalChunks: 11, currentChunk: 1, wantPercent: 10},
		{name: "current 2", totalChunks: 11, currentChunk: 2, wantPercent: 20},
		{name: "current 3", totalChunks: 11, currentChunk: 3, wantPercent: 30},
		{name: "current 4", totalChunks: 11, currentChunk: 4, wantPercent: 40},
		{name: "current 5", totalChunks: 11, currentChunk: 5, wantPercent: 50},
		{name: "current 6", totalChunks: 11, currentChunk: 6, wantPercent: 60},
		{name: "current 7", totalChunks: 11, currentChunk: 7, wantPercent: 70},
		{name: "current 8", totalChunks: 11, currentChunk: 8, wantPercent: 80},
		{name: "current 9", totalChunks: 11, currentChunk: 9, wantPercent: 90},
		{name: "current 10", totalChunks: 11, currentChunk: 10, wantPercent: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateCompletionPercent(storage.TextWithChunkInfo{
				TotalChunks:  tt.totalChunks,
				CurrentChunk: tt.currentChunk,
			})
			require.Equal(t, tt.wantPercent, got)
		})
	}
}
