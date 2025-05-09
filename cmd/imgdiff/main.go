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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xshoji/go-img-diff/config"
	"github.com/xshoji/go-img-diff/imageutil"
	"github.com/xshoji/go-img-diff/utils"
)

// 定数定義
const (
	UsageRequiredPrefix = "\u001B[33m(REQ)\u001B[0m "
	UsageDummy          = "########"
	TimeFormat          = "2006-01-02 15:04:05.0000 [MST]"
)

// アプリケーション設定とオプション
var (
	// コマンドオプション表示に関する設定
	commandDescription      = "Image difference detection and visualization tool."
	commandOptionFieldWidth = 0

	// 必須オプション
	optionImageInput1 = defineFlagValue("i1", "input1", UsageRequiredPrefix+"First image path", "").(*string)
	optionImageInput2 = defineFlagValue("i2", "input2", UsageRequiredPrefix+"Second image path", "").(*string)
	optionOutput      = defineFlagValue("o", "output", UsageRequiredPrefix+"Output diff image path", "").(*string)

	// 位置ずれ検出のための設定
	optionMaxOffset = defineFlagValue("m", "max-offset", "Maximum pixel offset to search for alignment", 10).(*int)

	// 閾値設定
	optionThreshold = defineFlagValue("d", "diff-threshold", "Color difference threshold (0-255)", 30).(*int)

	// 並列処理のためのCPU数設定
	optionNumCPU = defineFlagValue("c", "cpu", "Number of CPU cores to use for parallel processing", runtime.NumCPU()).(*int)

	// サンプリング設定
	optionSamplingRate = defineFlagValue("s", "sampling", "Sampling rate for pixel comparison (1=all pixels, 2=every other pixel, etc)", 4).(*int)

	// 高速モード設定
	optionPreciseMode = defineFlagValue("p", "precise", "Enable precise mode (disables the default fast mode for more accurate comparison)", false).(*bool)

	// 透過表示の設定
	optionNoOverlay    = defineFlagValue("od", "overlay-disable", "Disable transparent overlay of the first image in diff areas", false).(*bool)
	optionTransparency = defineFlagValue("ot", "overlay-transparency", "Transparency level for overlay (0.0=opaque, 1.0=transparent)", 0.95).(*float64)

	// 色調設定
	optionDisableTint      = defineFlagValue("td", "tint-disable", "Disable color tint on overlay", false).(*bool)
	optionTintColor        = defineFlagValue("tc", "tint-color", "Tint color as R,G,B (0-255 for each value)", "255,0,0").(*string)
	optionTintStrength     = defineFlagValue("ts", "tint-strength", "Tint strength (0.0=no tint, 1.0=full tint)", 0.05).(*float64)
	optionTintTransparency = defineFlagValue("tw", "tint-weight", "Transparency level for tint (0.0=opaque, 1.0=transparent)", 0.2).(*float64)

	// 差分検出時に終了ステータス1で終了するオプション
	optionExitOnDiff = defineFlagValue("e", "exit-on-diff", "Exit with status code 1 if differences are found (does not save diff image)", false).(*bool)
)

func init() {
	// ヘルプメッセージのカスタマイズ
	//customizeHelpMessage()
	customizeHelpMessage2(commandDescription, &commandOptionFieldWidth, new(bytes.Buffer))
}

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
	// exitOnDiffが指定されている場合は出力ファイルは不要
	if *optionOutput == "" && !*optionExitOnDiff {
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
		if a.Usage == UsageDummy {
			return
		}
		fmt.Printf("  %-32s %s\n",
			fmt.Sprintf("-%-2s, --%s %v", strings.Split(a.Usage, UsageDummy)[0], a.Name, a.Value),
			strings.Trim(strings.Split(a.Usage, UsageDummy)[1], "\n"))
	})
	fmt.Printf("\n\n")
}

// createAppConfig アプリケーション設定オブジェクトを作成
func createAppConfig() *config.AppConfig {
	// 色調のパース
	r, g, b := parseTintColor(*optionTintColor)

	// 透明度の範囲を制限
	transparency := utils.ClampFloat64(*optionTransparency, 0.0, 1.0)
	tintStrength := utils.ClampFloat64(*optionTintStrength, 0.0, 1.0)
	tintTransparency := utils.ClampFloat64(*optionTintTransparency, 0.0, 1.0)

	// 高速モードは厳密モードが無効の場合に有効
	fastMode := !*optionPreciseMode

	return &config.AppConfig{
		MaxOffset:              *optionMaxOffset,
		Threshold:              *optionThreshold,
		HighlightDiff:          true, // 常に差分を赤枠で強調表示
		NumCPU:                 *optionNumCPU,
		SamplingRate:           *optionSamplingRate,
		FastMode:               fastMode,
		ProgressStep:           5, // 進捗表示のステップを固定値に設定
		ShowTransparentOverlay: !*optionNoOverlay,
		OverlayTransparency:    transparency,
		OverlayTint:            color.RGBA{uint8(r), uint8(g), uint8(b), 255},
		UseTint:                !*optionDisableTint,
		TintStrength:           tintStrength,
		TintTransparency:       tintTransparency,
	}
}

// parseTintColor は "R,G,B" 形式の文字列から RGB の整数値を取得します
func parseTintColor(colorStr string) (r, g, b int) {
	r, g, b = 255, 0, 0 // デフォルト値は赤

	parts := strings.Split(colorStr, ",")
	if len(parts) != 3 {
		fmt.Printf("[WARNING] Invalid tint color format '%s'. Using default (255,0,0).\n", colorStr)
		return
	}

	var err error
	if r, err = strconv.Atoi(strings.TrimSpace(parts[0])); err != nil {
		fmt.Printf("[WARNING] Invalid red value in tint color. Using default (255).\n")
		r = 255
	}
	if g, err = strconv.Atoi(strings.TrimSpace(parts[1])); err != nil {
		fmt.Printf("[WARNING] Invalid green value in tint color. Using default (0).\n")
		g = 0
	}
	if b, err = strconv.Atoi(strings.TrimSpace(parts[2])); err != nil {
		fmt.Printf("[WARNING] Invalid blue value in tint color. Using default (0).\n")
		b = 0
	}

	// 範囲を制限
	r = utils.Clamp(r, 0, 255)
	g = utils.Clamp(g, 0, 255)
	b = utils.Clamp(b, 0, 255)

	return
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
	diffImage, hasDiff, err := detectDifferences(imageA, imageB, cfg)
	if err != nil {
		return err
	}

	// 差分があり、exitOnDiffオプションが有効な場合は早期終了
	if hasDiff && *optionExitOnDiff {
		fmt.Println("[INFO] Differences detected. Exiting with status code 1.")
		os.Exit(1)
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
func detectDifferences(imageA, imageB image.Image, cfg *config.AppConfig) (image.Image, bool, error) {
	// 差分分析器を生成
	diffAnalyzer := imageutil.NewDiffAnalyzer(cfg)

	// 最適なオフセットを検出
	offsetX, offsetY := diffAnalyzer.FindBestAlignment(imageA, imageB)
	fmt.Printf("Detected offset: (%d, %d)\n", offsetX, offsetY)

	// 差分があるかどうかを検出
	hasDiff := diffAnalyzer.HasDifferences(imageA, imageB, offsetX, offsetY)

	// 検出したオフセットに基づいて差分画像を生成
	return diffAnalyzer.GenerateDiffImage(imageA, imageB, offsetX, offsetY), hasDiff, nil
}

// Helper function for flag
func defineFlagValue(short, long, description string, defaultValue any) (f any) {
	flagUsage := short + UsageDummy + description
	switch v := defaultValue.(type) {
	case string:
		f = flag.String(short, "", UsageDummy)
		flag.StringVar(f.(*string), long, v, flagUsage)
	case int:
		f = flag.Int(short, 0, UsageDummy)
		flag.IntVar(f.(*int), long, v, flagUsage)
	case bool:
		f = flag.Bool(short, false, UsageDummy)
		flag.BoolVar(f.(*bool), long, v, flagUsage)
	case float64:
		f = flag.Float64(short, 0.0, UsageDummy)
		flag.Float64Var(f.(*float64), long, v, flagUsage)
	default:
		panic("unsupported flag type")
	}
	return
}

// customizeHelpMessage ヘルプメッセージの表示形式をカスタマイズする
func customizeHelpMessage() {
	b := new(bytes.Buffer)
	func() { flag.CommandLine.SetOutput(b); flag.Usage(); flag.CommandLine.SetOutput(os.Stderr) }()
	usage := strings.Replace(strings.Replace(b.String(), ":", " [OPTIONS] [-h, --help]\n\nDescription:\n  "+commandDescription+"\n\nOptions:\n", 1), "Usage of", "Usage:", 1)
	re := regexp.MustCompile(`[^,] +(-\S+)(?: (\S+))?\n*(\s+)(.*)\n`)
	flag.Usage = func() {
		_, _ = fmt.Fprint(flag.CommandLine.Output(), re.ReplaceAllStringFunc(usage, func(m string) string {
			return fmt.Sprintf("  %-"+strconv.Itoa(commandOptionFieldWidth)+"s %s\n", re.FindStringSubmatch(m)[1]+" "+strings.TrimSpace(re.FindStringSubmatch(m)[2]), re.FindStringSubmatch(m)[4])
		}))
	}
}

func customizeHelpMessage2(description string, maxLength *int, buffer *bytes.Buffer) {
	func() { flag.CommandLine.SetOutput(buffer); flag.Usage(); flag.CommandLine.SetOutput(os.Stderr) }()
	usageOption := regexp.MustCompile("(-\\S+)( *\\S*)+\n*\\s+"+UsageDummy+"\n\\s*").ReplaceAllString(buffer.String(), "")
	re := regexp.MustCompile("\\s(-\\S+)( *\\S*)( *\\S*)+\n\\s+(.+)")
	usageFirst := strings.Replace(strings.Replace(strings.Split(usageOption, "\n")[0], ":", " [OPTIONS] [-h, --help]", -1), " of ", ": ", -1) + "\n\nDescription:\n  " + description + "\n\nOptions:\n"
	usageOptions := re.FindAllString(usageOption, -1)
	for _, v := range usageOptions {
		*maxLength = max(*maxLength, len(re.ReplaceAllString(v, " -$1")+re.ReplaceAllString(v, "$2"))+2)
	}
	usageOptionsRep := make([]string, 0)
	for _, v := range usageOptions {
		usageOptionsRep = append(usageOptionsRep, fmt.Sprintf("  -%-2s,%-"+strconv.Itoa(*maxLength)+"s%s", strings.Split(re.ReplaceAllString(v, "$4"), UsageDummy)[0], re.ReplaceAllString(v, " -$1")+re.ReplaceAllString(v, "$2"), strings.Split(re.ReplaceAllString(v, "$4"), UsageDummy)[1]+"\n"))
	}
	sort.SliceStable(usageOptionsRep, func(i, j int) bool {
		return strings.Count(usageOptionsRep[i], UsageRequiredPrefix) > strings.Count(usageOptionsRep[j], UsageRequiredPrefix)
	})
	flag.Usage = func() { _, _ = fmt.Fprint(flag.CommandLine.Output(), usageFirst+strings.Join(usageOptionsRep, "")) }
}
