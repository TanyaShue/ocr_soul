package trainer

import (
	"fmt"
	"path/filepath"

	ocr "ocr_soul/internal/ocr"
)

type Soul struct {
	File         string
	Slot         int
	Position     int
	InitialLevel int
	FinalLevel   int
	Main         Attribute
	Subs         []Attribute
}

type Attribute struct {
	Name  string
	Init  string
	Final string
	New   bool
}

func Train(assetsDir string, samples []Soul) (*ocr.TemplateModel, error) {
	model := &ocr.TemplateModel{Version: 1}
	seenLabels := map[string]bool{}
	for _, sample := range samples {
		img, err := ocr.LoadPNG(filepath.Join(assetsDir, sample.File))
		if err != nil {
			return nil, fmt.Errorf("load training image %s: %w", sample.File, err)
		}
		slot := ocr.ScaleSlot(img.Bounds(), sample.Slot)
		mainRow := ocr.RowRect(img.Bounds(), slot, 0)
		mainMask := ocr.LabelImageMask(ocr.LabelImage(img, mainRow))
		if mainMask.Count() > 0 {
			model.LabelTemplates = append(model.LabelTemplates, ocr.LabelTemplate{Name: sample.Main.Name, Mask: mainMask})
			seenLabels[sample.Main.Name] = true
		}
		if sample.Position > 0 {
			model.PositionTemplates = append(model.PositionTemplates, ocr.PositionTemplate{
				Position: sample.Position,
				Mask:     ocr.PositionImageMask(img, slot),
			})
		}
		model.LearnIntegerDigits(img, ocr.InitialLevelRect(img.Bounds(), slot), sample.InitialLevel)
		model.LearnIntegerDigits(img, ocr.FinalLevelRect(img.Bounds(), slot), sample.FinalLevel)
		model.LearnValueDigits(img, slot, 0, true, sample.Main.Init)
		model.LearnValueDigits(img, slot, 0, false, sample.Main.Final)
		for index, attr := range sample.Subs {
			row := index + 1
			rowRect := ocr.RowRect(img.Bounds(), slot, row)
			mask := ocr.LabelImageMask(ocr.LabelImage(img, rowRect))
			if mask.Count() > 0 {
				model.LabelTemplates = append(model.LabelTemplates, ocr.LabelTemplate{Name: attr.Name, Mask: mask})
				seenLabels[attr.Name] = true
			}
			model.LearnValueDigits(img, slot, row, true, attr.Init)
			if attr.Final != "" {
				model.LearnValueDigits(img, slot, row, false, attr.Final)
			}
		}
	}
	if err := ocr.ValidateLearnedModel(model, seenLabels); err != nil {
		return nil, err
	}
	return model, nil
}
