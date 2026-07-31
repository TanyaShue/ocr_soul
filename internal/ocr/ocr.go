package ocrsoul

import (
	"bytes"
	"encoding/gob"
	"errors"
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
	Order      int        `json:"order"`
	Type       string     `json:"type"`
	Position   int        `json:"position"`
	Level      LevelRange `json:"level"`
	Attributes Attributes `json:"attributes"`
}

type AttributeValue struct {
	Name    string  `json:"name"`
	Initial *string `json:"initial"`
	Final   string  `json:"final"`
}

type LevelRange struct {
	Initial int `json:"initial"`
	Final   int `json:"final"`
}

type Attributes struct {
	Main AttributeValue   `json:"main"`
	Subs []AttributeValue `json:"subs"`
}

type ImageResult struct {
	Image string       `json:"image"`
	Souls []SoulResult `json:"souls"`
}

type RecognitionOutput struct {
	SchemaVersion int           `json:"schema_version"`
	Results       []ImageResult `json:"results"`
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

type PositionTemplate struct {
	Position int
	Mask     BinaryImage
}

type SoulTypeTemplate struct {
	Name   string
	Pixels []uint8
}

type TemplateModel struct {
	Version           int
	LabelTemplates    []LabelTemplate
	DigitTemplates    []DigitTemplate
	PositionTemplates []PositionTemplate
	SoulTypeTemplates []SoulTypeTemplate
}

type Recognizer struct {
	LabelTemplates    []LabelTemplate
	DigitTemplates    []DigitTemplate
	PositionTemplates []PositionTemplate
	SoulTypeTemplates []SoulTypeTemplate
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

func CollectInputs(input string) ([]string, error) {
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

func LoadPNG(path string) (image.Image, error) {
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

func SaveTemplateModel(path string, model *TemplateModel) error {
	if err := validateTemplateModel(model, nil); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gob.NewEncoder(f).Encode(model)
}

func LoadTemplateModel(path string) (*TemplateModel, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var model TemplateModel
	if err := gob.NewDecoder(f).Decode(&model); err != nil {
		return nil, err
	}
	if err := validateTemplateModel(&model, nil); err != nil {
		return nil, err
	}
	return &model, nil
}

func DecodeTemplateModel(data []byte) (*TemplateModel, error) {
	var model TemplateModel
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&model); err != nil {
		return nil, err
	}
	if err := validateTemplateModel(&model, nil); err != nil {
		return nil, err
	}
	return &model, nil
}

func NewRecognizer(model *TemplateModel) (*Recognizer, error) {
	if err := validateTemplateModel(model, nil); err != nil {
		return nil, err
	}
	return &Recognizer{
		LabelTemplates:    model.LabelTemplates,
		DigitTemplates:    model.DigitTemplates,
		PositionTemplates: model.PositionTemplates,
		SoulTypeTemplates: model.SoulTypeTemplates,
	}, nil
}

func ValidateLearnedModel(model *TemplateModel, seenLabels map[string]bool) error {
	return validateTemplateModel(model, seenLabels)
}

func ScaleSlot(bounds image.Rectangle, slotNum int) Point {
	return scaleSlot(bounds, slotNum)
}

func RowRect(bounds image.Rectangle, slot Point, row int) Rect {
	return rowRect(bounds, slot, row)
}

func LabelImage(img image.Image, row Rect) image.Image {
	return crop(img, labelRect(row))
}

func LabelImageMask(img image.Image) BinaryImage {
	return labelMask(img)
}

func PositionImageMask(img image.Image, slot Point) BinaryImage {
	return positionMask(img, slot)
}

func InitialLevelRect(bounds image.Rectangle, slot Point) Rect {
	return initialLevelRect(bounds, slot)
}

func FinalLevelRect(bounds image.Rectangle, slot Point) Rect {
	return finalLevelRect(bounds, slot)
}

func validateTemplateModel(model *TemplateModel, seenLabels map[string]bool) error {
	if model == nil {
		return errors.New("template model is nil")
	}
	if len(seenLabels) < 10 {
		if seenLabels != nil {
			return errors.New("not enough attribute label templates were learned")
		}
		unique := map[string]bool{}
		for _, tmpl := range model.LabelTemplates {
			unique[tmpl.Name] = true
		}
		if len(unique) < 10 {
			return errors.New("not enough attribute label templates in model")
		}
	}
	if len(model.PositionTemplates) < 6 {
		return errors.New("not enough position templates in model")
	}
	if len(model.SoulTypeTemplates) == 0 {
		return errors.New("no soul type templates in model")
	}
	for d := '0'; d <= '9'; d++ {
		found := false
		for _, tmpl := range model.DigitTemplates {
			if tmpl.Char == d {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("digit template %q is missing from model", string(d))
		}
	}
	return nil
}

func (m *TemplateModel) LearnIntegerDigits(img image.Image, rect Rect, value int) {
	text := fmt.Sprintf("%d", value)
	comps := integerComponents(crop(img, rect), "white")
	if len(comps) != len(text) {
		return
	}
	for i, ch := range text {
		m.DigitTemplates = append(m.DigitTemplates, DigitTemplate{Char: ch, Mask: normalizeMask(comps[i].Mask, 12, 18)})
	}
}

func (m *TemplateModel) LearnValueDigits(img image.Image, slot Point, row int, initial bool, text string) {
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
	beforeCount := strings.IndexRune(text, '.')
	if beforeCount < 0 {
		return
	}
	dot := findDot(comps)
	if dot < 0 {
		return
	}
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
		m.DigitTemplates = append(m.DigitTemplates, DigitTemplate{Char: ch, Mask: normalizeMask(selected[i].Mask, 12, 18)})
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
		position := r.recognizePosition(img, slot)
		initialLevel := r.recognizeInteger(crop(img, initialLevelRect(img.Bounds(), slot)), "white")
		finalLevel := r.recognizeInteger(crop(img, finalLevelRect(img.Bounds(), slot)), "white")
		mainName := r.recognizeLabel(crop(img, labelRect(rowRect(img.Bounds(), slot, 0))))
		mainInit := normalizeAttributeValue(mainName, r.recognizeValue(crop(img, initialValueRect(rowRect(img.Bounds(), slot, 0))), "white"))
		mainFinalCandidate := normalizeAttributeValue(mainName, r.recognizeValue(crop(img, finalValueRect(rowRect(img.Bounds(), slot, 0))), "gold"))
		if mainInit == "" && mainName != "" {
			mainInit = normalizeAttributeValue(mainName, "10.00")
		}
		mainFinal := mainFinalCandidate
		if mainFinal == "" {
			mainFinal = mainInit
		}
		main := AttributeValue{Name: mainName, Initial: stringPointer(mainInit), Final: mainFinal}
		subs := []AttributeValue{}
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
			isNew := hasNewMarker(img, rr)
			isUpgraded := isNew || (finalCandidate != "" && finalCandidate != initValue)
			if isUpgraded {
				if finalCandidate != "" {
					finalValue = finalCandidate
				}
			}
			var initial *string
			if !isNew {
				initial = stringPointer(initValue)
			}
			attr := AttributeValue{Name: name, Initial: initial, Final: finalValue}
			subs = append(subs, attr)
		}
		results = append(results, SoulResult{
			Order:    len(results) + 1,
			Type:     r.recognizeSoulType(img, slot),
			Position: position,
			Level:    LevelRange{Initial: initialLevel, Final: finalLevel},
			Attributes: Attributes{
				Main: main,
				Subs: subs,
			},
		})
	}
	return results, nil
}

const soulTemplateSize = 32

// SoulTypeTemplatesFromImage creates shifted variants so hand-cropped samples do
// not need to have exactly the same size or center.
func SoulTypeTemplatesFromImage(name string, img image.Image) []SoulTypeTemplate {
	variants := make([]SoulTypeTemplate, 0, 27)
	step := max(1, min(img.Bounds().Dx(), img.Bounds().Dy())/24)
	for _, scalePercent := range []int{90, 100, 110} {
		for _, dy := range []int{-step, 0, step} {
			for _, dx := range []int{-step, 0, step} {
				variants = append(variants, SoulTypeTemplate{
					Name:   name,
					Pixels: normalizedColorPixels(img, dx, dy, scalePercent),
				})
			}
		}
	}
	return variants
}

func (r *Recognizer) recognizeSoulType(img image.Image, slot Point) string {
	bestName := ""
	bestScore := math.MaxFloat64
	step := max(1, scale(3, img.Bounds().Dx(), 1280))
	icon := crop(img, soulIconRect(img.Bounds(), slot))
	for _, scalePercent := range []int{90, 100, 110} {
		for _, dy := range []int{-step, 0, step} {
			for _, dx := range []int{-step, 0, step} {
				pixels := normalizedColorPixels(icon, dx, dy, scalePercent)
				for _, tmpl := range r.SoulTypeTemplates {
					score := colorDistance(pixels, tmpl.Pixels)
					if score < bestScore {
						bestScore = score
						bestName = tmpl.Name
					}
				}
			}
		}
	}
	return bestName
}

func stringPointer(value string) *string {
	return &value
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

func initialLevelRect(bounds image.Rectangle, slot Point) Rect {
	return Rect{
		X0: slot.X + scale(122, bounds.Dx(), 1280),
		Y0: slot.Y + scale(35, bounds.Dy(), 720),
		X1: slot.X + scale(150, bounds.Dx(), 1280),
		Y1: slot.Y + scale(62, bounds.Dy(), 720),
	}
}

func finalLevelRect(bounds image.Rectangle, slot Point) Rect {
	return Rect{
		X0: slot.X + scale(208, bounds.Dx(), 1280),
		Y0: slot.Y + scale(35, bounds.Dy(), 720),
		X1: slot.X + scale(250, bounds.Dx(), 1280),
		Y1: slot.Y + scale(62, bounds.Dy(), 720),
	}
}

func positionRect(bounds image.Rectangle, slot Point) Rect {
	return Rect{
		X0: slot.X - scale(25, bounds.Dx(), 1280),
		Y0: slot.Y - scale(10, bounds.Dy(), 720),
		X1: slot.X + scale(95, bounds.Dx(), 1280),
		Y1: slot.Y + scale(100, bounds.Dy(), 720),
	}
}

func soulIconRect(bounds image.Rectangle, slot Point) Rect {
	return Rect{
		X0: slot.X + scale(8, bounds.Dx(), 1280),
		Y0: slot.Y - scale(1, bounds.Dy(), 720),
		X1: slot.X + scale(78, bounds.Dx(), 1280),
		Y1: slot.Y + scale(69, bounds.Dy(), 720),
	}
}

func normalizedColorPixels(img image.Image, offsetX, offsetY, scalePercent int) []uint8 {
	b := img.Bounds()
	side := min(b.Dx(), b.Dy())
	margin := side / 20
	side = (side - margin*2) * scalePercent / 100
	cx := b.Min.X + b.Dx()/2 + offsetX
	cy := b.Min.Y + b.Dy()/2 + offsetY
	x0 := cx - side/2
	y0 := cy - side/2
	pixels := make([]uint8, soulTemplateSize*soulTemplateSize*3)
	luminance := make([]float64, soulTemplateSize*soulTemplateSize)
	var sum, sumSquares float64
	for y := 0; y < soulTemplateSize; y++ {
		sy := y0 + y*side/soulTemplateSize
		sy = max(b.Min.Y, min(sy, b.Max.Y-1))
		for x := 0; x < soulTemplateSize; x++ {
			sx := x0 + x*side/soulTemplateSize
			sx = max(b.Min.X, min(sx, b.Max.X-1))
			r, g, blue := rgb(img.At(sx, sy))
			i := (y*soulTemplateSize + x) * 3
			lum := float64(r*3+g*6+blue) / 10
			luminance[y*soulTemplateSize+x] = lum
			sum += lum
			sumSquares += lum * lum
			pixels[i] = uint8(max(0, min(255, r-g+128)))
			pixels[i+1] = uint8(max(0, min(255, blue-g+128)))
		}
	}
	count := float64(len(luminance))
	mean := sum / count
	variance := max(1.0, sumSquares/count-mean*mean)
	stddev := math.Sqrt(variance)
	for i, lum := range luminance {
		pixels[i*3+2] = uint8(max(0, min(255, int(math.Round((lum-mean)*40/stddev+128)))))
	}
	return pixels
}

func colorDistance(a, b []uint8) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return math.MaxFloat64
	}
	var total, weightTotal float64
	for i := 0; i < len(a); i += 3 {
		pixel := i / 3
		x := pixel % soulTemplateSize
		y := pixel / soulTemplateSize
		dx := float64(x) - float64(soulTemplateSize-1)/2
		dy := float64(y) - float64(soulTemplateSize-1)/2
		radius := math.Sqrt(dx*dx + dy*dy)
		if radius > float64(soulTemplateSize)*0.38 {
			continue
		}
		weight := 1.0
		if radius > float64(soulTemplateSize)*0.31 {
			weight = 0.35
		}
		d0 := float64(int(a[i]) - int(b[i]))
		d1 := float64(int(a[i+1]) - int(b[i+1]))
		d2 := float64(int(a[i+2]) - int(b[i+2]))
		total += (d0*d0*0.7 + d1*d1*0.7 + d2*d2) * weight
		weightTotal += weight
	}
	return total / weightTotal
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

func hasNewMarker(img image.Image, row Rect) bool {
	r := Rect{X0: row.X0 - 28, Y0: row.Y0 - 2, X1: row.X0 + 10, Y1: row.Y1 + 2}
	return newMarkerMask(crop(img, r)).Count() > 20
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

func (r *Recognizer) recognizePosition(img image.Image, slot Point) int {
	mask := positionMask(img, slot)
	bestPosition := 0
	bestScore := math.MaxFloat64
	for _, tmpl := range r.PositionTemplates {
		score := maskDistance(mask, tmpl.Mask)
		if score < bestScore {
			bestScore = score
			bestPosition = tmpl.Position
		}
	}
	return bestPosition
}

func (r *Recognizer) recognizeInteger(img image.Image, maskKind string) int {
	comps := integerComponents(img, maskKind)
	if len(comps) == 0 {
		return 0
	}
	text := ""
	for _, c := range comps {
		text += string(r.recognizeDigit(c.Mask))
	}
	var value int
	for _, ch := range text {
		if ch < '0' || ch > '9' {
			return 0
		}
		value = value*10 + int(ch-'0')
	}
	return value
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
	comps := splitWideDigitComponents(connectedComponents(mask))
	filtered := comps[:0]
	for _, c := range comps {
		h := c.Y1 - c.Y0
		if c.Area < 3 || h < 2 {
			continue
		}
		if (c.X0 <= 1 || c.X1 >= mask.W-1) && c.Area < 12 {
			continue
		}
		filtered = append(filtered, c)
	}
	sortComponents(filtered)
	return filtered
}

func splitWideDigitComponents(comps []Component) []Component {
	out := make([]Component, 0, len(comps))
	for _, c := range comps {
		w := c.X1 - c.X0
		h := c.Y1 - c.Y0
		if w < 16 || h > 20 {
			out = append(out, c)
			continue
		}
		parts := int(math.Round(float64(w) / 9.0))
		if parts < 2 || parts > 3 {
			out = append(out, c)
			continue
		}
		for i := 0; i < parts; i++ {
			x0 := i * w / parts
			x1 := (i + 1) * w / parts
			part := subMaskComponent(c, x0, x1)
			if part.Area >= 3 {
				out = append(out, part)
			}
		}
	}
	return out
}

func subMaskComponent(c Component, localX0, localX1 int) Component {
	if localX0 < 0 {
		localX0 = 0
	}
	if localX1 > c.Mask.W {
		localX1 = c.Mask.W
	}
	points := [][2]int{}
	x0, y0, x1, y1 := localX1, c.Mask.H, localX0, 0
	for y := 0; y < c.Mask.H; y++ {
		for x := localX0; x < localX1; x++ {
			if !c.Mask.P[y*c.Mask.W+x] {
				continue
			}
			points = append(points, [2]int{x, y})
			if x < x0 {
				x0 = x
			}
			if y < y0 {
				y0 = y
			}
			if x+1 > x1 {
				x1 = x + 1
			}
			if y+1 > y1 {
				y1 = y + 1
			}
		}
	}
	if len(points) == 0 {
		return Component{}
	}
	mask := BinaryImage{W: x1 - x0, H: y1 - y0, P: make([]bool, (x1-x0)*(y1-y0))}
	for _, p := range points {
		mask.P[(p[1]-y0)*mask.W+(p[0]-x0)] = true
	}
	return Component{
		X0:   c.X0 + x0,
		Y0:   c.Y0 + y0,
		X1:   c.X0 + x1,
		Y1:   c.Y0 + y1,
		Area: len(points),
		Mask: mask,
	}
}

func integerComponents(img image.Image, maskKind string) []Component {
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
		if c.Area < 8 || h < 8 || w > 14 {
			continue
		}
		filtered = append(filtered, c)
	}
	sortComponents(filtered)
	return filtered
}

func positionMask(img image.Image, slot Point) BinaryImage {
	return makeMask(crop(img, positionRect(img.Bounds(), slot)), func(c color.Color) bool {
		r, g, b := rgb(c)
		return r > 165 && g > 110 && b < 105 && r > g && g > b+25
	})
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

func newMarkerMask(img image.Image) BinaryImage {
	return makeMask(img, func(c color.Color) bool {
		r, g, b := rgb(c)
		return g > 130 && r < 80 && b < 90 && g > r+80 && g > b+80
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
