package utils

import (
	"os"
	"testing"
)

func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"a < b", 1, 2, 1},
		{"a > b", 5, 3, 3},
		{"a = b", 4, 4, 4},
		{"negative values", -10, -5, -10},
		{"zero and positive", 0, 10, 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Min(test.a, test.b)
			if result != test.expected {
				t.Errorf("Min(%d, %d) = %d; expected %d", test.a, test.b, result, test.expected)
			}
		})
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"a < b", 1, 2, 2},
		{"a > b", 5, 3, 5},
		{"a = b", 4, 4, 4},
		{"negative values", -10, -5, -5},
		{"zero and positive", 0, 10, 10},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Max(test.a, test.b)
			if result != test.expected {
				t.Errorf("Max(%d, %d) = %d; expected %d", test.a, test.b, result, test.expected)
			}
		})
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		min      int
		max      int
		expected int
	}{
		{"value within range", 5, 1, 10, 5},
		{"value below min", 0, 1, 10, 1},
		{"value above max", 11, 1, 10, 10},
		{"value equal to min", 1, 1, 10, 1},
		{"value equal to max", 10, 1, 10, 10},
		{"negative range", -5, -10, -1, -5},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Clamp(test.value, test.min, test.max)
			if result != test.expected {
				t.Errorf("Clamp(%d, %d, %d) = %d; expected %d",
					test.value, test.min, test.max, result, test.expected)
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	// 環境変数をテスト終了時に戻す準備
	oldEnv := os.Getenv("TEST_VAR")
	defer os.Setenv("TEST_VAR", oldEnv)

	tests := []struct {
		name         string
		envValue     string
		envIsSet     bool
		key          string
		defaultValue string
		expected     string
	}{
		{"env var is set", "value1", true, "TEST_VAR", "default", "value1"},
		{"env var not set", "", false, "TEST_VAR", "default", "default"},
		{"env var is empty", "", true, "TEST_VAR", "default", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.envIsSet {
				os.Setenv("TEST_VAR", test.envValue)
			} else {
				os.Unsetenv("TEST_VAR")
			}

			result := GetEnvOrDefault(test.key, test.defaultValue)
			if result != test.expected {
				t.Errorf("GetEnvOrDefault(%s, %s) = %s; expected %s",
					test.key, test.defaultValue, result, test.expected)
			}
		})
	}
}

func TestAbsInt(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{"positive", 5, 5},
		{"negative", -5, 5},
		{"zero", 0, 0},
		{"max int32", 2147483647, 2147483647},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := AbsInt(test.value)
			if result != test.expected {
				t.Errorf("AbsInt(%d) = %d; expected %d", test.value, result, test.expected)
			}
		})
	}
}

func TestClampFloat64(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		min      float64
		max      float64
		expected float64
	}{
		{"value within range", 5.5, 1.0, 10.0, 5.5},
		{"value below min", 0.5, 1.0, 10.0, 1.0},
		{"value above max", 11.5, 1.0, 10.0, 10.0},
		{"value equal to min", 1.0, 1.0, 10.0, 1.0},
		{"value equal to max", 10.0, 1.0, 10.0, 10.0},
		{"negative range", -5.5, -10.0, -1.0, -5.5},
		{"float precision", 0.00001, 0.0, 0.0001, 0.00001},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := ClampFloat64(test.value, test.min, test.max)
			if result != test.expected {
				t.Errorf("ClampFloat64(%f, %f, %f) = %f; expected %f",
					test.value, test.min, test.max, result, test.expected)
			}
		})
	}
}
