package main

import (
	"time"
	"unicode"
	"unicode/utf8"
)

// field is a single text input within a form: its current value and a filter
// predicate deciding whether a typed character is accepted. current is the
// field's existing value, letting a filter make context-dependent decisions
// (e.g. accepting a character only if it extends a valid prefix of "null").
type field struct {
	value  string
	filter func(char, current string) bool
}

// form is the shared behavior for multi-field text entry: a set of fields, the
// index of the focused one, and a validation error string. handleRune,
// backspace, and tab are the entire input-handling surface, exercised directly
// in form_test.go without constructing the surrounding appModel.
type form struct {
	fields []field
	focus  int
	err    string
}

// handleRune applies the focused field's filter to r and appends it on success.
// It reports whether the character was accepted.
func (f *form) handleRune(r rune) bool {
	if f.focus < 0 || f.focus >= len(f.fields) {
		return false
	}
	fld := &f.fields[f.focus]
	char := string(r)
	if fld.filter(char, fld.value) {
		fld.value += char
		return true
	}
	return false
}

// backspace removes the last rune from the focused field. It trims a whole
// rune rather than a single byte so deleting a multibyte character (the
// comment/title/gunits fields accept any printable Unicode) leaves valid UTF-8.
func (f *form) backspace() {
	if f.focus < 0 || f.focus >= len(f.fields) {
		return
	}
	fld := &f.fields[f.focus]
	if len(fld.value) > 0 {
		_, size := utf8.DecodeLastRuneInString(fld.value)
		fld.value = fld.value[:len(fld.value)-size]
	}
}

// val returns the value of field i, or "" if i is out of range. This keeps
// named accessors safe to call on a zero-value form (nil fields), which the
// view does when a modal is open but its form has not been initialized yet.
func (f *form) val(i int) string {
	if i < 0 || i >= len(f.fields) {
		return ""
	}
	return f.fields[i].value
}

// tab moves focus to the next field, or the previous one when reverse is true,
// wrapping around.
func (f *form) tab(reverse bool) {
	n := len(f.fields)
	if n == 0 {
		return
	}
	if reverse {
		f.focus = (f.focus + n - 1) % n
	} else {
		f.focus = (f.focus + 1) % n
	}
}

// Field filters. Each adapts an existing character predicate (defined in
// handlers.go) to the field.filter signature, so behavior is identical to the
// pre-extraction handlers.

func filterSlug(char, _ string) bool            { return isAlphanumericOrDash(char) }
func filterLetter(char, _ string) bool          { return isLetter(char) }
func filterIntOrNull(char, cur string) bool     { return isNumericOrNull(char, cur) }
func filterDecimalOrNull(char, cur string) bool { return isNumericWithDecimal(char, cur) }

// filterPrintable accepts any single printable Unicode character.
func filterPrintable(char, _ string) bool {
	runes := []rune(char)
	return len(runes) == 1 && unicode.IsPrint(runes[0])
}

// filterDate accepts digits and dashes (YYYY-MM-DD entry).
func filterDate(char, _ string) bool {
	return (char >= "0" && char <= "9") || char == "-"
}

// filterDecimal accepts digits, a decimal point, and a negative sign.
func filterDecimal(char, _ string) bool {
	return (char >= "0" && char <= "9") || char == "." || char == "-"
}

// datapointForm is the in-progress datapoint entry shown inside the goal detail
// modal: the date/value/comment fields plus whether a submission is in flight.
type datapointForm struct {
	form
	submitting bool
}

// Field indices for datapointForm.
const (
	dpDate = iota
	dpValue
	dpComment
)

// newDatapointForm builds a datapoint entry form with sensible defaults.
// defaultValue is the pre-filled value field (typically the goal's last
// datapoint value, or "1").
func newDatapointForm(defaultValue string) datapointForm {
	fields := make([]field, 3)
	fields[dpDate] = field{value: time.Now().Format("2006-01-02"), filter: filterDate}
	fields[dpValue] = field{value: defaultValue, filter: filterDecimal}
	fields[dpComment] = field{value: "Added via buzz", filter: filterPrintable}
	return datapointForm{form: form{fields: fields}}
}

func (d *datapointForm) date() string    { return d.val(dpDate) }
func (d *datapointForm) value() string   { return d.val(dpValue) }
func (d *datapointForm) comment() string { return d.val(dpComment) }

// validate reports a validation error message, or "" when the form is valid.
func (d *datapointForm) validate() string {
	return validateDatapointInput(d.date(), d.value())
}

// createGoalForm is the in-progress new-goal entry shown in the create modal.
type createGoalForm struct {
	form
	creating bool
}

// Field indices for createGoalForm.
const (
	cgSlug = iota
	cgTitle
	cgGoalType
	cgGunits
	cgGoaldate
	cgGoalval
	cgRate
)

// newCreateGoalForm builds a goal-creation form with the default goal type,
// units, value, and rate pre-filled.
func newCreateGoalForm() createGoalForm {
	fields := make([]field, 7)
	fields[cgSlug] = field{filter: filterSlug}
	fields[cgTitle] = field{filter: filterPrintable}
	fields[cgGoalType] = field{value: "hustler", filter: filterLetter}
	fields[cgGunits] = field{value: "units", filter: filterPrintable}
	fields[cgGoaldate] = field{filter: filterIntOrNull}
	fields[cgGoalval] = field{value: "0", filter: filterDecimalOrNull}
	fields[cgRate] = field{value: "1", filter: filterDecimalOrNull}
	return createGoalForm{form: form{fields: fields}}
}

func (c *createGoalForm) slug() string     { return c.val(cgSlug) }
func (c *createGoalForm) title() string    { return c.val(cgTitle) }
func (c *createGoalForm) goalType() string { return c.val(cgGoalType) }
func (c *createGoalForm) gunits() string   { return c.val(cgGunits) }
func (c *createGoalForm) goaldate() string { return c.val(cgGoaldate) }
func (c *createGoalForm) goalval() string  { return c.val(cgGoalval) }
func (c *createGoalForm) rate() string     { return c.val(cgRate) }

// validate reports a validation error message, or "" when the form is valid.
func (c *createGoalForm) validate() string {
	return validateCreateGoalInput(c.slug(), c.title(), c.goalType(), c.gunits(),
		c.goaldate(), c.goalval(), c.rate())
}
