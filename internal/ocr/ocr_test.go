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
	wantTypes := []string{"招财猫", "日女巳时", "珍珠", "油赤子", "青女房"}
	if len(souls) != len(wantTypes) {
		t.Fatalf("got %d souls, want %d", len(souls), len(wantTypes))
	}
	for i, want := range wantTypes {
		if souls[i].Type != want {
			t.Errorf("soul %d type = %q, want %q", i+1, souls[i].Type, want)
		}
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

func TestSelectedModelAndRecognition(t *testing.T) {
	root := filepath.Join("..", "..")
	model, err := LoadTemplateModel(filepath.Join(root, "recognize", "models", "selected_soul_ocr_model.gob"))
	if err != nil {
		t.Fatal(err)
	}
	recognizer, err := NewRecognizer(model)
	if err != nil {
		t.Fatal(err)
	}
	img, err := LoadPNG(filepath.Join(root, "train", "assets", "selected", "MuMu-20260731-100340-830.png"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := recognizer.RecognizeSelected(img)
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != "招财猫" || got.Level != 3 || got.Attributes.Main.Name != "防御加成" || got.Attributes.Main.Value != "19.00%" || len(got.Attributes.Subs) != 4 {
		t.Fatalf("unexpected selected soul: %+v", got)
	}
}
