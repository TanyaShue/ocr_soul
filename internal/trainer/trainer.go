package trainer

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

type SelectedSoul struct {
	File     string
	Type     string
	Position int
	Level    int
	Main     SelectedAttribute
	Subs     []SelectedAttribute
}

type SelectedAttribute struct {
	Name  string
	Value string
}

func TrainSelected(assetsDir string, samples []SelectedSoul, base *ocr.TemplateModel) (*ocr.TemplateModel, error) {
	model := &ocr.TemplateModel{Version: 3, Kind: "selected"}
	// Reuse the complete soul catalogue; selected screenshots only refine their own layout.
	model.SoulTypeTemplates = append(model.SoulTypeTemplates, base.SoulTypeTemplates...)
	seenLabels := map[string]bool{}
	for _, sample := range samples {
		imagePath := filepath.Join(assetsDir, sample.File)
		if _, statErr := os.Stat(imagePath); statErr != nil {
			imagePath = filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(assetsDir))), "test", sample.File)
		}
		img, err := ocr.LoadPNG(imagePath)
		if err != nil {
			return nil, fmt.Errorf("load selected training image %s: %w", sample.File, err)
		}
		if sample.Position > 0 {
			model.PositionTemplates = append(model.PositionTemplates, ocr.PositionTemplate{Position: sample.Position, Mask: ocr.SelectedPositionImageMask(img)})
		}
		if sample.Level >= 0 {
			model.LearnSelectedIntegerDigits(img, ocr.SelectedHeaderRect(img.Bounds()), sample.Level)
		}
		attrs := []SelectedAttribute{}
		if sample.Main.Name != "" {
			attrs = append(attrs, sample.Main)
		}
		attrs = append(attrs, sample.Subs...)
		for row, attr := range attrs {
			rr := ocr.SelectedRowRect(img.Bounds(), row)
			mask := ocr.SelectedLabelImageMask(ocr.SelectedLabelImage(img, rr))
			if mask.Count() > 0 {
				model.LabelTemplates = append(model.LabelTemplates, ocr.LabelTemplate{Name: attr.Name, Mask: mask})
				seenLabels[attr.Name] = true
			}
			model.LearnSelectedValueDigits(img, ocr.SelectedValueRect(rr), attr.Value)
		}
		if sample.Type != "" {
			model.SoulTypeTemplates = append(model.SoulTypeTemplates, ocr.SoulTypeTemplatesFromImage(sample.Type, cropSelectedIcon(img))...)
		}
	}
	if err := ocr.ValidateLearnedModel(model, seenLabels); err != nil {
		return nil, err
	}
	return model, nil
}

func cropSelectedIcon(img image.Image) image.Image {
	r := ocr.SelectedIconRect(img.Bounds())
	return img.(interface {
		SubImage(image.Rectangle) image.Image
	}).SubImage(image.Rect(r.X0, r.Y0, r.X1, r.Y1))
}

func Train(assetsDir string, samples []Soul) (*ocr.TemplateModel, error) {
	model := &ocr.TemplateModel{Version: 2}
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
	if err := learnSoulTypes(model, filepath.Join(assetsDir, "soul")); err != nil {
		return nil, err
	}
	if err := ocr.ValidateLearnedModel(model, seenLabels); err != nil {
		return nil, err
	}
	return model, nil
}

func learnSoulTypes(model *ocr.TemplateModel, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read soul type samples: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".png") {
			continue
		}
		name, err := soulTypeFromFilename(entry.Name())
		if err != nil {
			return err
		}
		img, err := ocr.LoadPNG(filepath.Join(dir, entry.Name()))
		if err != nil {
			return fmt.Errorf("load soul type image %s: %w", entry.Name(), err)
		}
		model.SoulTypeTemplates = append(model.SoulTypeTemplates, ocr.SoulTypeTemplatesFromImage(name, img)...)
	}
	if len(model.SoulTypeTemplates) == 0 {
		return fmt.Errorf("no PNG soul type samples found in %s", dir)
	}
	return nil
}

func soulTypeFromFilename(filename string) (string, error) {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	index := strings.LastIndex(base, "_")
	if index < 0 || index == len(base)-1 {
		return "", fmt.Errorf("soul type filename must end in _<name>.png: %s", filename)
	}
	return base[index+1:], nil
}
