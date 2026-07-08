package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type SoulResult struct {
	Index             int              `json:"index"`
	MainAttribute     AttributeValue   `json:"main_attribute"`
	SubAttributeCount int              `json:"sub_attribute_count"`
	SubAttributes     []AttributeValue `json:"sub_attributes"`
	UpgradedAttribute *AttributeValue  `json:"upgraded_attribute,omitempty"`
	FinalAttributes   FinalAttributes  `json:"final_attributes"`
}

type AttributeValue struct {
	Name         string `json:"name"`
	InitialValue string `json:"initial_value,omitempty"`
	FinalValue   string `json:"final_value"`
	Upgraded     bool   `json:"upgraded,omitempty"`
}

type FinalAttributes struct {
	Main AttributeValue   `json:"main"`
	Subs []AttributeValue `json:"subs"`
}

type ImageResult struct {
	Image string       `json:"image"`
	Souls []SoulResult `json:"souls"`
}

type Point struct {
	X int
	Y int
}

type Rect struct {
	X0 int
	Y0 int
	X1 int
	Y1 int
}

type TrainingSoul struct {
	File string
	Slot int
	Main TrainingAttr
	Subs []TrainingAttr
}

type TrainingAttr struct {
	Name  string
	Init  string
	Final string
}

type BinaryImage struct {
	W int
	H int
	P []bool
}

type Component struct {
	X0   int
	Y0   int
	X1   int
	Y1   int
	Area int
	Mask BinaryImage
}

type DigitTemplate struct {
	Char rune
	Mask BinaryImage
}

type LabelTemplate struct {
	Name string
	Mask BinaryImage
}

type Recognizer struct {
	BaseDir        string
	LabelTemplates []LabelTemplate
	DigitTemplates []DigitTemplate
}

var slots = []Point{
	{204, 79}, {503, 79}, {802, 79},
	{204, 373}, {503, 373}, {802, 373},
}

var rowRects = []Rect{
	{10, 70, 260, 103},
	{10, 103, 260, 136},
	{10, 136, 260, 169},
	{10, 169, 260, 202},
	{10, 202, 260, 235},
}

var trainingSet = []TrainingSoul{
	{File: "MuMu-20260708-142511-164.png", Slot: 1, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "4.54"}, {Name: "防御加成", Init: "2.46%"}, {Name: "速度", Init: "2.99"}, {Name: "效果命中", Init: "3.46%", Final: "7.35%"},
	}},
	{File: "MuMu-20260708-142511-164.png", Slot: 2, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "4.99"}, {Name: "防御加成", Init: "2.77%"}, {Name: "效果命中", Init: "3.42%", Final: "7.01%"}, {Name: "效果抵抗", Init: "3.41%"},
	}},
	{File: "MuMu-20260708-142511-164.png", Slot: 3, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "生命", Init: "106.70"}, {Name: "防御加成", Init: "2.49%"}, {Name: "攻击加成", Init: "2.99%"}, {Name: "效果命中", Init: "3.22%", Final: "6.84%"},
	}},
	{File: "MuMu-20260708-142511-164.png", Slot: 4, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "5.00"}, {Name: "防御加成", Init: "2.80%", Final: "5.60%"}, {Name: "攻击加成", Init: "2.99%"}, {Name: "暴击", Init: "2.41%"},
	}},
	{File: "MuMu-20260708-142511-164.png", Slot: 5, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "生命加成", Init: "2.53%"}, {Name: "防御加成", Init: "2.44%"}, {Name: "速度", Init: "2.54", Final: "5.44"}, {Name: "效果命中", Init: "3.82%"},
	}},
	{File: "MuMu-20260708-142511-164.png", Slot: 6, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "4.96"}, {Name: "攻击", Init: "22.08"}, {Name: "防御加成", Init: "2.71%", Final: "5.48%"}, {Name: "速度", Init: "2.88"},
	}},
	{File: "MuMu-20260708-142607-072.png", Slot: 1, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "4.55"}, {Name: "防御加成", Init: "2.56%", Final: "5.26%"}, {Name: "速度", Init: "2.83"}, {Name: "暴击伤害", Init: "3.88%"},
	}},
	{File: "MuMu-20260708-142618-504.png", Slot: 1, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "4.28", Final: "9.15"}, {Name: "防御加成", Init: "2.47%"}, {Name: "暴击伤害", Init: "3.96%"}, {Name: "效果抵抗", Init: "3.38%"},
	}},
	{File: "MuMu-20260708-142618-504.png", Slot: 2, Main: TrainingAttr{Name: "防御加成", Init: "10.00%", Final: "19.00%"}, Subs: []TrainingAttr{
		{Name: "防御", Init: "4.21"}, {Name: "防御加成", Init: "2.81%"}, {Name: "暴击", Init: "2.50%", Final: "5.16%"}, {Name: "效果命中", Init: "3.74%"},
	}},
}

func main() {
	assetsDir := flag.String("assets", "assets", "directory containing the three training screenshots")
	input := flag.String("input", "assets", "image file or directory to recognize")
	flag.Parse()

	recognizer, err := NewRecognizer(*assetsDir)
	if err != nil {
		fatal(err)
	}
	paths, err := collectInputs(*input)
	if err != nil {
		fatal(err)
	}
	results := make([]ImageResult, 0, len(paths))
	for _, p := range paths {
		img, err := loadPNG(p)
		if err != nil {
			fatal(fmt.Errorf("%s: %w", p, err))
		}
		souls, err := recognizer.Recognize(img)
		if err != nil {
			fatal(fmt.Errorf("%s: %w", p, err))
		}
		results = append(results, ImageResult{Image: filepath.ToSlash(p), Souls: souls})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(results); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func collectInputs(input string) ([]string, error) {
	info, err := os.Stat(input)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{input}, nil
	}
	var paths []string
	err = filepath.WalkDir(input, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".png") {
			paths = append(paths, path)
		}
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

func loadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func NewRecognizer(assetsDir string) (*Recognizer, error) {
	r := &Recognizer{BaseDir: assetsDir}
	seenLabels := map[string]bool{}
	for _, sample := range trainingSet {
		img, err := loadPNG(filepath.Join(assetsDir, sample.File))
		if err != nil {
			return nil, fmt.Errorf("load training image %s: %w", sample.File, err)
		}
		slot := scaleSlot(img.Bounds(), sample.Slot)
		mainRow := rowRect(img.Bounds(), slot, 0)
		mainLabelMask := labelMask(crop(img, labelRect(mainRow)))
		if mainLabelMask.Count() > 0 {
			r.LabelTemplates = append(r.LabelTemplates, LabelTemplate{Name: sample.Main.Name, Mask: mainLabelMask})
			seenLabels[sample.Main.Name] = true
		}
		r.addDigitsFromValue(img, slot, 0, true, sample.Main.Init)
		r.addDigitsFromValue(img, slot, 0, false, sample.Main.Final)
		for i, attr := range sample.Subs {
			rowIndex := i + 1
			rr := rowRect(img.Bounds(), slot, rowIndex)
			mask := labelMask(crop(img, labelRect(rr)))
			if mask.Count() > 0 {
				r.LabelTemplates = append(r.LabelTemplates, LabelTemplate{Name: attr.Name, Mask: mask})
				seenLabels[attr.Name] = true
			}
			r.addDigitsFromValue(img, slot, rowIndex, true, attr.Init)
			if attr.Final != "" {
				r.addDigitsFromValue(img, slot, rowIndex, false, attr.Final)
			}
		}
	}
	if len(seenLabels) < 10 {
		return nil, errors.New("not enough attribute label templates were learned")
	}
	for d := '0'; d <= '9'; d++ {
		found := false
		for _, tmpl := range r.DigitTemplates {
			if tmpl.Char == d {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("digit template %q was not learned", string(d))
		}
	}
	return r, nil
}

func (r *Recognizer) addDigitsFromValue(img image.Image, slot Point, row int, initial bool, text string) {
	maskKind := "white"
	rect := initialValueRect(rowRect(img.Bounds(), slot, row))
	if !initial {
		maskKind = "gold"
		rect = finalValueRect(rowRect(img.Bounds(), slot, row))
	}
	comps := valueComponents(crop(img, rect), maskKind)
	digits := digitRunes(text)
	if len(digits) == 0 {
		return
	}
	dot := findDot(comps)
	if dot < 0 {
		return
	}
	beforeCount := strings.IndexRune(text, '.')
	afterCount := 2
	before := componentsBefore(comps, comps[dot])
	after := componentsAfter(comps, comps[dot])
	if len(before) < beforeCount || len(after) < afterCount {
		return
	}
	selected := append([]Component{}, before[len(before)-beforeCount:]...)
	selected = append(selected, after[:afterCount]...)
	if len(selected) != len(digits) {
		return
	}
	for i, ch := range digits {
		r.DigitTemplates = append(r.DigitTemplates, DigitTemplate{Char: ch, Mask: normalizeMask(selected[i].Mask, 12, 18)})
	}
}

func digitRunes(s string) []rune {
	out := []rune{}
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			out = append(out, ch)
		}
	}
	return out
}

func (r *Recognizer) Recognize(img image.Image) ([]SoulResult, error) {
	var results []SoulResult
	for i := range slots {
		slot := scaleSlot(img.Bounds(), i+1)
		if !hasSoul(img, slot) {
			continue
		}
		mainName := r.recognizeLabel(crop(img, labelRect(rowRect(img.Bounds(), slot, 0))))
		mainInit := normalizeAttributeValue(mainName, r.recognizeValue(crop(img, initialValueRect(rowRect(img.Bounds(), slot, 0))), "white"))
		mainFinal := normalizeAttributeValue(mainName, r.recognizeValue(crop(img, finalValueRect(rowRect(img.Bounds(), slot, 0))), "gold"))
		if mainInit == "" && mainName != "" {
			mainInit = normalizeAttributeValue(mainName, "10.00")
		}
		main := AttributeValue{Name: mainName, InitialValue: mainInit, FinalValue: mainFinal, Upgraded: mainInit != mainFinal}
		subs := []AttributeValue{}
		var upgraded *AttributeValue
		for row := 1; row <= 4; row++ {
			rr := rowRect(img.Bounds(), slot, row)
			labelCrop := crop(img, labelRect(rr))
			if labelMask(labelCrop).Count() < 16 {
				continue
			}
			name := r.recognizeLabel(labelCrop)
			initValue := normalizeAttributeValue(name, r.recognizeValue(crop(img, initialValueRect(rr)), "white"))
			finalValue := initValue
			finalCandidate := normalizeAttributeValue(name, r.recognizeValue(crop(img, finalValueRect(rr)), "gold"))
			isUpgraded := finalCandidate != ""
			if isUpgraded {
				finalValue = finalCandidate
			}
			attr := AttributeValue{Name: name, InitialValue: initValue, FinalValue: finalValue, Upgraded: isUpgraded}
			subs = append(subs, attr)
			if isUpgraded {
				copyAttr := attr
				upgraded = &copyAttr
			}
		}
		finalSubs := make([]AttributeValue, len(subs))
		for j, sub := range subs {
			finalSubs[j] = AttributeValue{Name: sub.Name, FinalValue: sub.FinalValue, Upgraded: sub.Upgraded}
		}
		results = append(results, SoulResult{
			Index:             len(results) + 1,
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
	return results, nil
}

func scaleSlot(bounds image.Rectangle, slotNum int) Point {
	base := slots[slotNum-1]
	w := bounds.Dx()
	h := bounds.Dy()
	return Point{X: scale(base.X, w, 1280), Y: scale(base.Y, h, 720)}
}

func rowRect(bounds image.Rectangle, slot Point, row int) Rect {
	base := rowRects[row]
	return Rect{
		X0: slot.X + scale(base.X0, bounds.Dx(), 1280),
		Y0: slot.Y + scale(base.Y0, bounds.Dy(), 720),
		X1: slot.X + scale(base.X1, bounds.Dx(), 1280),
		Y1: slot.Y + scale(base.Y1, bounds.Dy(), 720),
	}
}

func scale(v, actual, base int) int {
	return int(math.Round(float64(v) * float64(actual) / float64(base)))
}

func labelRect(row Rect) Rect {
	return Rect{X0: row.X0, Y0: row.Y0, X1: row.X0 + (row.X1-row.X0)*36/100, Y1: row.Y1}
}

func initialValueRect(row Rect) Rect {
	w := row.X1 - row.X0
	return Rect{X0: row.X0 + w*38/100, Y0: row.Y0, X1: row.X0 + w*68/100, Y1: row.Y1}
}

func finalValueRect(row Rect) Rect {
	w := row.X1 - row.X0
	return Rect{X0: row.X0 + w*72/100, Y0: row.Y0, X1: row.X0 + w*99/100, Y1: row.Y1}
}

func crop(img image.Image, r Rect) image.Image {
	b := img.Bounds()
	rr := image.Rect(r.X0, r.Y0, r.X1, r.Y1).Intersect(b)
	return img.(interface {
		SubImage(image.Rectangle) image.Image
	}).SubImage(rr)
}

func hasSoul(img image.Image, slot Point) bool {
	rect := Rect{X0: slot.X + 10, Y0: slot.Y + 70, X1: slot.X + 260, Y1: slot.Y + 103}
	mask := anyTextMask(crop(img, rect))
	return mask.Count() > 80
}

func hasGoldValue(img image.Image) bool {
	return goldMask(img).Count() > 20
}

func normalizeAttributeValue(name, value string) string {
	value = strings.TrimSuffix(value, "%")
	if value == "" {
		return ""
	}
	if attributeUsesPercent(name) {
		return value + "%"
	}
	return value
}

func attributeUsesPercent(name string) bool {
	switch name {
	case "攻击加成", "生命加成", "防御加成", "暴击", "暴击伤害", "效果命中", "效果抵抗":
		return true
	default:
		return false
	}
}

func (r *Recognizer) recognizeLabel(img image.Image) string {
	mask := labelMask(img)
	bestName := ""
	bestScore := math.MaxFloat64
	for _, tmpl := range r.LabelTemplates {
		score := maskDistance(mask, tmpl.Mask)
		if score < bestScore {
			bestScore = score
			bestName = tmpl.Name
		}
	}
	return bestName
}

func (r *Recognizer) recognizeValue(img image.Image, maskKind string) string {
	comps := valueComponents(img, maskKind)
	dot := findDot(comps)
	if dot < 0 {
		return ""
	}
	before := componentsBefore(comps, comps[dot])
	after := componentsAfter(comps, comps[dot])
	if len(before) == 0 || len(after) < 2 {
		return ""
	}
	intPart := ""
	for _, c := range before {
		intPart += string(r.recognizeDigit(c.Mask))
	}
	dec0 := r.recognizeDigit(after[0].Mask)
	dec1 := r.recognizeDigit(after[1].Mask)
	out := fmt.Sprintf("%s.%c%c", intPart, dec0, dec1)
	if hasTrailingPercent(after[1], after[2:]) {
		out += "%"
	}
	return out
}

func (r *Recognizer) recognizeDigit(mask BinaryImage) rune {
	norm := normalizeMask(mask, 12, 18)
	best := rune('0')
	bestScore := math.MaxFloat64
	for _, tmpl := range r.DigitTemplates {
		score := maskDistance(norm, tmpl.Mask)
		if score < bestScore {
			bestScore = score
			best = tmpl.Char
		}
	}
	return best
}

func componentsBefore(comps []Component, dot Component) []Component {
	var out []Component
	for _, c := range comps {
		if c.X1 <= dot.X0 && c.Area >= 8 {
			out = append(out, c)
		}
	}
	sortComponents(out)
	return out
}

func componentsAfter(comps []Component, dot Component) []Component {
	var out []Component
	for _, c := range comps {
		if c.X0 >= dot.X1 && c.Area >= 8 {
			out = append(out, c)
		}
	}
	sortComponents(out)
	return out
}

func hasTrailingPercent(secondDecimal Component, rest []Component) bool {
	area := 0
	for _, c := range rest {
		if c.X0 >= secondDecimal.X1-1 {
			area += c.Area
		}
	}
	return area >= 16
}

func findDot(comps []Component) int {
	best := -1
	for i, c := range comps {
		w := c.X1 - c.X0
		h := c.Y1 - c.Y0
		if w <= 4 && h <= 5 && c.Area <= 10 && c.Y0 > 14 {
			if best < 0 || c.X0 < comps[best].X0 {
				best = i
			}
		}
	}
	return best
}

func sortComponents(comps []Component) {
	sort.Slice(comps, func(i, j int) bool {
		if comps[i].X0 == comps[j].X0 {
			return comps[i].Y0 < comps[j].Y0
		}
		return comps[i].X0 < comps[j].X0
	})
}

func valueComponents(img image.Image, maskKind string) []Component {
	var mask BinaryImage
	if maskKind == "gold" {
		mask = goldMask(img)
	} else {
		mask = whiteMask(img)
	}
	comps := connectedComponents(mask)
	filtered := comps[:0]
	for _, c := range comps {
		w := c.X1 - c.X0
		h := c.Y1 - c.Y0
		if c.Area < 3 || h < 2 {
			continue
		}
		if (c.X0 <= 1 || c.X1 >= mask.W-1) && c.Area < 12 {
			continue
		}
		if w > 14 && h > 14 && c.Area > 90 {
			continue
		}
		filtered = append(filtered, c)
	}
	sortComponents(filtered)
	return filtered
}

func whiteMask(img image.Image) BinaryImage {
	return makeMask(img, func(c color.Color) bool {
		r, g, b := rgb(c)
		return r > 120 && g > 105 && b > 85 && max3(r, g, b)-min3(r, g, b) < 110 && !isGoldRGB(r, g, b)
	})
}

func goldMask(img image.Image) BinaryImage {
	return makeMask(img, func(c color.Color) bool {
		r, g, b := rgb(c)
		return isGoldRGB(r, g, b)
	})
}

func labelMask(img image.Image) BinaryImage {
	return makeMask(img, func(c color.Color) bool {
		r, g, b := rgb(c)
		return r > 110 && g > 100 && b > 80 && max3(r, g, b)-min3(r, g, b) < 110 && !isGoldRGB(r, g, b)
	})
}

func anyTextMask(img image.Image) BinaryImage {
	return makeMask(img, func(c color.Color) bool {
		r, g, b := rgb(c)
		return isGoldRGB(r, g, b) || (r > 110 && g > 100 && b > 80 && max3(r, g, b)-min3(r, g, b) < 120)
	})
}

func makeMask(img image.Image, keep func(color.Color) bool) BinaryImage {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	out := BinaryImage{W: w, H: h, P: make([]bool, w*h)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			out.P[y*w+x] = keep(img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return out
}

func connectedComponents(mask BinaryImage) []Component {
	seen := make([]bool, len(mask.P))
	var out []Component
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, -1}, {1, -1}, {-1, 1}}
	for y := 0; y < mask.H; y++ {
		for x := 0; x < mask.W; x++ {
			idx := y*mask.W + x
			if !mask.P[idx] || seen[idx] {
				continue
			}
			stack := [][2]int{{x, y}}
			seen[idx] = true
			points := [][2]int{}
			x0, y0, x1, y1 := x, y, x+1, y+1
			for len(stack) > 0 {
				p := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				px, py := p[0], p[1]
				points = append(points, p)
				if px < x0 {
					x0 = px
				}
				if py < y0 {
					y0 = py
				}
				if px+1 > x1 {
					x1 = px + 1
				}
				if py+1 > y1 {
					y1 = py + 1
				}
				for _, d := range dirs {
					nx, ny := px+d[0], py+d[1]
					if nx < 0 || ny < 0 || nx >= mask.W || ny >= mask.H {
						continue
					}
					ni := ny*mask.W + nx
					if mask.P[ni] && !seen[ni] {
						seen[ni] = true
						stack = append(stack, [2]int{nx, ny})
					}
				}
			}
			compMask := BinaryImage{W: x1 - x0, H: y1 - y0, P: make([]bool, (x1-x0)*(y1-y0))}
			for _, p := range points {
				compMask.P[(p[1]-y0)*compMask.W+(p[0]-x0)] = true
			}
			out = append(out, Component{X0: x0, Y0: y0, X1: x1, Y1: y1, Area: len(points), Mask: compMask})
		}
	}
	return out
}

func normalizeMask(src BinaryImage, w, h int) BinaryImage {
	dst := BinaryImage{W: w, H: h, P: make([]bool, w*h)}
	if src.W == 0 || src.H == 0 {
		return dst
	}
	for y := 0; y < h; y++ {
		sy := int(float64(y) * float64(src.H) / float64(h))
		if sy >= src.H {
			sy = src.H - 1
		}
		for x := 0; x < w; x++ {
			sx := int(float64(x) * float64(src.W) / float64(w))
			if sx >= src.W {
				sx = src.W - 1
			}
			dst.P[y*w+x] = src.P[sy*src.W+sx]
		}
	}
	return dst
}

func maskDistance(a, b BinaryImage) float64 {
	w := max(a.W, b.W)
	h := max(a.H, b.H)
	if w == 0 || h == 0 {
		return math.MaxFloat64
	}
	diff := 0
	total := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			av := maskAt(a, x, y, w, h)
			bv := maskAt(b, x, y, w, h)
			if av || bv {
				total++
				if av != bv {
					diff++
				}
			}
		}
	}
	if total == 0 {
		return math.MaxFloat64
	}
	sizePenalty := math.Abs(float64(a.W-b.W))*0.02 + math.Abs(float64(a.H-b.H))*0.02
	return float64(diff)/float64(total) + sizePenalty
}

func maskAt(m BinaryImage, x, y, w, h int) bool {
	if m.W == 0 || m.H == 0 {
		return false
	}
	mx := int(float64(x) * float64(m.W) / float64(w))
	my := int(float64(y) * float64(m.H) / float64(h))
	if mx >= m.W {
		mx = m.W - 1
	}
	if my >= m.H {
		my = m.H - 1
	}
	return m.P[my*m.W+mx]
}

func (m BinaryImage) Count() int {
	n := 0
	for _, v := range m.P {
		if v {
			n++
		}
	}
	return n
}

func rgb(c color.Color) (int, int, int) {
	r, g, b, _ := c.RGBA()
	return int(r >> 8), int(g >> 8), int(b >> 8)
}

func isGoldRGB(r, g, b int) bool {
	return r > 135 && g > 75 && b < 110 && r > g+15
}

func max3(a, b, c int) int {
	return max(max(a, b), c)
}

func min3(a, b, c int) int {
	return min(min(a, b), c)
}
