# go-img-diff

> **Note**: This repository contains code that was largely generated with the assistance of GitHub Copilot (Claude 3.7 Sonnet).

A tool to detect and visualize differences between images. It automatically detects positional deviations, performs optimal alignment, and highlights differences with red borders. It can also overlay the original image with colored transparency within the difference regions.

This tool is implemented with **zero external dependencies and uses only the standard Go libraries for image processing**.

---

**Input**

<img width="40%" alt="input1" src="https://github.com/user-attachments/assets/ff098e59-e5e5-406a-910e-f019d8e2f897"> <img width="40%" alt="input2" src="https://github.com/user-attachments/assets/b847449f-9c17-4400-a959-352ab9f82193">

**Output**

<img width="70%" alt="output" src="https://github.com/user-attachments/assets/ac358195-a15a-4673-a878-3a7080840516" />


## Usage

```bash
imgdiff -i1 original_image.png -i2 compared_image.png -o diff_image.png [options]
```

## Options

### Required Options

- `-i1` : Path to the original image
- `-i2` : Path to the comparison image
- `-o` : Path to the output diff image

### Misalignment Detection Settings

- `-m` : Maximum offset (in pixels) (default: 10)
  - Search range for image alignment. Larger values detect greater misalignments but increase processing time.

### Difference Detection Settings

- `-d` : Color difference threshold (0-255) (default: 30)
  - Lower values detect smaller differences; higher values detect only larger differences.
  
- `-e` : Exit with status code 1 if differences are found (default: false)
  - When enabled, the program will exit immediately after detecting differences without saving the diff image

### Speedup Settings

- `-s` : Sampling rate (default: 4)
  - 1=all pixels, 2=1/4 of pixels, 4=1/16 of pixels are compared. Higher values speed up processing but reduce accuracy.

- `-p` : Enable precise mode (default: false)
  - Disables the default fast mode for more accurate comparison.

### Display Settings

- `-od` : Disable transparent overlay of the original image in diff areas (default: false)
- `-ot` : Transparency of the original image (default: 0.95)
  - 0.0=completely opaque, 1.0=completely transparent

- `-n` : Apply color tint to the transparent overlay (default: true)
- `-tc` : Tint color as R,G,B (default: "255,0,0")
- `-ts` : Tint strength (default: 0.05)
  - 0.0=no tint (original image as is), 1.0=tint only
- `-tw` : Tint transparency (default: 0.2)
  - 0.0=completely opaque, 1.0=completely transparent

### Other

- `-c` : Number of CPU cores to use (default: number of cores on the system)
- `-v` : Display version information

## Processing Modes

### Fast Mode (Default)

Uses progressive sampling to reduce processing time for large images. It first identifies the overall position with coarse sampling, then gradually improves accuracy with finer sampling.

### Precise Mode (-p)

Performs all comparisons at the specified sampling rate for maximum accuracy. This increases processing time but is useful when more accurate alignment is required.

## Transparent Overlay Feature

Displays the original image with colored transparency in difference areas to make changes easier to see visually.

- Transparent overlay is enabled by default
- `-od`: Disable transparent overlay
- `-n=false`: No color tint applied
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
git tag v0.0.2 && git push --tags


# Delete tag
echo "v0.0.1" |xargs -I{} bash -c "git tag -d {} && git push origin :{}"

# Delete tag and recreate new tag and push
echo "v0.0.2" |xargs -I{} bash -c "git tag -d {} && git push origin :{}; git tag {} -m \"Release beta version.\"; git push --tags"
```
