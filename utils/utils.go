package utils

import (
	"os"
)

// Min は2つの整数のうち小さい方を返す
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max は2つの整数のうち大きい方を返す
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Clamp は値を指定範囲内に制限する
func Clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// GetEnvOrDefault は環境変数の値を取得し、設定されていない場合はデフォルト値を返す
func GetEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// AbsInt は整数の絶対値を返す
func AbsInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ClampFloat64 は浮動小数点値を指定範囲内に制限する
func ClampFloat64(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// MaxUint32 は2つのuint32値の大きい方を返す
func MaxUint32(a, b uint32) uint32 {
	if a > b {
		return a
	}
	return b
}

// AbsDiff は2つのuint32値の絶対差を返す
func AbsDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}
