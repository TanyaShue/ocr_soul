package main

import (
	"encoding/json"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRecognizeTrainingImages(t *testing.T) {
	model, err := TrainModel("assets")
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewRecognizer(model)
	if err != nil {
		t.Fatal(err)
	}
	want := expectedByFile()
	for file, expected := range want {
		img, err := loadPNG(filepath.Join("assets", file))
		if err != nil {
			t.Fatal(err)
		}
		got, err := r.Recognize(img)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("%s mismatch\nwant: %#v\n got: %#v", file, expected, got)
		}
	}
}

func TestFocusedEnhancedRecognitionScenarios(t *testing.T) {
	model, err := TrainModel("assets")
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewRecognizer(model)
	if err != nil {
		t.Fatal(err)
	}

	img, err := loadPNG(filepath.Join("assets", "MuMu-20260708-153716-954.png"))
	if err != nil {
		t.Fatal(err)
	}
	souls, err := r.Recognize(img)
	if err != nil {
		t.Fatal(err)
	}
	if len(souls) != 6 {
		t.Fatalf("want 6 souls, got %d", len(souls))
	}
	for i, soul := range souls {
		if soul.Position != i+1 {
			t.Fatalf("soul %d position: want %d, got %d", i+1, i+1, soul.Position)
		}
	}
	if souls[0].Attributes.Subs[3].Initial != nil {
		t.Fatalf("new attribute initial value: want nil, got %q", *souls[0].Attributes.Subs[3].Initial)
	}

	img, err = loadPNG(filepath.Join("assets", "MuMu-20260708-154000-143.png"))
	if err != nil {
		t.Fatal(err)
	}
	souls, err = r.Recognize(img)
	if err != nil {
		t.Fatal(err)
	}
	if souls[0].Level.Initial != 3 || souls[0].Level.Final != 12 {
		t.Fatalf("first soul level: want 3 -> 12, got %d -> %d", souls[0].Level.Initial, souls[0].Level.Final)
	}
	last := souls[5]
	if last.Level.Initial != 3 || last.Level.Final != 3 {
		t.Fatalf("last soul level: want 3 -> 3, got %d -> %d", last.Level.Initial, last.Level.Final)
	}
	if last.Attributes.Main.Final != "19.00%" || *last.Attributes.Main.Initial != last.Attributes.Main.Final {
		t.Fatalf("unupgraded main initial/final mismatch: %#v", last.Attributes.Main)
	}
}

func TestExportedModelCanBeLoadedForRecognition(t *testing.T) {
	model, err := TrainModel("assets")
	if err != nil {
		t.Fatal(err)
	}
	modelPath := filepath.Join(t.TempDir(), "soul_ocr_model.gob")
	if err := SaveTemplateModel(modelPath, model); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadTemplateModel(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewRecognizer(loaded)
	if err != nil {
		t.Fatal(err)
	}
	img, err := loadPNG(filepath.Join("assets", "MuMu-20260708-154000-143.png"))
	if err != nil {
		t.Fatal(err)
	}
	souls, err := r.Recognize(img)
	if err != nil {
		t.Fatal(err)
	}
	if len(souls) != 6 || souls[0].Level.Final != 12 || souls[5].Attributes.Main.Final != "19.00%" {
		t.Fatalf("loaded model recognition mismatch: %#v", souls)
	}
}

func TestEmbeddedModelCanBeLoadedForRecognition(t *testing.T) {
	model, err := LoadEmbeddedTemplateModel()
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewRecognizer(model)
	if err != nil {
		t.Fatal(err)
	}
	img, err := loadPNG(filepath.Join("assets", "MuMu-20260708-153716-954.png"))
	if err != nil {
		t.Fatal(err)
	}
	souls, err := r.Recognize(img)
	if err != nil {
		t.Fatal(err)
	}
	for i, soul := range souls {
		if soul.Position != i+1 {
			t.Fatalf("embedded model position mismatch at index %d: got %d", i+1, soul.Position)
		}
	}
}

func TestJSONOutputUsesCompactSchema(t *testing.T) {
	model, err := LoadEmbeddedTemplateModel()
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewRecognizer(model)
	if err != nil {
		t.Fatal(err)
	}
	img, err := loadPNG(filepath.Join("assets", "MuMu-20260708-153716-954.png"))
	if err != nil {
		t.Fatal(err)
	}
	souls, err := r.Recognize(img)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(RecognitionOutput{
		SchemaVersion: 2,
		Results:       []ImageResult{{Image: "sample.png", Souls: souls}},
	})
	if err != nil {
		t.Fatal(err)
	}
	output := string(data)
	for _, redundant := range []string{
		`"sub_attribute_count"`,
		`"upgraded_attributes"`,
		`"final_attributes"`,
		`"upgraded"`,
	} {
		if strings.Contains(output, redundant) {
			t.Fatalf("JSON should not expose redundant field %s, got %s", redundant, output)
		}
	}
	for _, required := range []string{`"schema_version":2`, `"results"`, `"level"`, `"attributes"`, `"initial":null`} {
		if !strings.Contains(output, required) {
			t.Fatalf("JSON should expose %s, got %s", required, output)
		}
	}
}

func TestRootUsageListsCommandsAndSupportedParameters(t *testing.T) {
	usage := rootUsageText()
	for _, want := range []string{
		"ocr_soul recognize [flags] [image-or-dir]",
		"ocr_soul export-model [flags]",
		"-input string",
		"-assets string",
		"-model string",
	} {
		if !strings.Contains(usage, want) {
			t.Fatalf("root usage should contain %q, got:\n%s", want, usage)
		}
	}
}

func TestCommandUsageListsPurposeAndFlags(t *testing.T) {
	recognizeUsage := recognizeUsageText()
	for _, want := range []string{"Recognize a PNG file", "ocr_soul recognize [flags]", "Flags:"} {
		if !strings.Contains(recognizeUsage, want) {
			t.Fatalf("recognize usage should contain %q, got:\n%s", want, recognizeUsage)
		}
	}

	exportUsage := exportModelUsageText()
	for _, want := range []string{"Rebuild the template model", "ocr_soul export-model [flags]", "Flags:"} {
		if !strings.Contains(exportUsage, want) {
			t.Fatalf("export-model usage should contain %q, got:\n%s", want, exportUsage)
		}
	}
}

func TestRecognizePositionalInputRules(t *testing.T) {
	fs, input, _ := newRecognizeFlagSet(io.Discard)
	if err := fs.Parse([]string{"assets\\sample.png"}); err != nil {
		t.Fatal(err)
	}
	if err := applyRecognizePositionalInput(fs, input); err != nil {
		t.Fatal(err)
	}
	if *input != "assets\\sample.png" {
		t.Fatalf("positional input was not applied: %q", *input)
	}

	fs, input, _ = newRecognizeFlagSet(io.Discard)
	if err := fs.Parse([]string{"-input", "assets", "assets\\sample.png"}); err != nil {
		t.Fatal(err)
	}
	if err := applyRecognizePositionalInput(fs, input); err == nil || !strings.Contains(err.Error(), "either -input or a positional input path") {
		t.Fatalf("expected mixed input error, got %v", err)
	}
}

func expectedByFile() map[string][]SoulResult {
	out := map[string][]SoulResult{}
	for _, sample := range trainingSet {
		main := AttributeValue{Name: sample.Main.Name, Initial: stringPointer(sample.Main.Init), Final: sample.Main.Final}
		subs := make([]AttributeValue, 0, len(sample.Subs))
		for _, sub := range sample.Subs {
			final := sub.Final
			if final == "" {
				final = sub.Init
			}
			var initial *string
			if !sub.New {
				initial = stringPointer(sub.Init)
			}
			subs = append(subs, AttributeValue{Name: sub.Name, Initial: initial, Final: final})
		}
		out[sample.File] = append(out[sample.File], SoulResult{
			Order:    len(out[sample.File]) + 1,
			Position: sample.Position,
			Level:    LevelRange{Initial: sample.InitialLevel, Final: sample.FinalLevel},
			Attributes: Attributes{
				Main: main,
				Subs: subs,
			},
		})
	}
	return out
}
