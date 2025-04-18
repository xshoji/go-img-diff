package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"image"
	"image/color"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/user/go-img-diff/config"
	"github.com/user/go-img-diff/imageutil"
	"github.com/user/go-img-diff/utils"
)

// 定数定義
const (
	UsageRequiredPrefix = "\u001B[33m(REQ)\u001B[0m "
	TimeFormat          = "2006-01-02 15:04:05.0000 [MST]"
)

// アプリケーション設定とオプション
var (
	// コマンドオプション表示に関する設定
	commandDescription      = "Image difference detection and visualization tool."
	commandOptionFieldWidth = "12" // フィールド幅の推奨値: 一般的に12、ブール値のみの場合は5

	// 必須オプション
	optionImageInput1 = flag.String("i1", "", UsageRequiredPrefix+"First image path")
	optionImageInput2 = flag.String("i2", "", UsageRequiredPrefix+"Second image path")
	optionOutput      = flag.String("o", "", UsageRequiredPrefix+"Output diff image path")

	// 位置ずれ検出のための設定
	optionMaxOffset = flag.Int("m", 10, "Maximum pixel offset to search for alignment")

	// 閾値設定
	optionThreshold = flag.Int("d", 30, "Color difference threshold (0-255)") // 'd' for 'difference threshold'

	// 並列処理のためのCPU数設定
	optionNumCPU = flag.Int("c", runtime.NumCPU(), "Number of CPU cores to use for parallel processing")

	// サンプリング設定
	optionSamplingRate = flag.Int("s", 4, "Sampling rate for pixel comparison (1=all pixels, 2=every other pixel, etc)")

	// 高速モード設定
	// FastMode (-f) パラメータは、画像比較処理を高速化するモードを有効にします。
	// 有効にすると、複数段階のサンプリングを適用し、最初に粗いサンプリングで大まかな位置合わせを行い、
	// 次の段階でより細かいサンプリングで精度を高めます。
	// 比較する画像が大きく、処理時間を短縮したい場合に有効です。
	// --- 追記された説明 ---
	// 高速モードが有効な場合、以下の処理が行われます：
	// 1. 粗いサンプリングレートを使用して、画像全体の大まかな位置合わせを高速に計算します。
	// 2. 粗い位置合わせ結果を基に、探索範囲を絞り込んで次の段階でより細かいサンプリングを適用します。
	// 3. 最終段階では、全ピクセルを使用して精密な位置合わせを行います。
	// この段階的なアプローチにより、計算量を大幅に削減しつつ、精度を維持することが可能です。
	// 特に高解像度の画像や大きな画像を比較する際に効果的です。
	// --- 追記ここまで ---
	optionFastMode = flag.Bool("f", false, "Enable fast mode with progressive sampling")

	// 透過表示の設定
	optionShowOverlay  = flag.Bool("oe", true, "Show transparent overlay of the first image in diff areas")       // overlay → l (layer)
	optionTransparency = flag.Float64("ot", 0.95, "Transparency level for overlay (0.0=opaque, 1.0=transparent)") // alpha → p (percent)

	// 色調設定
	optionUseTint          = flag.Bool("n", true, "Apply color tint to overlay")                                  // tint → n (tint)
	optionTintRed          = flag.Int("tr", 255, "Red component for tint color (0-255)")                          // tint-r → r
	optionTintGreen        = flag.Int("tg", 0, "Green component for tint color (0-255)")                          // tint-g → g
	optionTintBlue         = flag.Int("tb", 0, "Blue component for tint color (0-255)")                           // tint-b → b
	optionTintStrength     = flag.Float64("ts", 0.05, "Tint strength (0.0=no tint, 1.0=full tint)")               // tint-strength → i (intensity)
	optionTintTransparency = flag.Float64("tw", 0.2, "Transparency level for tint (0.0=opaque, 1.0=transparent)") // tint-alpha → w (weight)
)

func init() {
	// ヘルプメッセージのカスタマイズ
	customizeHelpMessage()
}

// main エントリポイント
func main() {
	// コマンドライン引数の解析
	flag.Parse()

	// 必須オプションのチェック
	if err := validateRequiredOptions(); err != nil {
		fmt.Println(err)
		flag.Usage()
		os.Exit(1)
	}

	// 設定情報の表示
	printFlagInfo()

	// 設定オブジェクトの作成
	cfg := createAppConfig()

	// 画像処理の実行
	if err := processImages(cfg); err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Diff image saved to %s\n", *optionOutput)
}

// validateRequiredOptions 必須オプションが指定されているかチェック
func validateRequiredOptions() error {
	var missingOptions []string

	if *optionImageInput1 == "" {
		missingOptions = append(missingOptions, "a")
	}
	if *optionImageInput2 == "" {
		missingOptions = append(missingOptions, "b")
	}
	if *optionOutput == "" {
		missingOptions = append(missingOptions, "o")
	}

	if len(missingOptions) > 0 {
		return fmt.Errorf("\n[ERROR] Missing required option(s): %s\n",
			strings.Join(missingOptions, ", "))
	}

	return nil
}

// printFlagInfo 設定情報を表示
func printFlagInfo() {
	fmt.Printf("[ Command options ]\n")
	flag.VisitAll(func(a *flag.Flag) {
		fmt.Printf("  -%-30s %s\n",
			fmt.Sprintf("%s %v", a.Name, a.Value),
			strings.Trim(a.Usage, "\n"))
	})

	fmt.Printf("\n\n")
}

// createAppConfig アプリケーション設定オブジェクトを作成
func createAppConfig() *config.AppConfig {
	// 色調の範囲を制限
	r := utils.Clamp(*optionTintRed, 0, 255)
	g := utils.Clamp(*optionTintGreen, 0, 255)
	b := utils.Clamp(*optionTintBlue, 0, 255)

	// 透明度の範囲を制限
	transparency := utils.ClampFloat64(*optionTransparency, 0.0, 1.0)
	tintStrength := utils.ClampFloat64(*optionTintStrength, 0.0, 1.0)
	tintTransparency := utils.ClampFloat64(*optionTintTransparency, 0.0, 1.0)

	return &config.AppConfig{
		MaxOffset:              *optionMaxOffset,
		Threshold:              *optionThreshold,
		HighlightDiff:          true, // 常に差分を赤枠で強調表示
		NumCPU:                 *optionNumCPU,
		SamplingRate:           *optionSamplingRate,
		FastMode:               *optionFastMode,
		ProgressStep:           5, // 進捗表示のステップを固定値に設定
		ShowTransparentOverlay: *optionShowOverlay,
		OverlayTransparency:    transparency,
		OverlayTint:            color.RGBA{uint8(r), uint8(g), uint8(b), 255},
		UseTint:                *optionUseTint,
		TintStrength:           tintStrength,
		TintTransparency:       tintTransparency,
	}
}

// processImages 画像処理のメインフロー
func processImages(cfg *config.AppConfig) error {
	startTime := time.Now()

	// 1. 画像の読み込み
	imageA, imageB, err := loadImages()
	if err != nil {
		return err
	}

	// 2. 画像サイズの確認と警告表示
	checkImageDimensions(imageA, imageB)

	// 3. 差分検出と画像生成
	diffImage, err := detectDifferences(imageA, imageB, cfg)
	if err != nil {
		return err
	}

	// 4. 差分画像を保存
	if err := imageutil.SaveDiffImage(diffImage, optionOutput); err != nil {
		return fmt.Errorf("Failed to save diff image: %v", err)
	}

	// 処理時間を表示
	elapsed := time.Since(startTime)
	fmt.Printf("[INFO] Total processing completed in %.2f seconds\n", elapsed.Seconds())

	return nil
}

// loadImages は入力画像を読み込む
func loadImages() (imageA, imageB image.Image, err error) {
	fmt.Printf("[INFO] Loading images...\n")

	imageA, err = imageutil.LoadImage(optionImageInput1)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to load image A: %v", err)
	}

	imageB, err = imageutil.LoadImage(optionImageInput2)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to load image B: %v", err)
	}

	return imageA, imageB, nil
}

// checkImageDimensions は画像サイズの確認と警告表示を行う
func checkImageDimensions(imageA, imageB image.Image) {
	boundsA := imageA.Bounds()
	boundsB := imageB.Bounds()

	fmt.Printf("Image A: %s (%dx%d)\n", *optionImageInput1, boundsA.Dx(), boundsA.Dy())
	fmt.Printf("Image B: %s (%dx%d)\n", *optionImageInput2, boundsB.Dx(), boundsB.Dy())

	// 画像サイズが異なる場合は警告
	if boundsA.Dx() != boundsB.Dx() || boundsA.Dy() != boundsB.Dy() {
		fmt.Printf("[WARNING] Image dimensions do not match!\n")
	}
}

// detectDifferences は画像の差分を検出して差分画像を生成する
func detectDifferences(imageA, imageB image.Image, cfg *config.AppConfig) (image.Image, error) {
	// 差分分析器を生成
	diffAnalyzer := imageutil.NewDiffAnalyzer(cfg)

	// 最適なオフセットを検出
	offsetX, offsetY := diffAnalyzer.FindBestAlignment(imageA, imageB)
	fmt.Printf("Detected offset: (%d, %d)\n", offsetX, offsetY)

	// 検出したオフセットに基づいて差分画像を生成
	return diffAnalyzer.GenerateDiffImage(imageA, imageB, offsetX, offsetY), nil
}

// customizeHelpMessage ヘルプメッセージの表示形式をカスタマイズする
func customizeHelpMessage() {
	b := new(bytes.Buffer)
	func() { flag.CommandLine.SetOutput(b); flag.Usage(); flag.CommandLine.SetOutput(os.Stderr) }()
	usage := strings.Replace(strings.Replace(b.String(), ":", " [OPTIONS] [-h, --help]\n\nDescription:\n  "+commandDescription+"\n\nOptions:\n", 1), "Usage of", "Usage:", 1)
	re := regexp.MustCompile(`[^,] +(-\S+)(?: (\S+))?\n*(\s+)(.*)\n`)
	flag.Usage = func() {
		_, _ = fmt.Fprint(flag.CommandLine.Output(), re.ReplaceAllStringFunc(usage, func(m string) string {
			return fmt.Sprintf("  %-"+commandOptionFieldWidth+"s %s\n", re.FindStringSubmatch(m)[1]+" "+strings.TrimSpace(re.FindStringSubmatch(m)[2]), re.FindStringSubmatch(m)[4])
		}))
	}
}
