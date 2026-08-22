package excel

import "github.com/xuri/excelize/v2"

// Common colors, as hex RGB strings accepted by Font.Color, Fill.Color,
// and Border.Color.
const (
	// ColorBlack is opaque black (#000000).
	ColorBlack = "#000000"
	// ColorWhite is opaque white (#FFFFFF).
	ColorWhite = "#FFFFFF"
	// ColorRed is pure red (#FF0000).
	ColorRed = "#FF0000"
	// ColorBlue is pure blue (#0000FF).
	ColorBlue = "#0000FF"
	// ColorGreen is pure green (#00FF00).
	ColorGreen = "#00FF00"
	// ColorGray is medium gray (#808080).
	ColorGray = "#808080"
)

// Border styles, accepted by Border.Left, Border.Right, Border.Top, and
// Border.Bottom.
const (
	// BorderNone renders no border line.
	BorderNone = "none"
	// BorderThin renders a single thin line.
	BorderThin = "thin"
	// BorderMedium renders a single medium-weight line.
	BorderMedium = "medium"
	// BorderThick renders a single thick line.
	BorderThick = "thick"
	// BorderDashed renders a dashed line.
	BorderDashed = "dashed"
	// BorderDotted renders a dotted line.
	BorderDotted = "dotted"
)

// Horizontal alignment values, accepted by Alignment.Horizontal.
const (
	// AlignLeft aligns content to the left edge of the cell.
	AlignLeft = "left"
	// AlignCenter horizontally centers content within the cell.
	AlignCenter = "center"
	// AlignRight aligns content to the right edge of the cell.
	AlignRight = "right"
)

// Vertical alignment values, accepted by Alignment.Vertical.
const (
	// AlignTop aligns content to the top edge of the cell.
	AlignTop = "top"
	// AlignMiddle vertically centers content within the cell. It shares
	// excelize's underlying "center" value with AlignCenter, but the two
	// are not interchangeable: AlignCenter is horizontal-only and
	// AlignMiddle is vertical-only.
	AlignMiddle = "center"
	// AlignBottom aligns content to the bottom edge of the cell.
	AlignBottom = "bottom"
)

// Style defines the visual appearance of a cell or range. Use NewStyle to
// construct one with sensible defaults, then apply it via
// ToExcelizeStyle.
type Style struct {
	// Font holds the text font properties. Never nil on a Style returned
	// by NewStyle.
	Font *Font
	// Fill holds the cell background fill properties. Never nil on a
	// Style returned by NewStyle.
	Fill *Fill
	// Border holds the per-side border properties. Never nil on a Style
	// returned by NewStyle.
	Border *Border
	// Alignment holds the text alignment properties. Never nil on a
	// Style returned by NewStyle.
	Alignment *Alignment
	// NumFmt is the excelize built-in number format ID applied to the
	// cell (see formatter.go's NumFmt* constants), or 0 for the default
	// "General" format.
	NumFmt int
}

// Font defines font properties applied to a cell's text.
type Font struct {
	// Bold renders the text in bold weight.
	Bold bool
	// Italic renders the text in italic style.
	Italic bool
	// Underline is the underline style: "single", "double", or "" for
	// none.
	Underline string
	// Size is the font size in points.
	Size float64
	// Color is the text color as a hex RGB string, e.g. ColorBlack.
	Color string
	// Family is the font family name, e.g. "Calibri" or "Arial".
	Family string
}

// Fill defines a cell's background fill properties.
type Fill struct {
	// Type is the excelize fill type. NewStyle sets this to "pattern",
	// the only type this package fills in; other excelize-supported
	// values (e.g. "gradient") are not populated by this package's
	// builder methods but are preserved if set directly.
	Type string
	// Color lists the fill colors as hex RGB strings. For a solid
	// pattern fill (the default from NewStyle), only the first entry is
	// used as the foreground color.
	Color []string
	// Pattern is the excelize pattern fill index. NewStyle sets this to
	// 1, which selects a solid fill.
	Pattern int
}

// Border defines a cell's per-side border properties. Each side accepts
// one of the Border* style constants.
type Border struct {
	// Left is the left border style.
	Left string
	// Right is the right border style.
	Right string
	// Top is the top border style.
	Top string
	// Bottom is the bottom border style.
	Bottom string
	// Color is the border color as a hex RGB string, shared by all four
	// sides.
	Color string
}

// Alignment defines a cell's text alignment properties.
type Alignment struct {
	// Horizontal is one of the Align* horizontal constants (AlignLeft,
	// AlignCenter, AlignRight), or "" for the default.
	Horizontal string
	// Vertical is one of the Align* vertical constants (AlignTop,
	// AlignMiddle, AlignBottom), or "" for the default.
	Vertical string
	// WrapText enables word wrapping within the cell.
	WrapText bool
}

// NewStyle creates a new empty Style builder.
func NewStyle() *Style {
	return &Style{
		Font:      &Font{Size: 11, Color: ColorBlack},
		Fill:      &Fill{Type: "pattern", Pattern: 1},
		Border:    &Border{},
		Alignment: &Alignment{Vertical: AlignMiddle},
	}
}

// Bold sets the font bold property.
func (s *Style) Bold(bold bool) *Style {
	s.Font.Bold = bold
	return s
}

// Italic sets the font italic property.
func (s *Style) Italic(italic bool) *Style {
	s.Font.Italic = italic
	return s
}

// FontSize sets the font size.
func (s *Style) FontSize(size float64) *Style {
	s.Font.Size = size
	return s
}

// FontColor sets the font color (hex code).
func (s *Style) FontColor(color string) *Style {
	s.Font.Color = color
	return s
}

// FontFamily sets the font family.
func (s *Style) FontFamily(family string) *Style {
	s.Font.Family = family
	return s
}

// BackgroundColor sets the background fill color.
func (s *Style) BackgroundColor(color string) *Style {
	s.Fill.Color = []string{color}
	return s
}

// BorderAll sets the border style for all sides.
func (s *Style) BorderAll(style string) *Style {
	s.Border.Left = style
	s.Border.Right = style
	s.Border.Top = style
	s.Border.Bottom = style
	return s
}

// BorderColor sets the border color.
func (s *Style) BorderColor(color string) *Style {
	s.Border.Color = color
	return s
}

// AlignHorizontal sets the horizontal alignment.
func (s *Style) AlignHorizontal(align string) *Style {
	s.Alignment.Horizontal = align
	return s
}

// AlignVertical sets the vertical alignment.
func (s *Style) AlignVertical(align string) *Style {
	s.Alignment.Vertical = align
	return s
}

// WrapText enables or disables text wrapping.
func (s *Style) WrapText(wrap bool) *Style {
	s.Alignment.WrapText = wrap
	return s
}

// Format sets the number format code (see formatter.go or excelize docs).
func (s *Style) Format(fmtID int) *Style {
	s.NumFmt = fmtID
	return s
}

// FillPattern sets the fill pattern index (excelize accepts 0-18; 1 is
// a solid fill, the default from NewStyle). Has no effect unless Fill
// is non-nil, which NewStyle guarantees.
func (s *Style) FillPattern(n int) *Style {
	s.Fill.Pattern = n
	return s
}

// ToExcelizeStyle converts the Style struct to excelize.Style format.
func (s *Style) ToExcelizeStyle() *excelize.Style {
	style := &excelize.Style{
		Font: &excelize.Font{
			Bold:   s.Font.Bold,
			Italic: s.Font.Italic,
			Size:   s.Font.Size,
			Color:  s.Font.Color,
			Family: s.Font.Family,
		},
		Fill: excelize.Fill{
			Type:    s.Fill.Type,
			Pattern: s.Fill.Pattern,
			Color:   s.Fill.Color,
		},
		Border: []excelize.Border{
			{Type: "left", Color: s.Border.Color, Style: convertBorderStyle(s.Border.Left)},
			{Type: "right", Color: s.Border.Color, Style: convertBorderStyle(s.Border.Right)},
			{Type: "top", Color: s.Border.Color, Style: convertBorderStyle(s.Border.Top)},
			{Type: "bottom", Color: s.Border.Color, Style: convertBorderStyle(s.Border.Bottom)},
		},
		Alignment: &excelize.Alignment{
			Horizontal: s.Alignment.Horizontal,
			Vertical:   s.Alignment.Vertical,
			WrapText:   s.Alignment.WrapText,
		},
		NumFmt: s.NumFmt,
	}
	return style
}

// convertBorderStyle converts string border style to excelize int style.
// This is a simplified mapping. Excelize uses int 0-13.
func convertBorderStyle(style string) int {
	switch style {
	case BorderThin:
		return 1
	case BorderMedium:
		return 2
	case BorderDashed:
		return 3
	case BorderDotted:
		return 4
	case BorderThick:
		return 5
	default:
		return 0
	}
}
