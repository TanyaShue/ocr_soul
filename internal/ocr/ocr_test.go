package ocrsoul

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadModelAndRecognize(t *testing.T) {
	root := filepath.Join("..", "..")
	model, err := LoadTemplateModel(filepath.Join(root, "recognize", "models", "soul_ocr_model.gob"))
	if err != nil {
		t.Fatal(err)
	}
	recognizer, err := NewRecognizer(model)
	if err != nil {
		t.Fatal(err)
	}
	img, err := LoadPNG(filepath.Join(root, "test", "MuMu-20260708-165639-194.png"))
	if err != nil {
		t.Fatal(err)
	}
	souls, err := recognizer.Recognize(img)
	if err != nil {
		t.Fatal(err)
	}
	if len(souls) == 0 {
		t.Fatal("want at least one recognized soul")
	}
}

func TestDecodeTemplateModel(t *testing.T) {
	path := filepath.Join("..", "..", "recognize", "models", "soul_ocr_model.gob")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeTemplateModel(data); err != nil {
		t.Fatal(err)
	}
}
