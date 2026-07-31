package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	ocr "ocr_soul/internal/ocr"
)

//go:embed models/soul_ocr_model.gob
var embeddedUpgradeModel []byte

//go:embed models/selected_soul_ocr_model.gob
var embeddedSelectedModel []byte

func main() {
	input := flag.String("input", "test", "PNG image file or directory to recognize")
	modelPath := flag.String("model", "", "model path; empty uses the embedded model")
	recognitionType := flag.String("type", "upgrade", "recognition type: upgrade or selected")
	flag.Parse()
	if flag.NArg() > 1 || (flag.NArg() == 1 && flagWasSet("input")) {
		fail(errors.New("use either -input or one positional input path"))
	}
	if flag.NArg() == 1 {
		*input = flag.Arg(0)
	}

	model, err := loadModel(*modelPath, *recognitionType)
	if err != nil {
		fail(err)
	}
	recognizer, err := ocr.NewRecognizer(model)
	if err != nil {
		fail(err)
	}
	paths, err := ocr.CollectInputs(*input)
	if err != nil {
		fail(err)
	}
	results := make([]ocr.ImageResult, 0, len(paths))
	for _, path := range paths {
		img, err := ocr.LoadPNG(path)
		if err != nil {
			fail(fmt.Errorf("%s: %w", path, err))
		}
		result := ocr.ImageResult{Image: filepath.ToSlash(path)}
		switch *recognitionType {
		case "upgrade":
			result.Souls, err = recognizer.Recognize(img)
		case "selected":
			result.SelectedSoul, err = recognizer.RecognizeSelected(img)
		default:
			err = fmt.Errorf("unknown recognition type %q; use upgrade or selected", *recognitionType)
		}
		if err != nil {
			fail(fmt.Errorf("%s: %w", path, err))
		}
		results = append(results, result)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(ocr.RecognitionOutput{SchemaVersion: 3, Results: results}); err != nil {
		fail(err)
	}
}

func loadModel(path, recognitionType string) (*ocr.TemplateModel, error) {
	if path != "" {
		return ocr.LoadTemplateModel(path)
	}
	switch recognitionType {
	case "upgrade":
		return ocr.DecodeTemplateModel(embeddedUpgradeModel)
	case "selected":
		return ocr.DecodeTemplateModel(embeddedSelectedModel)
	default:
		return nil, fmt.Errorf("unknown recognition type %q; use upgrade or selected", recognitionType)
	}
}

func flagWasSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) { found = found || f.Name == name })
	return found
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
