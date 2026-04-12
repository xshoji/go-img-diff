# go-img-diff

> **Note**: This repository contains code that was largely generated with the assistance of Claude Opus 4.6.

A tool to detect and visualize differences between images. It automatically detects positional deviations, performs optimal alignment, and highlights differences with red borders. It can also overlay the original image with colored transparency within the difference regions.

This tool is implemented with **zero external dependencies and uses only the standard Go libraries for image processing**.

---

**Input**

<img width="40%" alt="input1" src="https://github.com/user-attachments/assets/e14b09b9-b36d-43d3-8016-4a4fb267a225"> <img width="40%" alt="input2" src="https://github.com/user-attachments/assets/55c9bab5-5847-4d89-8a18-2f26a396e707">

**Output example**

`imgdiff -i1 image1.png -i2 image2.png -o diff1.png -ot 0.99`

<img width="60%" alt="output" src="https://github.com/user-attachments/assets/ee223fb6-0150-453b-ab2e-a27bfece28fb" />

`imgdiff -i1 image1.png -i2 image2.png -o diff2.png -od -l horizontal`

<img width="90%" alt="output" src="https://github.com/user-attachments/assets/d1964692-f8d9-4723-a231-8fba162fce09" />


## Install

### Homebrew

```bash
brew install xshoji/tap/imgdiff
```

### Go

```bash
go install github.com/xshoji/go-img-diff/cmd/imgdiff@latest
```

Or download pre-built binaries from the [Releases](https://github.com/xshoji/go-img-diff/releases) page.

## Usage

```bash
imgdiff -i1 original_image.png -i2 compared_image.png -o diff_image.png [options]
```

## Options

### Required Options

- `-i1`, `--input1` : Path to the first image
- `-i2`, `--input2` : Path to the second image
- `-o`, `--output` : Path to the output diff image (required unless `-e` is specified)

### Misalignment Detection Settings

- `-m`, `--max-offset` : Maximum pixel offset to search for alignment (default: 10)
  - Search range for image alignment. Larger values detect greater misalignments but increase processing time.

- `-sw`, `--strip-width` : Width of each vertical strip used for local DP realignment (default: 320)
  - Smaller values preserve independently fixed areas like sidebars more aggressively.
  - Larger values allow broader content blocks to move together, but may pull unrelated columns into the same alignment.

### Difference Detection Settings

- `-d`, `--diff-threshold` : Color difference threshold (0-255) (default: 30)
  - Lower values detect smaller differences; higher values detect only larger differences.

- `-nw`, `--noise-window-size` : Local window size for sparse-noise filtering (default: 0)
  - Set a value like `5`, `7`, or `9` to evaluate diff density in a local neighborhood.

- `-nr`, `--noise-min-ratio` : Minimum diff density in the local window to keep a diff pixel (default: 0.0)
  - Useful for ignoring sparse noise from compression artifacts or subtle image degradation.
  - Example: `-nw 7 -nr 0.08`

- `-ra`, `--min-region-area` : Minimum diff region area to keep (default: 4)
  - Higher values ignore tiny residual differences and small noise-like regions.
  
- `-e`, `--exit-on-diff` : Exit with status code 1 if differences are found (default: false)
  - When enabled, the program exits with status code 1 if differences are detected. The `-o` option can be omitted to skip saving the diff image.

### Speedup Settings

- `-s`, `--sampling` : Sampling rate for pixel comparison (default: 4)
  - 1=all pixels, higher values speed up processing but may reduce accuracy.

- `-p`, `--precise` : Enable precise mode (default: false)
  - Uses a larger pyramid min-size for more accurate comparison at the cost of processing time.

### Display Settings

- `-l`, `--layout` : Output layout (default: "simple")
  - `simple`: Outputs only the diff image
  - `horizontal`: Outputs the first image and diff image side by side

- `-od`, `--overlay-disable` : Disable transparent overlay of the first image in diff areas (default: false)
- `-ot`, `--overlay-transparency` : Transparency level for overlay (default: 0.95)
  - 0.0=completely opaque, 1.0=completely transparent

- `-td`, `--tint-disable` : Disable color tint on the transparent overlay (default: false)
- `-tc`, `--tint-color` : Tint color as R,G,B (0-255 for each value) (default: "255,0,0")
- `-ts`, `--tint-strength` : Tint strength (default: 0.05)
  - 0.0=no tint (original image as is), 1.0=full tint
- `-tw`, `--tint-weight` : Transparency level for tint (default: 0.2)
  - 0.0=completely opaque, 1.0=completely transparent

### Performance

- `-c`, `--cpu` : Number of CPU cores to use for parallel processing (default: number of available CPU cores)
  - Limits parallelization for processing multiple regions. Useful for controlling resource usage on multi-core systems.

## Processing Modes

### Fast Mode (Default)

Uses a pyramid multi-scale approach to reduce processing time for large images. It first identifies the overall position with coarse sampling at reduced scales, then gradually refines accuracy at finer scales.

### Precise Mode (-p, --precise)

Uses a larger pyramid min-size for more accurate comparison. This increases processing time but is useful when more accurate alignment is required.

## Transparent Overlay Feature

Displays the original image with colored transparency in difference areas to make changes easier to see visually.

- Transparent overlay is enabled by default
- `-od`: Disable transparent overlay
- `-td`: Disable color tint
- `-tc=0,0,255`: Blue tint
- `-tc=255,255,0`: Yellow tint

Combine parameters to fine-tune the visibility of differences.

## Unit Testing

```
# All tests
go test ./...

# Light tests only
go test -tags="light_test_only" ./...
```

## Release

The release flow for this repository is automated with GitHub Actions.
Pushing Git tags triggers the release job.

```
# Release
git tag v0.0.6 && git push --tags


# Delete tag
v="v0.0.6"; git tag -d "${v}" && git push origin :"${v}"

# Delete tag and recreate new tag and push
v="v0.0.6"; git tag -d "${v}" && git push origin :"${v}"; git tag "${v}" -m "Release "; git push --tags
```
