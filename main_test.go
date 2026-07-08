package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestRecognizeTrainingImages(t *testing.T) {
	r, err := NewRecognizer("assets")
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

func expectedByFile() map[string][]SoulResult {
	out := map[string][]SoulResult{}
	for _, sample := range trainingSet {
		main := AttributeValue{Name: sample.Main.Name, InitialValue: sample.Main.Init, FinalValue: sample.Main.Final, Upgraded: sample.Main.Init != sample.Main.Final}
		subs := make([]AttributeValue, 0, len(sample.Subs))
		var upgraded *AttributeValue
		for _, sub := range sample.Subs {
			final := sub.Final
			if final == "" {
				final = sub.Init
			}
			attr := AttributeValue{Name: sub.Name, InitialValue: sub.Init, FinalValue: final, Upgraded: sub.Final != ""}
			subs = append(subs, attr)
			if attr.Upgraded {
				copyAttr := attr
				upgraded = &copyAttr
			}
		}
		finalSubs := make([]AttributeValue, len(subs))
		for i, sub := range subs {
			finalSubs[i] = AttributeValue{Name: sub.Name, FinalValue: sub.FinalValue, Upgraded: sub.Upgraded}
		}
		out[sample.File] = append(out[sample.File], SoulResult{
			Index:             len(out[sample.File]) + 1,
			MainAttribute:     main,
			SubAttributeCount: len(subs),
			SubAttributes:     subs,
			UpgradedAttribute: upgraded,
			FinalAttributes: FinalAttributes{
				Main: AttributeValue{Name: main.Name, FinalValue: main.FinalValue, Upgraded: main.Upgraded},
				Subs: finalSubs,
			},
		})
	}
	return out
}
