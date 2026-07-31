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
var embeddedModel []byte

func main() {
	input := flag.String("input", "test", "PNG image file or directory to recognize")
	modelPath := flag.String("model", "", "model path; empty uses the embedded model")
	flag.Parse()
	if flag.NArg() > 1 || (flag.NArg() == 1 && flagWasSet("input")) {
		fail(errors.New("use either -input or one positional input path"))
	}
	if flag.NArg() == 1 {
		*input = flag.Arg(0)
	}

	model, err := loadModel(*modelPath)
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
		souls, err := recognizer.Recognize(img)
		if err != nil {
			fail(fmt.Errorf("%s: %w", path, err))
		}
		results = append(results, ocr.ImageResult{Image: filepath.ToSlash(path), Souls: souls})
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(ocr.RecognitionOutput{SchemaVersion: 2, Results: results}); err != nil {
		fail(err)
	}
}

func loadModel(path string) (*ocr.TemplateModel, error) {
	if path != "" {
		return ocr.LoadTemplateModel(path)
	}
	return ocr.DecodeTemplateModel(embeddedModel)
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
