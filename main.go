package main

import (
	"bytes"
	_ "embed"
	"encoding/gob"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed models/soul_ocr_model.gob
var embeddedModel []byte

const (
	defaultInputPath = "assets"
	defaultModelPath = "models/soul_ocr_model.gob"
)

type SoulResult struct {
	Index              int              `json:"index"`
	Position           int              `json:"position"`
	InitialLevel       int              `json:"initial_level"`
	FinalLevel         int              `json:"final_level"`
	MainAttribute      AttributeValue   `json:"main_attribute"`
	SubAttributeCount  int              `json:"sub_attribute_count"`
	SubAttributes      []AttributeValue `json:"sub_attributes"`
	UpgradedAttributes []AttributeValue `json:"upgraded_attributes,omitempty"`
	FinalAttributes    FinalAttributes  `json:"final_attributes"`
}

type AttributeValue struct {
	Name         string `json:"name"`
	InitialValue string `json:"initial_value,omitempty"`
	FinalValue   string `json:"final_value"`
	Upgraded     bool   `json:"upgraded,omitempty"`
}

type FinalAttributes struct {
	Main AttributeValue   `json:"main"`
	Subs []AttributeValue `json:"subs"`
}

type ImageResult struct {
	Image string       `json:"image"`
	Souls []SoulResult `json:"souls"`
}

type Point struct {
	X int
	Y int
}

type Rect struct {
	X0 int
	Y0 int
	X1 int
	Y1 int
}

type TrainingSoul struct {
	File         string
	Slot         int
	Position     int
	InitialLevel int
	FinalLevel   int
	Main         TrainingAttr
	Subs         []TrainingAttr
}

type TrainingAttr struct {
	Name  string
	Init  string
	Final string
	New   bool
}

type BinaryImage struct {
	W int
	H int
	P []bool
}

type Component struct {
	X0   int
	Y0   int
	X1   int
	Y1   int
	Area int
	Mask BinaryImage
}

type DigitTemplate struct {
	Char rune
	Mask BinaryImage
}

type LabelTemplate struct {
	Name string
	Mask BinaryImage
}

type PositionTemplate struct {
	Position int
	Mask     BinaryImage
}

type TemplateModel struct {
	Version           int
	LabelTemplates    []LabelTemplate
	DigitTemplates    []DigitTemplate
	PositionTemplates []PositionTemplate
}

type Recognizer struct {
	LabelTemplates    []LabelTemplate
	DigitTemplates    []DigitTemplate
	PositionTemplates []PositionTemplate
}

var slots = []Point{
	{204, 79}, {503, 79}, {802, 79},
	{204, 373}, {503, 373}, {802, 373},
}

var rowRects = []Rect{
	{10, 70, 260, 103},
	{10, 103, 260, 136},
	{10, 136, 260, 169},
	{10, 169, 260, 202},
	{10, 202, 260, 235},
}

var trainingSet = []TrainingSoul{
	{File: "MuMu-20260708-142511-164.png", Slot: 1, Position: 4, InitialLevel: 0, FinalLevel: 3, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "4.54"}, {Name: "防御加成", Init: "2.46%"}, {Name: "速度", Init: "2.99"}, {Name: "效果命中", Init: "3.46%", Final: "7.35%"},
	}},
	{File: "MuMu-20260708-142511-164.png", Slot: 2, Position: 2, InitialLevel: 0, FinalLevel: 3, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "4.99"}, {Name: "防御加成", Init: "2.77%"}, {Name: "效果命中", Init: "3.42%", Final: "7.01%"}, {Name: "效果抵抗", Init: "3.41%"},
	}},
	{File: "MuMu-20260708-142511-164.png", Slot: 3, Position: 4, InitialLevel: 0, FinalLevel: 3, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "生命", Init: "106.70"}, {Name: "防御加成", Init: "2.49%"}, {Name: "攻击加成", Init: "2.99%"}, {Name: "效果命中", Init: "3.22%", Final: "6.84%"},
	}},
	{File: "MuMu-20260708-142511-164.png", Slot: 4, Position: 2, InitialLevel: 0, FinalLevel: 3, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "5.00"}, {Name: "防御加成", Init: "2.80%", Final: "5.60%"}, {Name: "攻击加成", Init: "2.99%"}, {Name: "暴击", Init: "2.41%"},
	}},
	{File: "MuMu-20260708-142511-164.png", Slot: 5, Position: 2, InitialLevel: 0, FinalLevel: 3, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "生命加成", Init: "2.53%"}, {Name: "防御加成", Init: "2.44%"}, {Name: "速度", Init: "2.54", Final: "5.44"}, {Name: "效果命中", Init: "3.82%"},
	}},
	{File: "MuMu-20260708-142511-164.png", Slot: 6, Position: 6, InitialLevel: 0, FinalLevel: 3, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "4.96"}, {Name: "攻击", Init: "22.08"}, {Name: "防御加成", Init: "2.71%", Final: "5.48%"}, {Name: "速度", Init: "2.88"},
	}},
	{File: "MuMu-20260708-142607-072.png", Slot: 1, Position: 6, InitialLevel: 0, FinalLevel: 3, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "4.55"}, {Name: "防御加成", Init: "2.56%", Final: "5.26%"}, {Name: "速度", Init: "2.83"}, {Name: "暴击伤害", Init: "3.88%"},
	}},
	{File: "MuMu-20260708-142618-504.png", Slot: 1, Position: 4, InitialLevel: 0, FinalLevel: 3, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "4.28", Final: "9.15"}, {Name: "防御加成", Init: "2.47%"}, {Name: "暴击伤害", Init: "3.96%"}, {Name: "效果抵抗", Init: "3.38%"},
	}},
	{File: "MuMu-20260708-142618-504.png", Slot: 2, Position: 6, InitialLevel: 0, FinalLevel: 3, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "4.21"}, {Name: "防御加成", Init: "2.81%"}, {Name: "暴击", Init: "2.50%", Final: "5.16%"}, {Name: "效果命中", Init: "3.74%"},
	}},
	{File: "MuMu-20260708-152910-531.png", Slot: 1, Position: 4, InitialLevel: 0, FinalLevel: 3, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "4.28", Final: "9.15"}, {Name: "防御加成", Init: "2.47%"}, {Name: "暴击伤害", Init: "3.96%"}, {Name: "效果抵抗", Init: "3.38%"},
	}},
	{File: "MuMu-20260708-152910-531.png", Slot: 2, Position: 6, InitialLevel: 0, FinalLevel: 3, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "4.21"}, {Name: "防御加成", Init: "2.81%"}, {Name: "暴击", Init: "2.50%", Final: "5.16%"}, {Name: "效果命中", Init: "3.74%"},
	}},
	{File: "MuMu-20260708-153307-367.png", Slot: 1, Position: 4, InitialLevel: 3, FinalLevel: 6, Main: TrainingAttr{Name: "防御加成", Init: "19.00%", Final: "28.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "9.15"}, {Name: "防御加成", Init: "2.47%"}, {Name: "暴击伤害", Init: "3.96%"}, {Name: "效果抵抗", Init: "3.38%", Final: "7.02%"},
	}},
	{File: "MuMu-20260708-153307-367.png", Slot: 2, Position: 6, InitialLevel: 3, FinalLevel: 6, Main: TrainingAttr{Name: "防御加成", Init: "19.00%", Final: "28.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "4.21"}, {Name: "防御加成", Init: "2.81%", Final: "5.55%"}, {Name: "暴击", Init: "5.16%"}, {Name: "效果命中", Init: "3.74%"},
	}},
	{File: "MuMu-20260708-153716-954.png", Slot: 1, Position: 1, InitialLevel: 0, FinalLevel: 3, Main: TrainingAttr{Name: "攻击", Init: "81.00", Final: "162.00"}, Subs: []TrainingAttr{
		{Name: "防御加成", Init: "2.69%"}, {Name: "攻击加成", Init: "2.65%"}, {Name: "速度", Init: "2.65"}, {Name: "暴击", Init: "0", Final: "2.45%", New: true},
	}},
	{File: "MuMu-20260708-153716-954.png", Slot: 2, Position: 2, InitialLevel: 0, FinalLevel: 3, Main: TrainingAttr{Name: "生命加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "4.42"}, {Name: "攻击", Init: "22.36", Final: "48.49"},
	}},
	{File: "MuMu-20260708-153716-954.png", Slot: 3, Position: 3, InitialLevel: 0, FinalLevel: 3, Main: TrainingAttr{Name: "防御", Init: "14.00", Final: "32.00"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "4.01", Final: "8.78"}, {Name: "攻击", Init: "22.44"}, {Name: "生命加成", Init: "2.82%"}, {Name: "攻击加成", Init: "2.58%"},
	}},
	{File: "MuMu-20260708-153716-954.png", Slot: 4, Position: 4, InitialLevel: 0, FinalLevel: 3, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "生命加成", Init: "2.78%"}, {Name: "速度", Init: "2.50"}, {Name: "暴击伤害", Init: "3.99%"}, {Name: "效果抵抗", Init: "0", Final: "3.91%", New: true},
	}},
	{File: "MuMu-20260708-153716-954.png", Slot: 5, Position: 5, InitialLevel: 0, FinalLevel: 3, Main: TrainingAttr{Name: "生命", Init: "342.00", Final: "684.00"}, Subs: []TrainingAttr{
		{Name: "攻击", Init: "25.31"}, {Name: "生命加成", Init: "2.48%"}, {Name: "效果抵抗", Init: "0", Final: "3.42%", New: true},
	}},
	{File: "MuMu-20260708-153716-954.png", Slot: 6, Position: 6, InitialLevel: 0, FinalLevel: 3, Main: TrainingAttr{Name: "生命加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "生命加成", Init: "2.78%"}, {Name: "速度", Init: "2.75"}, {Name: "暴击伤害", Init: "3.22%", Final: "6.97%"},
	}},
	{File: "MuMu-20260708-154000-143.png", Slot: 1, Position: 2, InitialLevel: 3, FinalLevel: 12, Main: TrainingAttr{Name: "生命加成", Init: "19.00%", Final: "46.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "4.42", Final: "9.39"}, {Name: "攻击", Init: "48.49"}, {Name: "速度", Init: "0", Final: "5.49", New: true},
	}},
	{File: "MuMu-20260708-154000-143.png", Slot: 2, Position: 1, InitialLevel: 3, FinalLevel: 3, Main: TrainingAttr{Name: "攻击", Init: "162.00", Final: "162.00"}, Subs: []TrainingAttr{
		{Name: "防御加成", Init: "2.69%"}, {Name: "攻击加成", Init: "2.65%"}, {Name: "速度", Init: "2.65"}, {Name: "暴击", Init: "2.45%"},
	}},
	{File: "MuMu-20260708-154000-143.png", Slot: 3, Position: 3, InitialLevel: 3, FinalLevel: 3, Main: TrainingAttr{Name: "防御", Init: "32.00", Final: "32.00"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "8.78"}, {Name: "攻击", Init: "22.44"}, {Name: "生命加成", Init: "2.82%"}, {Name: "攻击加成", Init: "2.58%"},
	}},
	{File: "MuMu-20260708-154000-143.png", Slot: 4, Position: 4, InitialLevel: 3, FinalLevel: 3, Main: TrainingAttr{Name: "防御加成", Init: "19.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "生命加成", Init: "2.78%"}, {Name: "速度", Init: "2.50"}, {Name: "暴击伤害", Init: "3.99%"}, {Name: "效果抵抗", Init: "3.91%"},
	}},
	{File: "MuMu-20260708-154000-143.png", Slot: 5, Position: 5, InitialLevel: 3, FinalLevel: 3, Main: TrainingAttr{Name: "生命", Init: "684.00", Final: "684.00"}, Subs: []TrainingAttr{
		{Name: "攻击", Init: "25.31"}, {Name: "生命加成", Init: "2.48%"}, {Name: "效果抵抗", Init: "3.42%"},
	}},
	{File: "MuMu-20260708-154000-143.png", Slot: 6, Position: 6, InitialLevel: 3, FinalLevel: 3, Main: TrainingAttr{Name: "生命加成", Init: "19.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "生命加成", Init: "2.78%"}, {Name: "速度", Init: "2.75"}, {Name: "暴击伤害", Init: "6.97%"},
	}},
}

func main() {
	if err := runCLI(os.Args[1:]); err != nil {
		fatal(err)
	}
}

func runCLI(args []string) error {
	if len(args) == 0 {
		return runRecognize(nil)
	}
	switch args[0] {
	case "help", "-h", "--help":
		return runHelp(args[1:])
	case "export-model":
		return runExportModel(args[1:])
	case "recognize":
		return runRecognize(args[1:])
	default:
		if strings.HasPrefix(args[0], "-") {
			return runRecognize(args)
		}
		return fmt.Errorf("unknown command %q\n\n%s", args[0], rootUsageText())
	}
}

func runHelp(args []string) error {
	if len(args) == 0 {
		printRootUsage(os.Stdout)
		return nil
	}
	if len(args) > 1 {
		return fmt.Errorf("help accepts at most one command name")
	}
	switch args[0] {
	case "recognize":
		fs, _, _ := newRecognizeFlagSet(os.Stdout)
		fs.Usage()
	case "export-model":
		fs, _, _ := newExportModelFlagSet(os.Stdout)
		fs.Usage()
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], rootUsageText())
	}
	return nil
}

func runExportModel(args []string) error {
	fs, assetsDir, modelPath := newExportModelFlagSet(os.Stderr)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	model, err := TrainModel(*assetsDir)
	if err != nil {
		return err
	}
	if err := SaveTemplateModel(*modelPath, model); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "exported model: %s\n", *modelPath)
	return nil
}

func runRecognize(args []string) error {
	fs, input, modelPath := newRecognizeFlagSet(os.Stderr)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if err := applyRecognizePositionalInput(fs, input); err != nil {
		return err
	}
	var model *TemplateModel
	var err error
	if *modelPath != "" {
		model, err = LoadTemplateModel(*modelPath)
	} else {
		model, err = LoadEmbeddedTemplateModel()
	}
	if err != nil {
		return err
	}
	recognizer, err := NewRecognizer(model)
	if err != nil {
		return err
	}
	paths, err := collectInputs(*input)
	if err != nil {
		return err
	}
	results := make([]ImageResult, 0, len(paths))
	for _, p := range paths {
		img, err := loadPNG(p)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		souls, err := recognizer.Recognize(img)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		results = append(results, ImageResult{Image: filepath.ToSlash(p), Souls: souls})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

func newRecognizeFlagSet(out io.Writer) (*flag.FlagSet, *string, *string) {
	fs := flag.NewFlagSet("recognize", flag.ContinueOnError)
	fs.SetOutput(out)
	input := fs.String("input", defaultInputPath, "PNG image file or directory to recognize")
	modelPath := fs.String("model", "", "template model path; defaults to the embedded model")
	fs.Usage = func() {
		fmt.Fprint(out, recognizeUsageText())
		fs.PrintDefaults()
	}
	return fs, input, modelPath
}

func newExportModelFlagSet(out io.Writer) (*flag.FlagSet, *string, *string) {
	fs := flag.NewFlagSet("export-model", flag.ContinueOnError)
	fs.SetOutput(out)
	assetsDir := fs.String("assets", defaultInputPath, "directory containing labelled training screenshots")
	modelPath := fs.String("model", defaultModelPath, "output template model path")
	fs.Usage = func() {
		fmt.Fprint(out, exportModelUsageText())
		fs.PrintDefaults()
	}
	return fs, assetsDir, modelPath
}

func applyRecognizePositionalInput(fs *flag.FlagSet, input *string) error {
	args := fs.Args()
	switch len(args) {
	case 0:
		return nil
	case 1:
		if flagWasSet(fs, "input") {
			return errors.New("use either -input or a positional input path, not both")
		}
		*input = args[0]
		return nil
	default:
		return fmt.Errorf("recognize accepts at most one positional input path, got %d", len(args))
	}
}

func flagWasSet(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func printRootUsage(out io.Writer) {
	fmt.Fprint(out, rootUsageText())
}

func rootUsageText() string {
	return `ocr_soul recognizes Onmyoji soul enhancement result screenshots.

Usage:
  ocr_soul [recognize] [flags]
  ocr_soul recognize [flags] [image-or-dir]
  ocr_soul export-model [flags]
  ocr_soul help [command]

Commands:
  recognize     Recognize a PNG file or every PNG in a directory and write JSON to stdout.
  export-model  Rebuild the template model from labelled screenshots.
  help          Show this help, or command-specific help.

Supported parameters:
  recognize:
    -input string    PNG image file or directory to recognize. Default: assets.
    -model string    Template model path. Empty value uses the embedded model.

  export-model:
    -assets string   Directory containing labelled training screenshots. Default: assets.
    -model string    Output template model path. Default: models/soul_ocr_model.gob.

Examples:
  ocr_soul recognize -input assets
  ocr_soul recognize assets\sample.png
  ocr_soul -input assets
  ocr_soul export-model -assets assets -model models\soul_ocr_model.gob

`
}

func recognizeUsageText() string {
	return `Recognize a PNG file or every PNG in a directory.

Usage:
  ocr_soul recognize [flags]
  ocr_soul recognize [flags] [image-or-dir]
  ocr_soul [flags]

Output:
  JSON is written to stdout. Errors and progress messages are written to stderr.

Flags:
`
}

func exportModelUsageText() string {
	return `Rebuild the template model from labelled training screenshots.

Usage:
  ocr_soul export-model [flags]

Output:
  A gob template model is written to the path provided by -model.

Flags:
`
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func collectInputs(input string) ([]string, error) {
	info, err := os.Stat(input)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{input}, nil
	}
	var paths []string
	err = filepath.WalkDir(input, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".png") {
			paths = append(paths, path)
		}
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

func loadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func SaveTemplateModel(path string, model *TemplateModel) error {
	if err := validateTemplateModel(model, nil); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gob.NewEncoder(f).Encode(model)
}

func LoadTemplateModel(path string) (*TemplateModel, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var model TemplateModel
	if err := gob.NewDecoder(f).Decode(&model); err != nil {
		return nil, err
	}
	if err := validateTemplateModel(&model, nil); err != nil {
		return nil, err
	}
	return &model, nil
}

func LoadEmbeddedTemplateModel() (*TemplateModel, error) {
	var model TemplateModel
	if err := gob.NewDecoder(bytes.NewReader(embeddedModel)).Decode(&model); err != nil {
		return nil, fmt.Errorf("load embedded model: %w", err)
	}
	if err := validateTemplateModel(&model, nil); err != nil {
		return nil, err
	}
	return &model, nil
}

func TrainModel(assetsDir string) (*TemplateModel, error) {
	model := &TemplateModel{Version: 1}
	seenLabels := map[string]bool{}
	for _, sample := range trainingSet {
		img, err := loadPNG(filepath.Join(assetsDir, sample.File))
		if err != nil {
			return nil, fmt.Errorf("load training image %s: %w", sample.File, err)
		}
		slot := scaleSlot(img.Bounds(), sample.Slot)
		mainRow := rowRect(img.Bounds(), slot, 0)
		mainLabelMask := labelMask(crop(img, labelRect(mainRow)))
		if mainLabelMask.Count() > 0 {
			model.LabelTemplates = append(model.LabelTemplates, LabelTemplate{Name: sample.Main.Name, Mask: mainLabelMask})
			seenLabels[sample.Main.Name] = true
		}
		if sample.Position > 0 {
			model.PositionTemplates = append(model.PositionTemplates, PositionTemplate{Position: sample.Position, Mask: positionMask(img, slot)})
		}
		model.addDigitsFromInteger(img, initialLevelRect(img.Bounds(), slot), sample.InitialLevel)
		model.addDigitsFromInteger(img, finalLevelRect(img.Bounds(), slot), sample.FinalLevel)
		model.addDigitsFromValue(img, slot, 0, true, sample.Main.Init)
		model.addDigitsFromValue(img, slot, 0, false, sample.Main.Final)
		for i, attr := range sample.Subs {
			rowIndex := i + 1
			rr := rowRect(img.Bounds(), slot, rowIndex)
			mask := labelMask(crop(img, labelRect(rr)))
			if mask.Count() > 0 {
				model.LabelTemplates = append(model.LabelTemplates, LabelTemplate{Name: attr.Name, Mask: mask})
				seenLabels[attr.Name] = true
			}
			model.addDigitsFromValue(img, slot, rowIndex, true, attr.Init)
			if attr.Final != "" {
				model.addDigitsFromValue(img, slot, rowIndex, false, attr.Final)
			}
		}
	}
	if err := validateTemplateModel(model, seenLabels); err != nil {
		return nil, err
	}
	return model, nil
}

func NewRecognizer(model *TemplateModel) (*Recognizer, error) {
	if err := validateTemplateModel(model, nil); err != nil {
		return nil, err
	}
	return &Recognizer{
		LabelTemplates:    model.LabelTemplates,
		DigitTemplates:    model.DigitTemplates,
		PositionTemplates: model.PositionTemplates,
	}, nil
}

func validateTemplateModel(model *TemplateModel, seenLabels map[string]bool) error {
	if model == nil {
		return errors.New("template model is nil")
	}
	if len(seenLabels) < 10 {
		if seenLabels != nil {
			return errors.New("not enough attribute label templates were learned")
		}
		unique := map[string]bool{}
		for _, tmpl := range model.LabelTemplates {
			unique[tmpl.Name] = true
		}
		if len(unique) < 10 {
			return errors.New("not enough attribute label templates in model")
		}
	}
	if len(model.PositionTemplates) < 6 {
		return errors.New("not enough position templates in model")
	}
	for d := '0'; d <= '9'; d++ {
		found := false
		for _, tmpl := range model.DigitTemplates {
			if tmpl.Char == d {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("digit template %q is missing from model", string(d))
		}
	}
	return nil
}

func (m *TemplateModel) addDigitsFromInteger(img image.Image, rect Rect, value int) {
	text := fmt.Sprintf("%d", value)
	comps := integerComponents(crop(img, rect), "white")
	if len(comps) != len(text) {
		return
	}
	for i, ch := range text {
		m.DigitTemplates = append(m.DigitTemplates, DigitTemplate{Char: ch, Mask: normalizeMask(comps[i].Mask, 12, 18)})
	}
}

func (m *TemplateModel) addDigitsFromValue(img image.Image, slot Point, row int, initial bool, text string) {
	maskKind := "white"
	rect := initialValueRect(rowRect(img.Bounds(), slot, row))
	if !initial {
		maskKind = "gold"
		rect = finalValueRect(rowRect(img.Bounds(), slot, row))
	}
	comps := valueComponents(crop(img, rect), maskKind)
	digits := digitRunes(text)
	if len(digits) == 0 {
		return
	}
	beforeCount := strings.IndexRune(text, '.')
	if beforeCount < 0 {
		return
	}
	dot := findDot(comps)
	if dot < 0 {
		return
	}
	afterCount := 2
	before := componentsBefore(comps, comps[dot])
	after := componentsAfter(comps, comps[dot])
	if len(before) < beforeCount || len(after) < afterCount {
		return
	}
	selected := append([]Component{}, before[len(before)-beforeCount:]...)
	selected = append(selected, after[:afterCount]...)
	if len(selected) != len(digits) {
		return
	}
	for i, ch := range digits {
		m.DigitTemplates = append(m.DigitTemplates, DigitTemplate{Char: ch, Mask: normalizeMask(selected[i].Mask, 12, 18)})
	}
}

func digitRunes(s string) []rune {
	out := []rune{}
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			out = append(out, ch)
		}
	}
	return out
}

func (r *Recognizer) Recognize(img image.Image) ([]SoulResult, error) {
	var results []SoulResult
	for i := range slots {
		slot := scaleSlot(img.Bounds(), i+1)
		if !hasSoul(img, slot) {
			continue
		}
		position := r.recognizePosition(img, slot)
		initialLevel := r.recognizeInteger(crop(img, initialLevelRect(img.Bounds(), slot)), "white")
		finalLevel := r.recognizeInteger(crop(img, finalLevelRect(img.Bounds(), slot)), "white")
		mainName := r.recognizeLabel(crop(img, labelRect(rowRect(img.Bounds(), slot, 0))))
		mainInit := normalizeAttributeValue(mainName, r.recognizeValue(crop(img, initialValueRect(rowRect(img.Bounds(), slot, 0))), "white"))
		mainFinalCandidate := normalizeAttributeValue(mainName, r.recognizeValue(crop(img, finalValueRect(rowRect(img.Bounds(), slot, 0))), "gold"))
		if mainInit == "" && mainName != "" {
			mainInit = normalizeAttributeValue(mainName, "10.00")
		}
		mainFinal := mainFinalCandidate
		if mainFinal == "" {
			mainFinal = mainInit
		}
		main := AttributeValue{Name: mainName, InitialValue: mainInit, FinalValue: mainFinal, Upgraded: mainInit != mainFinal}
		subs := []AttributeValue{}
		upgradedAttributes := []AttributeValue{}
		for row := 1; row <= 4; row++ {
			rr := rowRect(img.Bounds(), slot, row)
			labelCrop := crop(img, labelRect(rr))
			if labelMask(labelCrop).Count() < 16 {
				continue
			}
			name := r.recognizeLabel(labelCrop)
			initValue := normalizeAttributeValue(name, r.recognizeValue(crop(img, initialValueRect(rr)), "white"))
			finalValue := initValue
			finalCandidate := normalizeAttributeValue(name, r.recognizeValue(crop(img, finalValueRect(rr)), "gold"))
			isNew := hasNewMarker(img, rr)
			if isNew {
				initValue = "0"
			}
			isUpgraded := isNew || (finalCandidate != "" && finalCandidate != initValue)
			if isUpgraded {
				if finalCandidate != "" {
					finalValue = finalCandidate
				}
			}
			attr := AttributeValue{Name: name, InitialValue: initValue, FinalValue: finalValue, Upgraded: isUpgraded}
			subs = append(subs, attr)
			if isUpgraded {
				upgradedAttributes = append(upgradedAttributes, attr)
			}
		}
		finalSubs := make([]AttributeValue, len(subs))
		for j, sub := range subs {
			finalSubs[j] = AttributeValue{Name: sub.Name, FinalValue: sub.FinalValue, Upgraded: sub.Upgraded}
		}
		results = append(results, SoulResult{
			Index:              len(results) + 1,
			Position:           position,
			InitialLevel:       initialLevel,
			FinalLevel:         finalLevel,
			MainAttribute:      main,
			SubAttributeCount:  len(subs),
			SubAttributes:      subs,
			UpgradedAttributes: upgradedAttributes,
			FinalAttributes: FinalAttributes{
				Main: AttributeValue{Name: main.Name, FinalValue: main.FinalValue, Upgraded: main.Upgraded},
				Subs: finalSubs,
			},
		})
	}
	return results, nil
}

func scaleSlot(bounds image.Rectangle, slotNum int) Point {
	base := slots[slotNum-1]
	w := bounds.Dx()
	h := bounds.Dy()
	return Point{X: scale(base.X, w, 1280), Y: scale(base.Y, h, 720)}
}

func rowRect(bounds image.Rectangle, slot Point, row int) Rect {
	base := rowRects[row]
	return Rect{
		X0: slot.X + scale(base.X0, bounds.Dx(), 1280),
		Y0: slot.Y + scale(base.Y0, bounds.Dy(), 720),
		X1: slot.X + scale(base.X1, bounds.Dx(), 1280),
		Y1: slot.Y + scale(base.Y1, bounds.Dy(), 720),
	}
}

func scale(v, actual, base int) int {
	return int(math.Round(float64(v) * float64(actual) / float64(base)))
}

func labelRect(row Rect) Rect {
	return Rect{X0: row.X0, Y0: row.Y0, X1: row.X0 + (row.X1-row.X0)*36/100, Y1: row.Y1}
}

func initialValueRect(row Rect) Rect {
	w := row.X1 - row.X0
	return Rect{X0: row.X0 + w*38/100, Y0: row.Y0, X1: row.X0 + w*68/100, Y1: row.Y1}
}

func finalValueRect(row Rect) Rect {
	w := row.X1 - row.X0
	return Rect{X0: row.X0 + w*72/100, Y0: row.Y0, X1: row.X0 + w*99/100, Y1: row.Y1}
}

func initialLevelRect(bounds image.Rectangle, slot Point) Rect {
	return Rect{
		X0: slot.X + scale(122, bounds.Dx(), 1280),
		Y0: slot.Y + scale(35, bounds.Dy(), 720),
		X1: slot.X + scale(150, bounds.Dx(), 1280),
		Y1: slot.Y + scale(62, bounds.Dy(), 720),
	}
}

func finalLevelRect(bounds image.Rectangle, slot Point) Rect {
	return Rect{
		X0: slot.X + scale(208, bounds.Dx(), 1280),
		Y0: slot.Y + scale(35, bounds.Dy(), 720),
		X1: slot.X + scale(250, bounds.Dx(), 1280),
		Y1: slot.Y + scale(62, bounds.Dy(), 720),
	}
}

func positionRect(bounds image.Rectangle, slot Point) Rect {
	return Rect{
		X0: slot.X - scale(25, bounds.Dx(), 1280),
		Y0: slot.Y - scale(10, bounds.Dy(), 720),
		X1: slot.X + scale(95, bounds.Dx(), 1280),
		Y1: slot.Y + scale(100, bounds.Dy(), 720),
	}
}

func crop(img image.Image, r Rect) image.Image {
	b := img.Bounds()
	rr := image.Rect(r.X0, r.Y0, r.X1, r.Y1).Intersect(b)
	return img.(interface {
		SubImage(image.Rectangle) image.Image
	}).SubImage(rr)
}

func hasSoul(img image.Image, slot Point) bool {
	rect := Rect{X0: slot.X + 10, Y0: slot.Y + 70, X1: slot.X + 260, Y1: slot.Y + 103}
	mask := anyTextMask(crop(img, rect))
	return mask.Count() > 80
}

func hasGoldValue(img image.Image) bool {
	return goldMask(img).Count() > 20
}

func hasNewMarker(img image.Image, row Rect) bool {
	r := Rect{X0: row.X0 - 28, Y0: row.Y0 - 2, X1: row.X0 + 10, Y1: row.Y1 + 2}
	return newMarkerMask(crop(img, r)).Count() > 20
}

func normalizeAttributeValue(name, value string) string {
	value = strings.TrimSuffix(value, "%")
	if value == "" {
		return ""
	}
	if attributeUsesPercent(name) {
		return value + "%"
	}
	return value
}

func attributeUsesPercent(name string) bool {
	switch name {
	case "攻击加成", "生命加成", "防御加成", "暴击", "暴击伤害", "效果命中", "效果抵抗":
		return true
	default:
		return false
	}
}

func (r *Recognizer) recognizeLabel(img image.Image) string {
	mask := labelMask(img)
	bestName := ""
	bestScore := math.MaxFloat64
	for _, tmpl := range r.LabelTemplates {
		score := maskDistance(mask, tmpl.Mask)
		if score < bestScore {
			bestScore = score
			bestName = tmpl.Name
		}
	}
	return bestName
}

func (r *Recognizer) recognizePosition(img image.Image, slot Point) int {
	mask := positionMask(img, slot)
	bestPosition := 0
	bestScore := math.MaxFloat64
	for _, tmpl := range r.PositionTemplates {
		score := maskDistance(mask, tmpl.Mask)
		if score < bestScore {
			bestScore = score
			bestPosition = tmpl.Position
		}
	}
	return bestPosition
}

func (r *Recognizer) recognizeInteger(img image.Image, maskKind string) int {
	comps := integerComponents(img, maskKind)
	if len(comps) == 0 {
		return 0
	}
	text := ""
	for _, c := range comps {
		text += string(r.recognizeDigit(c.Mask))
	}
	var value int
	for _, ch := range text {
		if ch < '0' || ch > '9' {
			return 0
		}
		value = value*10 + int(ch-'0')
	}
	return value
}

func (r *Recognizer) recognizeValue(img image.Image, maskKind string) string {
	comps := valueComponents(img, maskKind)
	dot := findDot(comps)
	if dot < 0 {
		return ""
	}
	before := componentsBefore(comps, comps[dot])
	after := componentsAfter(comps, comps[dot])
	if len(before) == 0 || len(after) < 2 {
		return ""
	}
	intPart := ""
	for _, c := range before {
		intPart += string(r.recognizeDigit(c.Mask))
	}
	dec0 := r.recognizeDigit(after[0].Mask)
	dec1 := r.recognizeDigit(after[1].Mask)
	out := fmt.Sprintf("%s.%c%c", intPart, dec0, dec1)
	if hasTrailingPercent(after[1], after[2:]) {
		out += "%"
	}
	return out
}

func (r *Recognizer) recognizeDigit(mask BinaryImage) rune {
	norm := normalizeMask(mask, 12, 18)
	best := rune('0')
	bestScore := math.MaxFloat64
	for _, tmpl := range r.DigitTemplates {
		score := maskDistance(norm, tmpl.Mask)
		if score < bestScore {
			bestScore = score
			best = tmpl.Char
		}
	}
	return best
}

func componentsBefore(comps []Component, dot Component) []Component {
	var out []Component
	for _, c := range comps {
		if c.X1 <= dot.X0 && c.Area >= 8 {
			out = append(out, c)
		}
	}
	sortComponents(out)
	return out
}

func componentsAfter(comps []Component, dot Component) []Component {
	var out []Component
	for _, c := range comps {
		if c.X0 >= dot.X1 && c.Area >= 8 {
			out = append(out, c)
		}
	}
	sortComponents(out)
	return out
}

func hasTrailingPercent(secondDecimal Component, rest []Component) bool {
	area := 0
	for _, c := range rest {
		if c.X0 >= secondDecimal.X1-1 {
			area += c.Area
		}
	}
	return area >= 16
}

func findDot(comps []Component) int {
	best := -1
	for i, c := range comps {
		w := c.X1 - c.X0
		h := c.Y1 - c.Y0
		if w <= 4 && h <= 5 && c.Area <= 10 && c.Y0 > 14 {
			if best < 0 || c.X0 < comps[best].X0 {
				best = i
			}
		}
	}
	return best
}

func sortComponents(comps []Component) {
	sort.Slice(comps, func(i, j int) bool {
		if comps[i].X0 == comps[j].X0 {
			return comps[i].Y0 < comps[j].Y0
		}
		return comps[i].X0 < comps[j].X0
	})
}

func valueComponents(img image.Image, maskKind string) []Component {
	var mask BinaryImage
	if maskKind == "gold" {
		mask = goldMask(img)
	} else {
		mask = whiteMask(img)
	}
	comps := splitWideDigitComponents(connectedComponents(mask))
	filtered := comps[:0]
	for _, c := range comps {
		h := c.Y1 - c.Y0
		if c.Area < 3 || h < 2 {
			continue
		}
		if (c.X0 <= 1 || c.X1 >= mask.W-1) && c.Area < 12 {
			continue
		}
		filtered = append(filtered, c)
	}
	sortComponents(filtered)
	return filtered
}

func splitWideDigitComponents(comps []Component) []Component {
	out := make([]Component, 0, len(comps))
	for _, c := range comps {
		w := c.X1 - c.X0
		h := c.Y1 - c.Y0
		if w < 16 || h > 20 {
			out = append(out, c)
			continue
		}
		parts := int(math.Round(float64(w) / 9.0))
		if parts < 2 || parts > 3 {
			out = append(out, c)
			continue
		}
		for i := 0; i < parts; i++ {
			x0 := i * w / parts
			x1 := (i + 1) * w / parts
			part := subMaskComponent(c, x0, x1)
			if part.Area >= 3 {
				out = append(out, part)
			}
		}
	}
	return out
}

func subMaskComponent(c Component, localX0, localX1 int) Component {
	if localX0 < 0 {
		localX0 = 0
	}
	if localX1 > c.Mask.W {
		localX1 = c.Mask.W
	}
	points := [][2]int{}
	x0, y0, x1, y1 := localX1, c.Mask.H, localX0, 0
	for y := 0; y < c.Mask.H; y++ {
		for x := localX0; x < localX1; x++ {
			if !c.Mask.P[y*c.Mask.W+x] {
				continue
			}
			points = append(points, [2]int{x, y})
			if x < x0 {
				x0 = x
			}
			if y < y0 {
				y0 = y
			}
			if x+1 > x1 {
				x1 = x + 1
			}
			if y+1 > y1 {
				y1 = y + 1
			}
		}
	}
	if len(points) == 0 {
		return Component{}
	}
	mask := BinaryImage{W: x1 - x0, H: y1 - y0, P: make([]bool, (x1-x0)*(y1-y0))}
	for _, p := range points {
		mask.P[(p[1]-y0)*mask.W+(p[0]-x0)] = true
	}
	return Component{
		X0:   c.X0 + x0,
		Y0:   c.Y0 + y0,
		X1:   c.X0 + x1,
		Y1:   c.Y0 + y1,
		Area: len(points),
		Mask: mask,
	}
}

func integerComponents(img image.Image, maskKind string) []Component {
	var mask BinaryImage
	if maskKind == "gold" {
		mask = goldMask(img)
	} else {
		mask = whiteMask(img)
	}
	comps := connectedComponents(mask)
	filtered := comps[:0]
	for _, c := range comps {
		w := c.X1 - c.X0
		h := c.Y1 - c.Y0
		if c.Area < 8 || h < 8 || w > 14 {
			continue
		}
		filtered = append(filtered, c)
	}
	sortComponents(filtered)
	return filtered
}

func positionMask(img image.Image, slot Point) BinaryImage {
	return makeMask(crop(img, positionRect(img.Bounds(), slot)), func(c color.Color) bool {
		r, g, b := rgb(c)
		return r > 165 && g > 110 && b < 105 && r > g && g > b+25
	})
}

func whiteMask(img image.Image) BinaryImage {
	return makeMask(img, func(c color.Color) bool {
		r, g, b := rgb(c)
		return r > 120 && g > 105 && b > 85 && max3(r, g, b)-min3(r, g, b) < 110 && !isGoldRGB(r, g, b)
	})
}

func goldMask(img image.Image) BinaryImage {
	return makeMask(img, func(c color.Color) bool {
		r, g, b := rgb(c)
		return isGoldRGB(r, g, b)
	})
}

func labelMask(img image.Image) BinaryImage {
	return makeMask(img, func(c color.Color) bool {
		r, g, b := rgb(c)
		return r > 110 && g > 100 && b > 80 && max3(r, g, b)-min3(r, g, b) < 110 && !isGoldRGB(r, g, b)
	})
}

func anyTextMask(img image.Image) BinaryImage {
	return makeMask(img, func(c color.Color) bool {
		r, g, b := rgb(c)
		return isGoldRGB(r, g, b) || (r > 110 && g > 100 && b > 80 && max3(r, g, b)-min3(r, g, b) < 120)
	})
}

func newMarkerMask(img image.Image) BinaryImage {
	return makeMask(img, func(c color.Color) bool {
		r, g, b := rgb(c)
		return g > 130 && r < 80 && b < 90 && g > r+80 && g > b+80
	})
}

func makeMask(img image.Image, keep func(color.Color) bool) BinaryImage {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	out := BinaryImage{W: w, H: h, P: make([]bool, w*h)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			out.P[y*w+x] = keep(img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return out
}

func connectedComponents(mask BinaryImage) []Component {
	seen := make([]bool, len(mask.P))
	var out []Component
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, -1}, {1, -1}, {-1, 1}}
	for y := 0; y < mask.H; y++ {
		for x := 0; x < mask.W; x++ {
			idx := y*mask.W + x
			if !mask.P[idx] || seen[idx] {
				continue
			}
			stack := [][2]int{{x, y}}
			seen[idx] = true
			points := [][2]int{}
			x0, y0, x1, y1 := x, y, x+1, y+1
			for len(stack) > 0 {
				p := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				px, py := p[0], p[1]
				points = append(points, p)
				if px < x0 {
					x0 = px
				}
				if py < y0 {
					y0 = py
				}
				if px+1 > x1 {
					x1 = px + 1
				}
				if py+1 > y1 {
					y1 = py + 1
				}
				for _, d := range dirs {
					nx, ny := px+d[0], py+d[1]
					if nx < 0 || ny < 0 || nx >= mask.W || ny >= mask.H {
						continue
					}
					ni := ny*mask.W + nx
					if mask.P[ni] && !seen[ni] {
						seen[ni] = true
						stack = append(stack, [2]int{nx, ny})
					}
				}
			}
			compMask := BinaryImage{W: x1 - x0, H: y1 - y0, P: make([]bool, (x1-x0)*(y1-y0))}
			for _, p := range points {
				compMask.P[(p[1]-y0)*compMask.W+(p[0]-x0)] = true
			}
			out = append(out, Component{X0: x0, Y0: y0, X1: x1, Y1: y1, Area: len(points), Mask: compMask})
		}
	}
	return out
}

func normalizeMask(src BinaryImage, w, h int) BinaryImage {
	dst := BinaryImage{W: w, H: h, P: make([]bool, w*h)}
	if src.W == 0 || src.H == 0 {
		return dst
	}
	for y := 0; y < h; y++ {
		sy := int(float64(y) * float64(src.H) / float64(h))
		if sy >= src.H {
			sy = src.H - 1
		}
		for x := 0; x < w; x++ {
			sx := int(float64(x) * float64(src.W) / float64(w))
			if sx >= src.W {
				sx = src.W - 1
			}
			dst.P[y*w+x] = src.P[sy*src.W+sx]
		}
	}
	return dst
}

func maskDistance(a, b BinaryImage) float64 {
	w := max(a.W, b.W)
	h := max(a.H, b.H)
	if w == 0 || h == 0 {
		return math.MaxFloat64
	}
	diff := 0
	total := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			av := maskAt(a, x, y, w, h)
			bv := maskAt(b, x, y, w, h)
			if av || bv {
				total++
				if av != bv {
					diff++
				}
			}
		}
	}
	if total == 0 {
		return math.MaxFloat64
	}
	sizePenalty := math.Abs(float64(a.W-b.W))*0.02 + math.Abs(float64(a.H-b.H))*0.02
	return float64(diff)/float64(total) + sizePenalty
}

func maskAt(m BinaryImage, x, y, w, h int) bool {
	if m.W == 0 || m.H == 0 {
		return false
	}
	mx := int(float64(x) * float64(m.W) / float64(w))
	my := int(float64(y) * float64(m.H) / float64(h))
	if mx >= m.W {
		mx = m.W - 1
	}
	if my >= m.H {
		my = m.H - 1
	}
	return m.P[my*m.W+mx]
}

func (m BinaryImage) Count() int {
	n := 0
	for _, v := range m.P {
		if v {
			n++
		}
	}
	return n
}

func rgb(c color.Color) (int, int, int) {
	r, g, b, _ := c.RGBA()
	return int(r >> 8), int(g >> 8), int(b >> 8)
}

func isGoldRGB(r, g, b int) bool {
	return r > 135 && g > 75 && b < 110 && r > g+15
}

func max3(a, b, c int) int {
	return max(max(a, b), c)
}

func min3(a, b, c int) int {
	return min(min(a, b), c)
}
