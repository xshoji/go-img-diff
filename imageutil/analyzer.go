package imageutil

import (
	"fmt"
	"image"
	"runtime"
	"sync"
	"time"

	"github.com/xshoji/go-img-diff/config"
	"github.com/xshoji/go-img-diff/utils"
)

// DiffAnalyzer 画像差分の解析とビジュアル化を行う構造体
type DiffAnalyzer struct {
	cfg *config.AppConfig
}

// NewDiffAnalyzer 設定をもとに新しいDiffAnalyzerインスタンスを作成
func NewDiffAnalyzer(cfg *config.AppConfig) *DiffAnalyzer {
	return &DiffAnalyzer{
		cfg: cfg,
	}
}

// FindBestAlignment は2つの画像間の最適なオフセット（位置合わせ）を検出する
// 類似度が最も高くなるオフセットを総当たりで探索する
func (da *DiffAnalyzer) FindBestAlignment(imgA, imgB image.Image) (int, int) {
	fmt.Printf("[INFO] Starting alignment detection...\n")
	startTime := time.Now()

	// 使用するCPUコア数を設定
	runtime.GOMAXPROCS(da.cfg.NumCPU)
	fmt.Printf("[INFO] Using %d CPU cores for parallel processing\n", da.cfg.NumCPU)

	// 高速モードが有効な場合は段階的サンプリングを使用
	if da.cfg.FastMode {
		fmt.Printf("[INFO] Fast mode enabled: using progressive sampling\n")
		return da.findBestAlignmentWithProgressiveSampling(imgA, imgB)
	}

	// 以下は通常モード（単一サンプリングレート）での処理
	if da.cfg.SamplingRate > 1 {
		fmt.Printf("[INFO] Using sampling rate 1/%d (analyzing %d%% of pixels)\n",
			da.cfg.SamplingRate, 100/da.cfg.SamplingRate)
	}

	// 探索する総オフセット数を計算
	maxOffset := da.cfg.MaxOffset
	totalOffsets := (2*maxOffset + 1) * (2*maxOffset + 1)
	fmt.Printf("[INFO] Searching for best alignment (max offset: %d, total offsets to check: %d)...\n",
		maxOffset, totalOffsets)

	// 処理を並列化するためのチャネル
	type OffsetScore struct {
		offsetX, offsetY int
		score            float64
	}

	results := make(chan OffsetScore, totalOffsets)

	// ワーカー数を決定（CPUコア数を超えないように）
	numWorkers := utils.Min(da.cfg.NumCPU, totalOffsets)

	// オフセットの組み合わせをすべて生成
	offsets := make([]struct{ x, y int }, 0, totalOffsets)
	for y := -maxOffset; y <= maxOffset; y++ {
		for x := -maxOffset; x <= maxOffset; x++ {
			offsets = append(offsets, struct{ x, y int }{x, y})
		}
	}

	// 並列処理用のワーカープールを作成
	var wg sync.WaitGroup
	offsetCh := make(chan struct{ x, y int }, totalOffsets)

	// ワーカーを起動
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for offset := range offsetCh {
				// 特定のオフセットでの類似度を計算
				score := da.calculateSimilarityScore(imgA, imgB, offset.x, offset.y)
				results <- OffsetScore{offset.x, offset.y, score}
			}
		}()
	}

	// チャネルにオフセットを送信
	for _, offset := range offsets {
		offsetCh <- offset
	}
	close(offsetCh)

	// すべてのワーカーが完了するまで待機
	go func() {
		wg.Wait()
		close(results)
	}()

	// 最も類似度の高いオフセットを選択
	bestOffsetX, bestOffsetY := 0, 0
	bestScore := 0.0

	// 進捗表示用の変数
	processed := 0
	lastPercentReported := -1
	progressStep := da.cfg.ProgressStep

	fmt.Printf("[INFO] Calculating similarity scores for all possible offsets...\n")

	for result := range results {
		processed++

		if result.score > bestScore {
			bestScore = result.score
			bestOffsetX = result.offsetX
			bestOffsetY = result.offsetY
		}

		// 進捗状況を表示
		percent := (processed * 100) / totalOffsets
		if percent != lastPercentReported && percent%progressStep == 0 && percent > lastPercentReported {
			elapsed := time.Since(startTime)
			remaining := float64(0)
			if processed > 0 {
				remaining = float64(elapsed) * float64(totalOffsets-processed) / float64(processed)
			}
			fmt.Printf("[INFO] Alignment search progress: %d%% - Elapsed: %.1fs, Est. remaining: %.1fs\n",
				percent, elapsed.Seconds(), remaining/float64(time.Second))
			lastPercentReported = percent
		}
	}

	elapsed := time.Since(startTime)
	fmt.Printf("[INFO] Best alignment found: offset=(%d, %d) with score=%.4f (%.2fs elapsed)\n",
		bestOffsetX, bestOffsetY, bestScore, elapsed.Seconds())

	return bestOffsetX, bestOffsetY
}

// findBestAlignmentWithProgressiveSampling は段階的サンプリングを使用して最適な位置合わせを検出する
// 最初に粗いサンプリングでおおよその位置を特定し、徐々に精度を上げていく
func (da *DiffAnalyzer) findBestAlignmentWithProgressiveSampling(imgA, imgB image.Image) (int, int) {
	fmt.Printf("[INFO] Using progressive sampling for alignment detection\n")
	startTime := time.Now()

	// 段階的なサンプリングレートを定義（大きい値から小さい値へ）
	samplingStages := []int{8, 4, 2}
	if da.cfg.SamplingRate > 1 {
		// ユーザー指定のサンプリングレートが最終段階
		samplingStages = append(samplingStages, da.cfg.SamplingRate)
	} else {
		// 最終的に全ピクセル比較
		samplingStages = append(samplingStages, 1)
	}

	// 段階ごとに探索範囲を狭めていく
	maxOffset := da.cfg.MaxOffset
	bestOffsetX, bestOffsetY := 0, 0

	for stageIdx, samplingRate := range samplingStages {
		stageStartTime := time.Now()
		fmt.Printf("[INFO] Progressive sampling stage %d/%d: sampling rate=1/%d, max offset=%d\n",
			stageIdx+1, len(samplingStages), samplingRate, maxOffset)

		// 現在のサンプリングレートを一時的に設定
		origSamplingRate := da.cfg.SamplingRate
		da.cfg.SamplingRate = samplingRate

		// 探索範囲内で最適なオフセットを検索
		searchMaxOffset := maxOffset
		if stageIdx > 0 {
			// 2段階目以降は直前の最適オフセット周辺に探索範囲を絞る
			searchMaxOffset = utils.Max(2, maxOffset/(2*(stageIdx)))
		}

		// 現在のステージでの最適オフセットを検索
		stageOffsetX, stageOffsetY, score := da.searchBestOffsetInRange(
			imgA, imgB,
			bestOffsetX-searchMaxOffset, bestOffsetX+searchMaxOffset,
			bestOffsetY-searchMaxOffset, bestOffsetY+searchMaxOffset)

		bestOffsetX = stageOffsetX
		bestOffsetY = stageOffsetY

		stageDuration := time.Since(stageStartTime)
		fmt.Printf("[INFO] Stage %d completed: best offset=(%d, %d), score=%.4f, time=%.2fs\n",
			stageIdx+1, bestOffsetX, bestOffsetY, score, stageDuration.Seconds())

		// 元のサンプリングレートを復元
		da.cfg.SamplingRate = origSamplingRate

		// 探索範囲を縮小
		maxOffset = searchMaxOffset
	}

	elapsed := time.Since(startTime)
	fmt.Printf("[INFO] Progressive alignment search completed in %.2fs\n", elapsed.Seconds())

	return bestOffsetX, bestOffsetY
}

// searchBestOffsetInRange は指定された範囲内で最適なオフセットを検索する
// 並列処理を行いつつ、指定範囲内の全オフセットを評価
func (da *DiffAnalyzer) searchBestOffsetInRange(
	imgA, imgB image.Image,
	minX, maxX, minY, maxY int) (bestX, bestY int, bestScore float64) {

	// 範囲内のすべてのオフセットを生成
	offsets := make([]struct{ x, y int }, 0)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			offsets = append(offsets, struct{ x, y int }{x, y})
		}
	}

	totalOffsets := len(offsets)
	fmt.Printf("[INFO] Searching %d offsets in range X:[%d,%d], Y:[%d,%d]\n",
		totalOffsets, minX, maxX, minY, maxY)

	// 並列処理用の準備
	type OffsetScore struct {
		offsetX, offsetY int
		score            float64
	}
	results := make(chan OffsetScore, totalOffsets)

	// ワーカー数を決定
	numWorkers := utils.Min(da.cfg.NumCPU, totalOffsets)

	// 並列処理用のワーカープールを作成
	var wg sync.WaitGroup
	offsetCh := make(chan struct{ x, y int }, totalOffsets)

	// ワーカーを起動
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for offset := range offsetCh {
				score := da.calculateSimilarityScore(imgA, imgB, offset.x, offset.y)
				results <- OffsetScore{offset.x, offset.y, score}
			}
		}()
	}

	// チャネルにオフセットを送信
	for _, offset := range offsets {
		offsetCh <- offset
	}
	close(offsetCh)

	// すべてのワーカーが完了するまで待機
	go func() {
		wg.Wait()
		close(results)
	}()

	// 最も類似度の高いオフセットを選択
	bestX, bestY = 0, 0
	bestScore = 0.0

	for result := range results {
		if result.score > bestScore {
			bestScore = result.score
			bestX = result.offsetX
			bestY = result.offsetY
		}
	}

	return bestX, bestY, bestScore
}

// 以下の関数は、既存のコードをそのまま使用します
// - calculateSimilarityScore
// - colorDifference
// - GenerateDiffImage
// - drawRedBorders
