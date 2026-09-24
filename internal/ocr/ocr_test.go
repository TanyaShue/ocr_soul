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
	img, err := LoadPNG(filepath.Join(root, "train", "assets", "selected-v2", "MuMu-20260924-184447-961.png"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := recognizer.RecognizeSelected(img)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("want a recognized selected soul")
	}
	if got.Type != "海月火玉" || got.Position != 4 || got.Attributes.Main.Name != "生命加成" || got.Attributes.Main.Value != "10.00%" {
		t.Fatalf("unexpected selected soul: %+v", got)
	}
}

func TestSelectedLevelFromMainValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{name: "速度", value: "12.00", want: 0},
		{name: "速度", value: "15.00", want: 1},
		{name: "攻击加成", value: "28.00%", want: 6},
		{name: "攻击", value: "162.00", want: 3},
	}
	for _, tt := range tests {
		got, ok := selectedLevelFromMainValue(tt.name, tt.value)
		if !ok || got != tt.want {
			t.Errorf("selectedLevelFromMainValue(%q, %q) = %d, %v; want %d, true", tt.name, tt.value, got, ok, tt.want)
		}
	}
	if _, ok := selectedLevelFromMainValue("速度", "13.00"); ok {
		t.Error("unexpected level for invalid speed main value")
	}
}
