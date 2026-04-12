package daily_reset

import "testing"

func TestIsWinningWordleAttempt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		attempts [][]map[string]interface{}
		expected bool
	}{
		{
			name: "winning attempt exists",
			attempts: [][]map[string]interface{}{
				{
					{"state": "present"},
					{"state": "correct"},
					{"state": "present"},
					{"state": "absent"},
					{"state": "present"},
				},
				{
					{"state": "correct"},
					{"state": "correct"},
					{"state": "correct"},
					{"state": "correct"},
					{"state": "correct"},
				},
			},
			expected: true,
		},
		{
			name: "non-winning attempts only",
			attempts: [][]map[string]interface{}{
				{
					{"state": "present"},
					{"state": "correct"},
					{"state": "present"},
					{"state": "absent"},
					{"state": "present"},
				},
			},
			expected: false,
		},
		{
			name: "invalid length attempt ignored",
			attempts: [][]map[string]interface{}{
				{
					{"state": "correct"},
					{"state": "correct"},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isWinningWordleAttempt(tt.attempts); got != tt.expected {
				t.Fatalf("isWinningWordleAttempt() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsSolvedCrossword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		progress       map[string]interface{}
		crosswordIndex int
		totalWords     int
		expected       bool
	}{
		{
			name: "solved with exact amount",
			progress: map[string]interface{}{
				"2": []interface{}{"СЛОВО1", "СЛОВО2", "СЛОВО3"},
			},
			crosswordIndex: 2,
			totalWords:     3,
			expected:       true,
		},
		{
			name: "duplicates do not overcount",
			progress: map[string]interface{}{
				"0": []interface{}{"СЛОВО1", "слово1", "СЛОВО2"},
			},
			crosswordIndex: 0,
			totalWords:     3,
			expected:       false,
		},
		{
			name: "missing index not solved",
			progress: map[string]interface{}{
				"1": []interface{}{"СЛОВО1"},
			},
			crosswordIndex: 0,
			totalWords:     1,
			expected:       false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isSolvedCrossword(tt.progress, tt.crosswordIndex, tt.totalWords); got != tt.expected {
				t.Fatalf("isSolvedCrossword() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAwardedUsersRoundTrip(t *testing.T) {
	t.Parallel()

	source := map[int]struct{}{
		42:  {},
		7:   {},
		105: {},
	}

	raw := joinAwardedUsers(source)
	parsed := parseAwardedUsers(raw)

	if len(parsed) != len(source) {
		t.Fatalf("expected %d users after roundtrip, got %d", len(source), len(parsed))
	}

	for id := range source {
		if _, ok := parsed[id]; !ok {
			t.Fatalf("missing id %d after roundtrip", id)
		}
	}
}
