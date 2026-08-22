package excel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStyle(t *testing.T) {
	s := NewStyle()
	assert.NotNil(t, s)
	assert.NotNil(t, s.Font)
	assert.NotNil(t, s.Fill)
	assert.NotNil(t, s.Border)
	assert.NotNil(t, s.Alignment)
	assert.Equal(t, 11.0, s.Font.Size)
	assert.Equal(t, ColorBlack, s.Font.Color)
}

func TestStyleBuilder_Font(t *testing.T) {
	s := NewStyle().
		Bold(true).
		Italic(true).
		FontSize(14).
		FontColor(ColorRed).
		FontFamily("Arial")

	assert.True(t, s.Font.Bold)
	assert.True(t, s.Font.Italic)
	assert.Equal(t, 14.0, s.Font.Size)
	assert.Equal(t, ColorRed, s.Font.Color)
	assert.Equal(t, "Arial", s.Font.Family)
}

func TestStyleBuilder_Fill(t *testing.T) {
	s := NewStyle().BackgroundColor(ColorBlue)
	assert.Equal(t, []string{ColorBlue}, s.Fill.Color)
}

func TestStyleBuilder_Border(t *testing.T) {
	s := NewStyle().
		BorderAll(BorderThin).
		BorderColor(ColorBlack)

	assert.Equal(t, BorderThin, s.Border.Left)
	assert.Equal(t, BorderThin, s.Border.Right)
	assert.Equal(t, BorderThin, s.Border.Top)
	assert.Equal(t, BorderThin, s.Border.Bottom)
	assert.Equal(t, ColorBlack, s.Border.Color)
}

func TestStyleBuilder_Alignment(t *testing.T) {
	s := NewStyle().
		AlignHorizontal(AlignCenter).
		AlignVertical(AlignTop).
		WrapText(true)

	assert.Equal(t, AlignCenter, s.Alignment.Horizontal)
	assert.Equal(t, AlignTop, s.Alignment.Vertical)
	assert.True(t, s.Alignment.WrapText)
}

func TestStyleBuilder_Format(t *testing.T) {
	s := NewStyle().Format(FormatCurrency)
	assert.Equal(t, FormatCurrency, s.NumFmt)
}

func TestStyleBuilder_FillPattern(t *testing.T) {
	s := NewStyle().FillPattern(9)
	assert.Equal(t, 9, s.Fill.Pattern)

	es := s.ToExcelizeStyle()
	assert.Equal(t, 9, es.Fill.Pattern)
}

func TestStyle_ToExcelizeStyle(t *testing.T) {
	s := NewStyle().
		Bold(true).
		BackgroundColor(ColorGreen).
		BorderAll(BorderMedium).
		AlignHorizontal(AlignRight)

	es := s.ToExcelizeStyle()
	assert.NotNil(t, es)
	assert.True(t, es.Font.Bold)
	assert.Equal(t, []string{ColorGreen}, es.Fill.Color)
	assert.Equal(t, 2, es.Border[0].Style) // Medium = 2
	assert.Equal(t, AlignRight, es.Alignment.Horizontal)
}

func TestConvertBorderStyle(t *testing.T) {
	assert.Equal(t, 1, convertBorderStyle(BorderThin))
	assert.Equal(t, 2, convertBorderStyle(BorderMedium))
	assert.Equal(t, 3, convertBorderStyle(BorderDashed))
	assert.Equal(t, 4, convertBorderStyle(BorderDotted))
	assert.Equal(t, 5, convertBorderStyle(BorderThick))
	assert.Equal(t, 0, convertBorderStyle("unknown"))
}
