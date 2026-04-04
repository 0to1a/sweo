package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSummarizeCI(t *testing.T) {
	tests := []struct {
		name   string
		checks []CICheck
		want   CISummary
	}{
		{
			name:   "empty",
			checks: nil,
			want:   CINone,
		},
		{
			name: "all passing",
			checks: []CICheck{
				{Name: "build", State: "SUCCESS"},
				{Name: "test", State: "SUCCESS"},
			},
			want: CIPassing,
		},
		{
			name: "one failing",
			checks: []CICheck{
				{Name: "build", State: "SUCCESS"},
				{Name: "test", State: "FAILURE"},
			},
			want: CIFailing,
		},
		{
			name: "one pending",
			checks: []CICheck{
				{Name: "build", State: "SUCCESS"},
				{Name: "test", State: "PENDING"},
			},
			want: CIPending,
		},
		{
			name: "failing takes priority over pending",
			checks: []CICheck{
				{Name: "build", State: "FAILURE"},
				{Name: "test", State: "PENDING"},
			},
			want: CIFailing,
		},
		{
			name: "in progress counts as pending",
			checks: []CICheck{
				{Name: "build", State: "IN_PROGRESS"},
			},
			want: CIPending,
		},
		{
			name: "skipped only counts as passing",
			checks: []CICheck{
				{Name: "lint", State: "SKIPPED"},
			},
			want: CIPassing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SummarizeCI(tt.checks)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeCheckState(t *testing.T) {
	assert.Equal(t, "passed", normalizeCheckState("SUCCESS"))
	assert.Equal(t, "failed", normalizeCheckState("FAILURE"))
	assert.Equal(t, "failed", normalizeCheckState("TIMED_OUT"))
	assert.Equal(t, "running", normalizeCheckState("IN_PROGRESS"))
	assert.Equal(t, "pending", normalizeCheckState("PENDING"))
	assert.Equal(t, "pending", normalizeCheckState("QUEUED"))
	assert.Equal(t, "skipped", normalizeCheckState("SKIPPED"))
	assert.Equal(t, "skipped", normalizeCheckState("NEUTRAL"))
	assert.Equal(t, "skipped", normalizeCheckState("unknown"))
}

func TestBranchName(t *testing.T) {
	assert.Equal(t, "feat/issue-42", BranchName(42))
	assert.Equal(t, "feat/issue-1", BranchName(1))
}
