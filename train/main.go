package main

import (
	"flag"
	"fmt"
	"os"

	ocr "ocr_soul/internal/ocr"
	"ocr_soul/internal/trainer"
	"ocr_soul/train/trainingdata"
)

func main() {
	assetsDir := flag.String("assets", "train/assets", "directory containing labelled training screenshots")
	modelPath := flag.String("model", "recognize/models/soul_ocr_model.gob", "output template model path")
	flag.Parse()

	model, err := trainer.Train(*assetsDir, trainingdata.Samples)
	if err == nil {
		err = ocr.SaveTemplateModel(*modelPath, model)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "exported model:", *modelPath)
}
