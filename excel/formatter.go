package excel

// Number format IDs for Style.NumFmt, based on Excel's built-in format
// codes. See: https://github.com/xuri/excelize/blob/master/style.go#L29
const (
	// FormatGeneral applies Excel's default "General" format.
	FormatGeneral = 0
	// FormatNumber renders an integer with format code "0".
	FormatNumber = 1
	// FormatDecimal renders a number with two decimal places, format
	// code "0.00".
	FormatDecimal = 2
	// FormatCurrency renders US dollar currency with format code
	// "$#,##0.00". Unlike the other constants in this block, 164 is not
	// one of Excel's built-in IDs; it is the first ID excelize reserves
	// for custom formats, so this value only resolves to a currency
	// format within this package's own styling helpers.
	FormatCurrency = 164
	// FormatPercentage renders a percentage with two decimal places,
	// format code "0.00%".
	FormatPercentage = 10
	// FormatDate renders a date with format code "mm-dd-yy".
	FormatDate = 14
	// FormatTime renders a time with format code "h:mm".
	FormatTime = 20
	// FormatDateTime renders a date and time with format code
	// "m/d/yy h:mm".
	FormatDateTime = 22
	// FormatText renders the cell as plain text, format code "@".
	FormatText = 49
)

// Common custom format strings, for use where excelize accepts a format
// code string directly rather than a built-in format ID.
const (
	// FmtCurrencyUSD is a custom format string for US dollar currency:
	// "$#,##0.00".
	FmtCurrencyUSD = "$#,##0.00"
	// FmtDateISO8601 is a custom format string for ISO 8601 dates:
	// "yyyy-mm-dd".
	FmtDateISO8601 = "yyyy-mm-dd"
)
