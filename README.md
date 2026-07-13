# ocr_soul

Go command-line recognizer for Onmyoji soul enhancement result screenshots.

The recognizer is tuned for the 1280x720 enhancement result UI shown in the
sample screenshots under `assets`. It does not depend on the inaccurate OCR
model in `ocr`; instead it loads a Go template model generated from labelled
screenshots. The default model is embedded from `models/soul_ocr_model.gob`, so
normal recognition does not retrain from `assets` on startup.

## Usage

Show supported commands, parameters, and examples:

```powershell
go run . help
go run . help recognize
go run . help export-model
```

The built binary uses the same commands:

```powershell
.\ocr_soul.exe help
.\ocr_soul.exe recognize -input assets
```

### Commands and parameters

| Command | Purpose | Parameters |
| --- | --- | --- |
| `recognize` | Recognize a PNG file or every PNG in a directory and write JSON to stdout. This is the default command when the first argument is a flag. | `-input <file-or-dir>`: input PNG or directory, default `assets`.<br>`-model <file>`: optional template model path; empty value uses the embedded model. |
| `export-model` | Rebuild the Go template model from labelled screenshots. | `-assets <dir>`: labelled screenshot directory, default `assets`.<br>`-model <file>`: output model path, default `models\soul_ocr_model.gob`. |

General forms:

```powershell
go run . [recognize] [flags]
go run . recognize [flags] [image-or-dir]
go run . export-model [flags]
go run . help [command]
```

Recognize every PNG in `assets` with the embedded model:

```powershell
go run . recognize -input assets
```

Recognize one image:

```powershell
go run . recognize -input assets\MuMu-20260708-142607-072.png
```

The input path may also be passed positionally after `recognize`:

```powershell
go run . recognize assets\MuMu-20260708-142607-072.png
```

The old shorthand is still supported and also uses the embedded model:

```powershell
go run . -input assets
```

Use an explicit model file:

```powershell
go run . recognize -model models\soul_ocr_model.gob -input assets
```

Export a new Go template model from the labelled screenshots:

```powershell
go run . export-model -assets assets -model models\soul_ocr_model.gob
```

## Output

The command writes versioned JSON to stdout:

```json
{
  "schema_version": 2,
  "results": [
    {
      "image": "assets/MuMu-20260708-142607-072.png",
      "souls": [
        {
          "order": 1,
          "position": 6,
          "level": {
            "initial": 0,
            "final": 3
          },
          "attributes": {
            "main": {
              "name": "防御加成",
              "initial": "10.00%",
              "final": "19.00%"
            },
            "subs": [
              {
                "name": "防御加成",
                "initial": "2.56%",
                "final": "5.26%"
              }
            ]
          }
        }
      ]
    }
  ]
}
```

- `schema_version`: output schema version, currently `2`.
- `results`: recognition results grouped by input image.
- `order`: order of the soul in the visible result grid.
- `position`: soul position, from 1 to 6. It is independent from `order`.
- `level.initial` / `level.final`: level before and after enhancement. Levels
  from 0 through 15 are supported, including two-digit values.
- `attributes.main`: the main stat before and after enhancement.
- `attributes.subs`: the complete sub-stat list, with each attribute stored only
  once. Its array length is the sub-stat count.
- `initial: null`: the sub stat was newly added during enhancement.

Consumers can derive changed attributes by comparing `initial` with `final`,
and can read the complete final stat set directly from each attribute's `final`
value. The output no longer duplicates attributes into separate upgraded and
final-stat collections.

When a soul did not level up, missing gold final values are filled with the
initial value. For example, `Lv.3 -> Lv.3` with a main stat of `19.00%` outputs
`final: "19.00%"`, not an empty value.

## Verification

```powershell
go test ./...
```

The tests assert exact recognition for all provided screenshots, verify the
exported model can be loaded, and verify the embedded model can recognize
position and level data without rebuilding templates from `assets`.
