# go-img-diff

A tool to detect and visualize differences between two images. It automatically detects positional deviations in images, performs optimal alignment, and highlights differences with red borders. It can also overlay the original image with colored transparency within the difference regions.

This tool is implemented with zero external dependencies and does not rely on OpenCV or any other image processing libraries. **It uses only the standard Go libraries for image processing**.

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
  - Search range for image alignment. Larger values increase processing time but can detect larger misalignments.

### Difference Detection Settings

- `-d` : Color difference threshold (0-255) (default: 30)
  - Lowering the threshold detects smaller differences; raising it detects only larger differences.

### Speedup Settings

- `-s` : Sampling rate (default: 4)
  - 1=all pixels, 2=1/4 of pixels, 4=1/16 of pixels are compared. Increasing the value speeds up processing but reduces accuracy.

- `-p` : Enable precise mode (default: false)
  - Disables the default fast mode for more accurate comparison. Use this when accuracy is more important than speed.

### Display Settings

- `-od` : Disable transparent overlay of the first image in diff areas (default: false)
  - By default, original image is overlaid in difference areas. Use this flag to disable the overlay.

- `-ot` : Transparency of the original image (default: 0.95)
  - 0.0=completely opaque, 1.0=completely transparent

- `-n` : Apply color tint to the transparent overlay (default: true)
  - Makes the differences more noticeable by adding a color tint to the original image.

- `-tc` : Tint color as R,G,B (default: "255,0,0")
  - Specify the color of the tint in RGB format as a comma-separated string.

- `-ts` : Tint strength (default: 0.05)
  - 0.0=no tint (original image as is), 1.0=tint only

- `-tw` : Tint transparency (default: 0.2)
  - 0.0=completely opaque, 1.0=completely transparent
  - Can be set separately from the original image transparency (`-ot`)

### Other

- `-c` : Number of CPU cores to use (default: number of cores on the system)
- `-v` : Display version information

## Details of the MaxOffset (-m) Parameter

The `-m` option specifies the maximum positional deviation (offset) detection range in pixels when comparing two images.

Increasing the value:
- Benefit: Increases the possibility of accurately aligning even largely shifted images.
- Drawback: Significantly increases processing time (because the search range increases quadratically).
- Drawback: Increases the possibility of false detections.

Decreasing the value:
- Benefit: Speeds up processing.
- Benefit: Reduces false matches to locally similar areas.
- Drawback: Makes it impossible to correctly align largely shifted images.

## Details of the Sampling Rate (-s) Parameter

The `-s` option specifies the sampling interval when comparing pixels.

- `s=1`: Compare all pixels (most accurate but slowest)
- `s=2`: Compare every other pixel (number of pixels to compare is reduced to 1/4)
- `s=4`: Compare every 3 pixels (number of pixels to compare is reduced to 1/16)

## Details of the Processing Mode Parameters

### Fast Mode (Default)

By default, the tool operates in fast mode with progressive sampling, which significantly reduces processing time for large images.

In this mode, the overall position is first identified with coarse sampling, and then the accuracy is gradually improved with finer sampling. This approach is especially effective for high-resolution images.

### Precise Mode (-p)

The `-p` option enables precise mode by disabling the default fast mode.

In precise mode, all comparisons are performed with the specified sampling rate without progressive optimization. This ensures maximum accuracy but increases processing time, especially for large images or when searching for large offsets.

Use this mode when:
- You need the most accurate alignment possible
- Fast mode produces unsatisfactory results
- You're analyzing small details in images

## Transparent Overlay Display Function

Using this function, the pixel information of the original image (the image specified with `-i1`) is displayed with colored transparency in the area where the difference is detected. This makes it easier to visually check what kind of changes have been made.

### Basic Transparent Display

- By default, transparent overlay is enabled
- `-no`: Disable transparent overlay
- `-ot=0.95` (default): 95% transparency of the original image

### Tint Adjustment

You can make the difference more noticeable by adding a color tint to the original image:

- `-n=true` (default): Apply tint
- `-n=false`: Do not apply tint; display with the original color as is
- `-tc=255,0,0` (default): Red tint
- `-tc=0,0,255`: Blue tint
- `-tc=255,255,0`: Yellow tint

### Detailed Control of Tint and Transparency

- `-ts=0.05` (default): Set the tint strength to 5%
- `-ts=0.3`: Tint moderately (original image color remains strong)
- `-ts=1.0`: Tint only (original image color does not remain)

- `-tw=0.2` (default): Set the tint transparency to 20% (relatively clear)
- `-ot=0.5 -tw=0.1`: Original image is translucent, tint is clear

By combining these parameters, you can finely adjust the visibility of the differences.
