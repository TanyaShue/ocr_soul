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
	if souls[0].SubAttributes[3].InitialValue != "0" {
		t.Fatalf("new attribute initial value: want 0, got %q", souls[0].SubAttributes[3].InitialValue)
	}
	if len(souls[0].UpgradedAttributes) != 1 || souls[0].UpgradedAttributes[0].Name != "暴击" {
		t.Fatalf("new attribute should be included in upgraded_attributes: %#v", souls[0].UpgradedAttributes)
	}

	img, err = loadPNG(filepath.Join("assets", "MuMu-20260708-154000-143.png"))
	if err != nil {
		t.Fatal(err)
	}
	souls, err = r.Recognize(img)
	if err != nil {
		t.Fatal(err)
	}
	if souls[0].InitialLevel != 3 || souls[0].FinalLevel != 12 {
		t.Fatalf("first soul level: want 3 -> 12, got %d -> %d", souls[0].InitialLevel, souls[0].FinalLevel)
	}
	last := souls[5]
	if last.InitialLevel != 3 || last.FinalLevel != 3 {
		t.Fatalf("last soul level: want 3 -> 3, got %d -> %d", last.InitialLevel, last.FinalLevel)
	}
	if last.MainAttribute.FinalValue != "19.00%" || last.MainAttribute.Upgraded {
		t.Fatalf("unupgraded main final/upgraded mismatch: %#v", last.MainAttribute)
	}
	if len(last.UpgradedAttributes) != 0 {
		t.Fatalf("unupgraded soul should not expose upgraded attributes: %#v", last.UpgradedAttributes)
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
	if len(souls) != 6 || souls[0].FinalLevel != 12 || souls[5].MainAttribute.FinalValue != "19.00%" {
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

func TestJSONOutputDoesNotContainSingularUpgradedAttribute(t *testing.T) {
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
	data, err := json.Marshal(souls)
	if err != nil {
		t.Fatal(err)
	}
	output := string(data)
	if strings.Contains(output, `"upgraded_attribute"`) {
		t.Fatalf("JSON should only expose upgraded_attributes, got %s", output)
	}
	if !strings.Contains(output, `"upgraded_attributes"`) {
		t.Fatalf("JSON should expose upgraded_attributes, got %s", output)
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
		main := AttributeValue{Name: sample.Main.Name, InitialValue: sample.Main.Init, FinalValue: sample.Main.Final, Upgraded: sample.Main.Init != sample.Main.Final}
		subs := make([]AttributeValue, 0, len(sample.Subs))
		upgradedAttrs := []AttributeValue{}
		for _, sub := range sample.Subs {
			final := sub.Final
			if final == "" {
				final = sub.Init
			}
			init := sub.Init
			if sub.New {
				init = "0"
			}
			attr := AttributeValue{Name: sub.Name, InitialValue: init, FinalValue: final, Upgraded: sub.Final != "" || sub.New}
			subs = append(subs, attr)
			if attr.Upgraded {
				upgradedAttrs = append(upgradedAttrs, attr)
			}
		}
		finalSubs := make([]AttributeValue, len(subs))
		for i, sub := range subs {
			finalSubs[i] = AttributeValue{Name: sub.Name, FinalValue: sub.FinalValue, Upgraded: sub.Upgraded}
		}
		out[sample.File] = append(out[sample.File], SoulResult{
			Index:              len(out[sample.File]) + 1,
			Position:           sample.Position,
			InitialLevel:       sample.InitialLevel,
			FinalLevel:         sample.FinalLevel,
			MainAttribute:      main,
			SubAttributeCount:  len(subs),
			SubAttributes:      subs,
			UpgradedAttributes: upgradedAttrs,
			FinalAttributes: FinalAttributes{
				Main: AttributeValue{Name: main.Name, FinalValue: main.FinalValue, Upgraded: main.Upgraded},
				Subs: finalSubs,
			},
		})
	}
	return out
}
