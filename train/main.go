package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	ocr "ocr_soul/internal/ocr"
	"ocr_soul/internal/trainer"
	"ocr_soul/train/selectedv2data"
	"ocr_soul/train/trainingdata"
)

func main() {
	assetsDir := flag.String("assets", "train/assets", "directory containing labelled training screenshots")
	modelPath := flag.String("model", "recognize/models/soul_ocr_model.gob", "output template model path")
	recognitionType := flag.String("type", "upgrade", "model type: upgrade or selected")
	flag.Parse()
	if *recognitionType == "selected" && !flagWasSet("model") {
		*modelPath = "recognize/models/selected_soul_ocr_model.gob"
	}

	var model *ocr.TemplateModel
	var err error
	switch *recognitionType {
	case "upgrade":
		model, err = trainer.Train(*assetsDir, trainingdata.Samples)
	case "selected":
		base, loadErr := ocr.LoadTemplateModel("recognize/models/soul_ocr_model.gob")
		if loadErr != nil {
			err = fmt.Errorf("load upgrade model for complete soul types: %w", loadErr)
		} else {
			model, err = trainer.TrainSelected(filepath.Join(*assetsDir, "selected-v2"), selectedv2data.Samples, base)
		}
	default:
		err = fmt.Errorf("unknown model type %q; use upgrade or selected", *recognitionType)
	}
	if err == nil {
		err = ocr.SaveTemplateModel(*modelPath, model)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "exported model:", *modelPath)
}

func flagWasSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) { found = found || f.Name == name })
	return found
}
