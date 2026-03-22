package main

import (
	"flag"
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/xshoji/go-img-diff/internal/app"
	"github.com/xshoji/go-img-diff/internal/core"
)

// version is set at build time via ldflags.
var version = "dev"

const (
	Req        = "\033[33m(required)\033[0m "
	UsageDummy = "########"
)

var (
	commandDescription = "Image difference detection and visualization tool."

	// Required
	optionImageInput1 = defineFlagValue("i1", "input1", Req+"First image path", "", flag.String, flag.StringVar)
	optionImageInput2 = defineFlagValue("i2", "input2", Req+"Second image path", "", flag.String, flag.StringVar)
	optionOutput      = defineFlagValue("o", "output", Req+"Output diff image path", "", flag.String, flag.StringVar)

	// Alignment
	optionMaxOffset = defineFlagValue("m", "max-offset", "Maximum pixel offset to search for alignment", 10, flag.Int, flag.IntVar)

	// Diff
	optionThreshold = defineFlagValue("d", "diff-threshold", "Color difference threshold (0-255)", 30, flag.Int, flag.IntVar)

	// Runtime
	optionNumCPU = defineFlagValue("c", "cpu", "Number of CPU cores to use for parallel processing", runtime.NumCPU(), flag.Int, flag.IntVar)

	// Sampling (kept for backward compat, controls MinPyramidSize)
	optionSamplingRate = defineFlagValue("s", "sampling", "Sampling rate for pixel comparison (1=all pixels, higher=faster)", 4, flag.Int, flag.IntVar)

	// Precise mode (disables pyramid multi-scale, uses single-scale brute force)
	optionPreciseMode = defineFlagValue("p", "precise", "Enable precise mode (larger pyramid min-size for more accurate comparison)", false, flag.Bool, flag.BoolVar)

	// Overlay
	optionNoOverlay    = defineFlagValue("od", "overlay-disable", "Disable transparent overlay of the first image in diff areas", false, flag.Bool, flag.BoolVar)
	optionTransparency = defineFlagValue("ot", "overlay-transparency", "Transparency level for overlay (0.0=opaque, 1.0=transparent)", 0.95, flag.Float64, flag.Float64Var)

	// Tint
	optionDisableTint      = defineFlagValue("td", "tint-disable", "Disable color tint on overlay", false, flag.Bool, flag.BoolVar)
	optionTintColor        = defineFlagValue("tc", "tint-color", "Tint color as R,G,B (0-255 for each value)", "255,0,0", flag.String, flag.StringVar)
	optionTintStrength     = defineFlagValue("ts", "tint-strength", "Tint strength (0.0=no tint, 1.0=full tint)", 0.05, flag.Float64, flag.Float64Var)
	optionTintTransparency = defineFlagValue("tw", "tint-weight", "Transparency level for tint (0.0=opaque, 1.0=transparent)", 0.2, flag.Float64, flag.Float64Var)

	// Layout
	optionOutputLayout = defineFlagValue("l", "layout", "Output layout: 'simple' (diff image only) or 'horizontal' (input1 + diff side by side)", "simple", flag.String, flag.StringVar)

	// Exit on diff
	optionExitOnDiff = defineFlagValue("e", "exit-on-diff", "Exit with status code 1 if differences are found (does not save diff image)", false, flag.Bool, flag.BoolVar)
)

func init() {
	flag.Usage = customUsage(commandDescription)
}

func main() {
	flag.Parse()

	if err := validateRequiredOptions(); err != nil {
		fmt.Println(err)
		flag.Usage()
		os.Exit(1)
	}

	layout := core.Layout(*optionOutputLayout)
	if layout != core.LayoutSimple && layout != core.LayoutHorizontal {
		fmt.Printf("[ERROR] Invalid layout value '%s'. Must be 'simple' or 'horizontal'.\n", *optionOutputLayout)
		os.Exit(1)
	}

	// Print current options
	optionValues, _ := getOptionsUsage(true)
	fmt.Printf("[ Command options ]\n%s\n", optionValues)

	// Build options
	opts := buildOptions(layout)

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	hasDiff, err := app.Run(opts, *optionExitOnDiff, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}

	if *optionExitOnDiff && hasDiff {
		fmt.Println("[INFO] Differences detected. Exiting with status code 1.")
		os.Exit(1)
	}

	if opts.Output.Path != "" {
		fmt.Printf("Diff image saved to %s\n", opts.Output.Path)
	}
}

func validateRequiredOptions() error {
	var missing []string
	if *optionImageInput1 == "" {
		missing = append(missing, "i1")
	}
	if *optionImageInput2 == "" {
		missing = append(missing, "i2")
	}
	if *optionOutput == "" && !*optionExitOnDiff {
		missing = append(missing, "o")
	}
	if len(missing) > 0 {
		return fmt.Errorf("[ERROR] Missing required option(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

func buildOptions(layout core.Layout) core.Options {
	r, g, b := parseTintColor(*optionTintColor)

	transparency := clampF64(*optionTransparency, 0.0, 1.0)
	tintStrength := clampF64(*optionTintStrength, 0.0, 1.0)
	tintTransparency := clampF64(*optionTintTransparency, 0.0, 1.0)

	// Precise mode: use larger MinPyramidSize to reduce pyramid levels
	minPyramidSize := 32
	if *optionPreciseMode {
		minPyramidSize = 8 // more levels for more accuracy
	}

	// Sampling rate affects MinPyramidSize inversely
	_ = *optionSamplingRate // kept for backward compat

	return core.Options{
		Input1: *optionImageInput1,
		Input2: *optionImageInput2,
		Align: core.AlignOptions{
			MaxOffset:        *optionMaxOffset,
			MinPyramidSize:   minPyramidSize,
			RefinementRadius: 2,
		},
		Diff: core.DiffOptions{
			Threshold: uint8(clampInt(*optionThreshold, 0, 255)),
		},
		Region: core.RegionOptions{
			MinArea:      4,
			Padding:      5,
			DilateRadius: 1,
		},
		Render: core.RenderOptions{
			DrawOverlay:      !*optionNoOverlay,
			OverlayAlpha:     transparency,
			TintEnabled:      !*optionDisableTint,
			TintColor:        color.NRGBA{uint8(r), uint8(g), uint8(b), 255},
			TintStrength:     tintStrength,
			TintTransparency: tintTransparency,
			BorderColor:      color.NRGBA{255, 0, 0, 255},
			BorderWidth:      3,
			Layout:           layout,
		},
		Runtime: core.RuntimeOptions{
			Workers: *optionNumCPU,
		},
		Output: core.OutputOptions{
			Path: *optionOutput,
		},
	}
}

func parseTintColor(colorStr string) (r, g, b int) {
	r, g, b = 255, 0, 0
	parts := strings.Split(colorStr, ",")
	if len(parts) != 3 {
		fmt.Printf("[WARNING] Invalid tint color format '%s'. Using default (255,0,0).\n", colorStr)
		return
	}
	var err error
	if r, err = strconv.Atoi(strings.TrimSpace(parts[0])); err != nil {
		r = 255
	}
	if g, err = strconv.Atoi(strings.TrimSpace(parts[1])); err != nil {
		g = 0
	}
	if b, err = strconv.Atoi(strings.TrimSpace(parts[2])); err != nil {
		b = 0
	}
	r = clampInt(r, 0, 255)
	g = clampInt(g, 0, 255)
	b = clampInt(b, 0, 255)
	return
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampF64(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// =======================================
// flag Utils
// =======================================

func defineFlagValue[T comparable](short, long, description string, defaultValue T, flagFunc func(name string, value T, usage string) *T, flagVarFunc func(p *T, name string, value T, usage string)) *T {
	flagUsage := short + UsageDummy + description
	var zero T
	if defaultValue != zero {
		flagUsage = flagUsage + fmt.Sprintf(" (default %v)", defaultValue)
	}
	f := flagFunc(long, defaultValue, flagUsage)
	flagVarFunc(f, short, defaultValue, UsageDummy)
	return f
}

func customUsage(description string) func() {
	return func() {
		optionsUsage, requiredOptionExample := getOptionsUsage(false)
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s %s[OPTIONS]\n  version: %s\n\n", func() string { e, _ := os.Executable(); return filepath.Base(e) }(), requiredOptionExample, version)
		fmt.Fprintf(flag.CommandLine.Output(), "Description:\n  %s\n\n", description)
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n%s", optionsUsage)
	}
}

func getOptionsUsage(currentValue bool) (string, string) {
	requiredOptionExample := ""
	optionNameWidth := 0
	usages := make([]string, 0)
	getType := func(v string) string {
		return strings.NewReplacer("*flag.boolValue", "", "*flag.", "<", "Value", ">").Replace(v)
	}
	flag.VisitAll(func(f *flag.Flag) {
		optionNameWidth = max(optionNameWidth, len(fmt.Sprintf("%s %s", f.Name, getType(fmt.Sprintf("%T", f.Value))))+4)
	})
	flag.VisitAll(func(f *flag.Flag) {
		if f.Usage == UsageDummy {
			return
		}
		value := getType(fmt.Sprintf("%T", f.Value))
		if currentValue {
			value = f.Value.String()
		}
		short := strings.Split(f.Usage, UsageDummy)[0]
		mainUsage := strings.Split(f.Usage, UsageDummy)[1]
		if strings.Contains(mainUsage, Req) {
			requiredOptionExample += fmt.Sprintf("--%s %s ", f.Name, value)
		}
		usages = append(usages, fmt.Sprintf("  -%-2s, --%-"+strconv.Itoa(optionNameWidth)+"s %s\n", short, f.Name+" "+value, mainUsage))
	})
	sort.SliceStable(usages, func(i, j int) bool {
		return strings.Count(usages[i], Req) > strings.Count(usages[j], Req)
	})
	return strings.Join(usages, ""), requiredOptionExample
}
