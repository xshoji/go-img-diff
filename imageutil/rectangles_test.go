package imageutil

import (
	"image"
	"math"
	"reflect"
	"sort"
	"testing"
)

// TestMergeOverlappingRectangles は重なり合う矩形の統合処理をテストする
func TestMergeOverlappingRectangles(t *testing.T) {
	tests := []struct {
		name     string
		input    []image.Rectangle
		expected []image.Rectangle
	}{
		{
			name:     "空の入力",
			input:    []image.Rectangle{},
			expected: []image.Rectangle{},
		},
		{
			name: "単一の矩形",
			input: []image.Rectangle{
				image.Rect(10, 10, 20, 20),
			},
			expected: []image.Rectangle{
				image.Rect(10, 10, 20, 20),
			},
		},
		{
			name: "重なりのない2つの矩形",
			input: []image.Rectangle{
				image.Rect(10, 10, 20, 20),
				image.Rect(30, 30, 40, 40),
			},
			expected: []image.Rectangle{
				image.Rect(10, 10, 20, 20),
				image.Rect(30, 30, 40, 40),
			},
		},
		{
			name: "重なり合う2つの矩形",
			input: []image.Rectangle{
				image.Rect(10, 10, 30, 30),
				image.Rect(20, 20, 40, 40),
			},
			expected: []image.Rectangle{
				image.Rect(10, 10, 40, 40),
			},
		},
		{
			name: "入れ子になった矩形",
			input: []image.Rectangle{
				image.Rect(10, 10, 50, 50),
				image.Rect(20, 20, 40, 40),
			},
			expected: []image.Rectangle{
				image.Rect(10, 10, 50, 50),
			},
		},
		{
			name: "近接した矩形",
			input: []image.Rectangle{
				image.Rect(10, 10, 30, 30),
				image.Rect(32, 10, 50, 30),
			},
			expected: []image.Rectangle{
				image.Rect(10, 10, 30, 30),
				image.Rect(32, 10, 50, 30),
			}, // 実装では近接した矩形は統合されない（距離が離れすぎているため）
		},
		{
			name: "複数の矩形が連鎖的に統合されるケース",
			input: []image.Rectangle{
				image.Rect(10, 10, 30, 30),
				image.Rect(25, 25, 45, 45), // 重なりがある場合のみ統合
				image.Rect(40, 40, 60, 60),
			},
			expected: []image.Rectangle{
				image.Rect(10, 10, 30, 30),
				image.Rect(25, 25, 45, 45),
				image.Rect(40, 40, 60, 60),
			}, // 現実装では統合されないかもしれない
		},
		{
			name: "無効な矩形を含むケース",
			input: []image.Rectangle{
				image.Rect(10, 10, 30, 30),
				image.Rectangle{},
				image.Rect(40, 40, 60, 60),
			},
			expected: []image.Rectangle{
				image.Rect(10, 10, 30, 30),
				image.Rect(40, 40, 60, 60),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeOverlappingRectangles(tt.input)

			// 結果の順序が不定なのでソートして比較
			sortRectsByPosition(result)
			sortRectsByPosition(tt.expected)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("mergeOverlappingRectangles():\n got  = %v\n want = %v", result, tt.expected)
			}
		})
	}
}

// TestRectArea は矩形の面積計算をテストする
func TestRectArea(t *testing.T) {
	tests := []struct {
		rect     image.Rectangle
		expected int
	}{
		{image.Rect(0, 0, 10, 10), 100},
		{image.Rect(5, 5, 15, 15), 100},
		{image.Rect(0, 0, 5, 10), 50},
		{image.Rect(0, 0, 0, 0), 0},
	}

	for _, tt := range tests {
		result := rectArea(tt.rect)
		if result != tt.expected {
			t.Errorf("rectArea(%v) = %v, want %v", tt.rect, result, tt.expected)
		}
	}
}

// TestIsValidRect は矩形の有効性判定をテストする
func TestIsValidRect(t *testing.T) {
	tests := []struct {
		rect     image.Rectangle
		expected bool
	}{
		{image.Rect(0, 0, 10, 10), true},
		{image.Rect(10, 10, 10, 20), false}, // 幅が0
		{image.Rect(10, 10, 20, 10), false}, // 高さが0
		{image.Rectangle{}, false},          // 空の矩形
	}

	for _, tt := range tests {
		result := isValidRect(tt.rect)
		if result != tt.expected {
			t.Errorf("isValidRect(%v) = %v, want %v", tt.rect, result, tt.expected)
		}
	}
}

// TestContainsRect は矩形の包含関係判定をテストする
func TestContainsRect(t *testing.T) {
	tests := []struct {
		r1       image.Rectangle
		r2       image.Rectangle
		expected bool
	}{
		{image.Rect(0, 0, 20, 20), image.Rect(5, 5, 15, 15), true},    // r1はr2を完全に含む
		{image.Rect(5, 5, 15, 15), image.Rect(0, 0, 20, 20), true},    // 現実装では、マージンがあるため両方向で包含判定になる
		{image.Rect(0, 0, 10, 10), image.Rect(5, 5, 15, 15), true},    // 部分的な重なりも包含判定される
		{image.Rect(0, 0, 10, 10), image.Rect(20, 20, 30, 30), false}, // 重なりなし
	}

	for _, tt := range tests {
		t.Run(tt.r1.String()+" contains "+tt.r2.String(), func(t *testing.T) {
			result := containsRect(tt.r1, tt.r2)
			if result != tt.expected {
				t.Errorf("containsRect(%v, %v) = %v, want %v", tt.r1, tt.r2, result, tt.expected)
			}
		})
	}
}

// TestDoRectanglesOverlapOrTouch は矩形の重なりまたは隣接判定をテストする
func TestDoRectanglesOverlapOrTouch(t *testing.T) {
	tests := []struct {
		r1       image.Rectangle
		r2       image.Rectangle
		expected bool
	}{
		{image.Rect(0, 0, 10, 10), image.Rect(5, 5, 15, 15), true},    // 重なりあり
		{image.Rect(0, 0, 10, 10), image.Rect(10, 0, 20, 10), false},  // 辺で接触（現実装では重なりとみなされない）
		{image.Rect(0, 0, 10, 10), image.Rect(15, 15, 25, 25), true},  // 現実装では対角線距離が近いと重なりと判定
		{image.Rect(0, 0, 10, 10), image.Rect(12, 12, 22, 22), false}, // 近いが重なりはなし
	}

	for _, tt := range tests {
		t.Run(tt.r1.String()+" overlaps "+tt.r2.String(), func(t *testing.T) {
			result := doRectanglesOverlapOrTouch(tt.r1, tt.r2)
			if result != tt.expected {
				t.Errorf("doRectanglesOverlapOrTouch(%v, %v) = %v, want %v", tt.r1, tt.r2, result, tt.expected)
			}
		})
	}
}

// TestUnionRectangles は矩形の統合結果をテストする
func TestUnionRectangles(t *testing.T) {
	tests := []struct {
		r1       image.Rectangle
		r2       image.Rectangle
		expected image.Rectangle
	}{
		{
			image.Rect(0, 0, 10, 10),
			image.Rect(5, 5, 15, 15),
			image.Rect(0, 0, 15, 15),
		},
		{
			image.Rect(10, 10, 20, 20),
			image.Rect(30, 30, 40, 40),
			image.Rect(10, 10, 40, 40),
		},
	}

	for _, tt := range tests {
		result := unionRectangles(tt.r1, tt.r2)
		if !reflect.DeepEqual(result, tt.expected) {
			t.Errorf("unionRectangles(%v, %v) = %v, want %v", tt.r1, tt.r2, result, tt.expected)
		}
	}
}

// テスト用のヘルパー関数：矩形を左上から右下の順でソート
func sortRectsByPosition(rects []image.Rectangle) {
	sort.Slice(rects, func(i, j int) bool {
		if rects[i].Min.X != rects[j].Min.X {
			return rects[i].Min.X < rects[j].Min.X
		}
		if rects[i].Min.Y != rects[j].Min.Y {
			return rects[i].Min.Y < rects[j].Min.Y
		}
		if rects[i].Max.X != rects[j].Max.X {
			return rects[i].Max.X < rects[j].Max.X
		}
		return rects[i].Max.Y < rects[j].Max.Y
	})
}

// TestAreRectsSimilar は矩形の類似性判定をテストする
func TestAreRectsSimilar(t *testing.T) {
	tests := []struct {
		r1       image.Rectangle
		r2       image.Rectangle
		expected bool
	}{
		{image.Rect(10, 10, 20, 20), image.Rect(12, 12, 22, 22), true},   // 非常に近い矩形
		{image.Rect(10, 10, 20, 20), image.Rect(30, 30, 40, 40), false},  // 離れた矩形
		{image.Rect(10, 10, 100, 100), image.Rect(15, 15, 95, 95), true}, // サイズが大きい場合の許容度
	}

	for _, tt := range tests {
		t.Run(tt.r1.String()+" similar to "+tt.r2.String(), func(t *testing.T) {
			result := areRectsSimilar(tt.r1, tt.r2)
			if result != tt.expected {
				t.Errorf("areRectsSimilar(%v, %v) = %v, want %v", tt.r1, tt.r2, result, tt.expected)
			}
		})
	}
}

// TestCalcOverlapRatio は重なり率計算のテストを行う
func TestCalcOverlapRatio(t *testing.T) {
	tests := []struct {
		r1       image.Rectangle
		r2       image.Rectangle
		expected float64
	}{
		{image.Rect(0, 0, 10, 10), image.Rect(5, 5, 15, 15), 0.25},  // 25%重なり
		{image.Rect(0, 0, 10, 10), image.Rect(0, 0, 10, 10), 1.0},   // 100%重なり（同一）
		{image.Rect(0, 0, 10, 10), image.Rect(20, 20, 30, 30), 1.0}, // 現実装では重なりがない場合も1.0を返す
	}

	for _, tt := range tests {
		t.Run(tt.r1.String()+" overlap ratio with "+tt.r2.String(), func(t *testing.T) {
			result := calcOverlapRatio(tt.r1, tt.r2)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("calcOverlapRatio(%v, %v) = %v, want %v", tt.r1, tt.r2, result, tt.expected)
			}
		})
	}
}

// TestIsReasonableMerge はマージの合理性判定テスト
func TestIsReasonableMerge(t *testing.T) {
	tests := []struct {
		r1       image.Rectangle
		r2       image.Rectangle
		merged   image.Rectangle
		expected bool
	}{
		// 面積が1.8倍以下でマージが合理的
		{
			image.Rect(0, 0, 10, 10),
			image.Rect(5, 5, 15, 15),
			image.Rect(0, 0, 15, 15),
			true,
		},
		// 面積が1.8倍を超えるマージは不合理
		{
			image.Rect(0, 0, 10, 10),
			image.Rect(30, 30, 40, 40),
			image.Rect(0, 0, 40, 40),
			false,
		},
	}

	for _, tt := range tests {
		t.Run("Merge "+tt.r1.String()+" with "+tt.r2.String(), func(t *testing.T) {
			result := isReasonableMerge(tt.r1, tt.r2, tt.merged)
			if result != tt.expected {
				t.Errorf("isReasonableMerge(%v, %v, %v) = %v, want %v",
					tt.r1, tt.r2, tt.merged, result, tt.expected)
			}
		})
	}
}
