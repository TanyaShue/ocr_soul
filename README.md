# ocr_soul

Go command-line recognizer for Onmyoji soul enhancement result screenshots.

The recognizer is tuned for the 1280x720 enhancement result UI shown in the
sample screenshots under `assets`. It does not depend on the inaccurate OCR
model in `ocr`; instead it uses fixed UI geometry, text-color segmentation, and
templates learned from the supplied screenshots. This is more reliable for this
specific game screen than generic OCR.

## Usage

Recognize every PNG in `assets`:

```powershell
go run . -input assets
```

Recognize one image:

```powershell
go run . -input assets\MuMu-20260708-142607-072.png
```

If the training screenshots are moved, point `-assets` at the directory that
contains the three sample images:

```powershell
go run . -assets assets -input path\to\screenshot.png
```

## Output

The command writes JSON to stdout. Each image contains a `souls` array with:

- `index`: order of the soul in the visible result grid.
- `main_attribute`: main stat before and after enhancement.
- `sub_attribute_count`: number of sub stats detected.
- `sub_attributes`: sub stats before and after enhancement.
- `upgraded_attribute`: the sub stat that received the upgrade.
- `final_attributes`: final complete stat list.

## Verification

```powershell
go test ./...
```

The tests assert exact recognition for the three provided screenshots,
covering 1, 2, and 6 simultaneous soul enhancement results.
