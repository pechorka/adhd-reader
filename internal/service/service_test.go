package service

import (
	"reflect"
	"testing"

	"github.com/pechorka/adhd-reader/internal/storage"
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
			got := calculateCompletionPercent(
				storage.TextWithChunkInfo{
					TotalChunks:  tt.totalChunks,
					CurrentChunk: tt.currentChunk,
				})
			require.Equal(t, tt.wantPercent, got)
		})
	}
}

func TestGetLevelByExperience(t *testing.T) {
	type args struct {
		experience int64
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{name: "0", args: args{experience: 0}, want: 0},
		{name: "1", args: args{experience: 1}, want: 0},
		{name: "99", args: args{experience: 99}, want: 0},
		{name: "100", args: args{experience: 100}, want: 1},
		{name: "101", args: args{experience: 101}, want: 1},
		{name: "210", args: args{experience: 210}, want: 2},
		{name: "331", args: args{experience: 331}, want: 3},
		{name: "464", args: args{experience: 464}, want: 4},
		{name: "610", args: args{experience: 610}, want: 5},
		{name: "770", args: args{experience: 770}, want: 6},
		{name: "1000", args: args{experience: 1000}, want: 7},
		{name: "1500", args: args{experience: 1500}, want: 9},
		{name: "5629", args: args{experience: 5629}, want: 19},
		{name: "5630", args: args{experience: 5630}, want: 20},
		{name: "5631", args: args{experience: 5631}, want: 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectLevelByExperience(tt.args.experience); got != tt.want {
				t.Errorf("GetLevelByExperience() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_calculateExperienceGainByChunkSize(t *testing.T) {
	type args struct {
		chunkSize int64
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{name: "0", args: args{chunkSize: 0}, want: 2},
		{name: "1", args: args{chunkSize: 1}, want: 1},
		{name: "250", args: args{chunkSize: 250}, want: 1},
		{name: "500", args: args{chunkSize: 500}, want: 2},
		{name: "501", args: args{chunkSize: 501}, want: 2},
		{name: "501", args: args{chunkSize: 751}, want: 3},
		{name: "1000", args: args{chunkSize: 1000}, want: 4},
		{name: "1250", args: args{chunkSize: 1250}, want: 5},
		{name: "1500", args: args{chunkSize: 1500}, want: 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateExperienceGainByChunkSize(tt.args.chunkSize); got != tt.want {
				t.Errorf("calculateExperienceGainByChunkSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLevelUpStatDistribution(t *testing.T) {
	type args struct {
		level int64
	}
	tests := []struct {
		name string
		args args
		want Stat
	}{
		{name: "0", args: args{level: 0}, want: Stat{Accuracy: 0, Attention: 0, TimeManagement: 0, Charizma: 0, Luck: 0}},
		{name: "1", args: args{level: 1}, want: Stat{Accuracy: 1, Attention: 0, TimeManagement: 0, Charizma: 0, Luck: 0}},
		{name: "2", args: args{level: 2}, want: Stat{Accuracy: 1, Attention: 1, TimeManagement: 0, Charizma: 0, Luck: 0}},
		{name: "3", args: args{level: 3}, want: Stat{Accuracy: 2, Attention: 1, TimeManagement: 0, Charizma: 0, Luck: 0}},
		{name: "4", args: args{level: 4}, want: Stat{Accuracy: 2, Attention: 2, TimeManagement: 0, Charizma: 0, Luck: 0}},
		{name: "5", args: args{level: 5}, want: Stat{Accuracy: 3, Attention: 2, TimeManagement: 0, Charizma: 0, Luck: 0}},
		{name: "6", args: args{level: 6}, want: Stat{Accuracy: 3, Attention: 3, TimeManagement: 0, Charizma: 0, Luck: 0}},
		{name: "7", args: args{level: 7}, want: Stat{Accuracy: 4, Attention: 3, TimeManagement: 0, Charizma: 0, Luck: 0}},
		{name: "8", args: args{level: 8}, want: Stat{Accuracy: 4, Attention: 3, TimeManagement: 1, Charizma: 0, Luck: 0}},
		{name: "9", args: args{level: 9}, want: Stat{Accuracy: 4, Attention: 3, TimeManagement: 1, Charizma: 0, Luck: 1}},
		{name: "10", args: args{level: 10}, want: Stat{Accuracy: 4, Attention: 3, TimeManagement: 2, Charizma: 0, Luck: 1}},
		{name: "11", args: args{level: 11}, want: Stat{Accuracy: 4, Attention: 4, TimeManagement: 2, Charizma: 0, Luck: 1}},
		{name: "12", args: args{level: 12}, want: Stat{Accuracy: 4, Attention: 4, TimeManagement: 2, Charizma: 0, Luck: 2}},
		{name: "13", args: args{level: 13}, want: Stat{Accuracy: 4, Attention: 4, TimeManagement: 3, Charizma: 0, Luck: 2}},
		{name: "14", args: args{level: 14}, want: Stat{Accuracy: 4, Attention: 4, TimeManagement: 3, Charizma: 0, Luck: 3}},
		{name: "15", args: args{level: 15}, want: Stat{Accuracy: 5, Attention: 4, TimeManagement: 3, Charizma: 0, Luck: 3}},
		{name: "16", args: args{level: 16}, want: Stat{Accuracy: 5, Attention: 4, TimeManagement: 4, Charizma: 0, Luck: 3}},
		{name: "17", args: args{level: 17}, want: Stat{Accuracy: 5, Attention: 4, TimeManagement: 4, Charizma: 0, Luck: 4}},
		{name: "18", args: args{level: 18}, want: Stat{Accuracy: 5, Attention: 5, TimeManagement: 4, Charizma: 0, Luck: 4}},
		{name: "19", args: args{level: 19}, want: Stat{Accuracy: 5, Attention: 5, TimeManagement: 5, Charizma: 0, Luck: 4}},
		{name: "20", args: args{level: 20}, want: Stat{Accuracy: 5, Attention: 5, TimeManagement: 5, Charizma: 0, Luck: 5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LevelUpStatDistribution(tt.args.level); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LevelUpStatDistribution() = %v, want %v", got, tt.want)
			}
		})
	}
}
