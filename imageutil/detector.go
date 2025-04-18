package imageutil

import (
	"fmt"
	"image"
	"image/draw"
	"time"

	"github.com/xshoji/go-img-diff/utils"
)

// GenerateDiffImage は位置ずれを考慮した差分画像を生成する
// imgB（second画像）をベースにして、その上に差分を赤枠で強調表示する
func (da *DiffAnalyzer) GenerateDiffImage(imgA, imgB image.Image, offsetX, offsetY int) image.Image {
	fmt.Printf("[INFO] Generating diff image with offset (%d, %d)...\n", offsetX, offsetY)
	startTime := time.Now()

	boundsB := imgB.Bounds()

	// 出力画像のサイズを決定
	width := utils.Max(imgA.Bounds().Dx(), boundsB.Dx())
	height := utils.Max(imgA.Bounds().Dy(), boundsB.Dy())

	fmt.Printf("[INFO] Creating result image (%dx%d)...\n", width, height)

	// 新しい画像を作成
	result := image.NewRGBA(image.Rect(0, 0, width, height))

	// まずimgB（second画像）を描画して、これをベースとする
	fmt.Printf("[INFO] Using second image as base for output...\n")
	draw.Draw(result, result.Bounds(), imgB, boundsB.Min, draw.Src)

	// 差分領域を検出
	fmt.Printf("[INFO] Detecting diff regions with global offset...\n")
	diffRegions := da.detectDiffRegions(imgA, imgB, offsetX, offsetY)
	fmt.Printf("[INFO] Found %d diff regions\n", len(diffRegions))

	// 透過表示が有効な場合はメッセージを表示
	if da.cfg.ShowTransparentOverlay {
		if da.cfg.UseTint {
			tint := da.cfg.OverlayTint
			fmt.Printf("[INFO] Applying tinted overlay (R:%d G:%d B:%d) with transparency: %.1f%%, tint strength: %.1f%%, tint transparency: %.1f%%\n",
				tint.R, tint.G, tint.B,
				da.cfg.OverlayTransparency*100,
				da.cfg.TintStrength*100,
				da.cfg.TintTransparency*100)
		} else {
			fmt.Printf("[INFO] Applying transparent overlay with transparency: %.1f%%\n",
				da.cfg.OverlayTransparency*100)
		}
	}

	// 差分領域を赤枠で囲む（および透過表示する）
	fmt.Printf("[INFO] Drawing red borders around diff regions...\n")
	da.drawRedBorders(result, diffRegions, imgA, offsetX, offsetY) // オフセット情報を渡す

	elapsed := time.Since(startTime)
	fmt.Printf("[INFO] Diff image generation completed in %.2f seconds\n", elapsed.Seconds())

	return result
}

// abs は整数の絶対値を返す
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// detectDiffRegions は2つの画像の差分領域を検出する
func (da *DiffAnalyzer) detectDiffRegions(imgA, imgB image.Image, offsetX, offsetY int) []image.Rectangle {
	startTime := time.Now()
	boundsA := imgA.Bounds()
	boundsB := imgB.Bounds()

	fmt.Printf("[INFO] Creating diff map for dimensions %dx%d...\n", boundsB.Dx(), boundsB.Dy())

	// 差分を格納するためのマップ
	diffMap := make([][]bool, boundsB.Dy())
	for i := range diffMap {
		diffMap[i] = make([]bool, boundsB.Dx())
	}

	// サンプリングレートを設定
	samplingRate := da.cfg.SamplingRate
	if samplingRate > 1 {
		fmt.Printf("[INFO] Using sampling rate 1/%d for diff detection\n", samplingRate)
	}

	// 進捗表示用の変数
	totalRows := boundsB.Dy()
	lastPercentReported := -1
	progressStep := da.cfg.ProgressStep // 進捗表示の粒度

	fmt.Printf("[INFO] Comparing pixels to detect differences...\n")

	// 差分を検出
	for y := 0; y < boundsB.Dy(); y += samplingRate {
		for x := 0; x < boundsB.Dx(); x += samplingRate {
			// A画像の対応座標
			xA := x - offsetX
			yA := y - offsetY

			// A画像の範囲外ならスキップ
			if xA < 0 || xA >= boundsA.Dx() || yA < 0 || yA >= boundsA.Dy() {
				// サンプリング領域内のすべてのピクセルを差分として扱う
				for sy := 0; sy < samplingRate && y+sy < boundsB.Dy(); sy++ {
					for sx := 0; sx < samplingRate && x+sx < boundsB.Dx(); sx++ {
						diffMap[y+sy][x+sx] = true
					}
				}
				continue
			}

			// 色の差が閾値を超えているか確認
			isDifferent := da.colorDifference(
				imgA.At(boundsA.Min.X+xA, boundsA.Min.Y+yA),
				imgB.At(boundsB.Min.X+x, boundsB.Min.Y+y),
			) > float64(da.cfg.Threshold)

			// サンプリング領域内のすべてのピクセルに適用
			if isDifferent {
				for sy := 0; sy < samplingRate && y+sy < boundsB.Dy(); sy++ {
					for sx := 0; sx < samplingRate && x+sx < boundsB.Dx(); sx++ {
						diffMap[y+sy][x+sx] = true
					}
				}
			}
		}

		// 進捗を表示
		percentComplete := (y * 100) / totalRows
		if percentComplete != lastPercentReported && percentComplete%progressStep == 0 && percentComplete > lastPercentReported {
			elapsed := time.Since(startTime)
			remainingEstimate := float64(0)
			if y > 0 {
				remainingEstimate = float64(elapsed) * float64(totalRows-y) / float64(y)
			}

			fmt.Printf("[INFO] Diff detection progress: %d%% - Elapsed: %.1fs, Est. remaining: %.1fs\n",
				percentComplete, elapsed.Seconds(), remainingEstimate/float64(time.Second))
			lastPercentReported = percentComplete
		}
	}

	fmt.Printf("[INFO] Diff detection complete. Grouping diff regions...\n")

	// 差分領域をまとめる
	regions := da.groupDiffRegions(diffMap, boundsB)

	elapsed := time.Since(startTime)
	fmt.Printf("[INFO] Diff region detection completed in %.2f seconds\n", elapsed.Seconds())

	return regions
}

// groupDiffRegions は差分ピクセルを矩形領域にグループ化する
func (da *DiffAnalyzer) groupDiffRegions(diffMap [][]bool, bounds image.Rectangle) []image.Rectangle {
	var regions []image.Rectangle
	visited := make([][]bool, len(diffMap))
	for i := range visited {
		visited[i] = make([]bool, len(diffMap[0]))
	}

	// 差分ピクセルを走査
	for y := 0; y < len(diffMap); y++ {
		for x := 0; x < len(diffMap[0]); x++ {
			if diffMap[y][x] && !visited[y][x] {
				// 新しい差分領域を見つけた
				minX, minY := x, y
				maxX, maxY := x, y

				// 周囲の差分ピクセルを探索（より広い範囲で探索）
				for dy := -10; dy <= 10; dy++ {
					for dx := -10; dx <= 10; dx++ {
						nx, ny := x+dx, y+dy
						if nx >= 0 && nx < len(diffMap[0]) && ny >= 0 && ny < len(diffMap) {
							if diffMap[ny][nx] {
								visited[ny][nx] = true
								minX = utils.Min(minX, nx)
								minY = utils.Min(minY, ny)
								maxX = utils.Max(maxX, nx)
								maxY = utils.Max(maxY, ny)
							}
						}
					}
				}

				// 領域に余白を追加（より大きな余白）
				padding := 5 // 2から5に増加
				minX = utils.Max(0, minX-padding)
				minY = utils.Max(0, minY-padding)
				maxX = utils.Min(len(diffMap[0])-1, maxX+padding)
				maxY = utils.Min(len(diffMap)-1, maxY+padding)

				regions = append(regions, image.Rect(
					bounds.Min.X+minX,
					bounds.Min.Y+minY,
					bounds.Min.X+maxX+1,
					bounds.Min.Y+maxY+1,
				))
			}
		}
	}

	// 非常に小さい領域は除外または拡大する
	var filteredRegions []image.Rectangle
	for _, rect := range regions {
		width := rect.Max.X - rect.Min.X
		height := rect.Max.Y - rect.Min.Y

		// 小さすぎる領域は少し大きくする
		if width < 20 || height < 20 {
			// 中心点を計算
			centerX := (rect.Min.X + rect.Max.X) / 2
			centerY := (rect.Min.Y + rect.Max.Y) / 2

			// 最小サイズを確保
			minSize := 20
			newWidth := utils.Max(width, minSize)
			newHeight := utils.Max(height, minSize)

			// 新しい矩形を作成
			newRect := image.Rect(
				utils.Max(bounds.Min.X, centerX-newWidth/2),
				utils.Max(bounds.Min.Y, centerY-newHeight/2),
				utils.Min(bounds.Max.X, centerX+newWidth/2),
				utils.Min(bounds.Max.Y, centerY+newHeight/2),
			)
			filteredRegions = append(filteredRegions, newRect)
		} else {
			filteredRegions = append(filteredRegions, rect)
		}
	}

	// 重なり合う矩形を連結する
	mergedRegions := mergeOverlappingRectangles(filteredRegions)

	// 多くの四角が連結された場合は、その処理結果を表示
	if len(mergedRegions) < len(filteredRegions) {
		fmt.Printf("[INFO] Merged %d diff regions into %d combined regions\n",
			len(filteredRegions), len(mergedRegions))
	}

	return mergedRegions
}
