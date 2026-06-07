package main

import (
	"testing"
	"unicode/utf8"
)

// typeInto feeds each rune of s into the form one at a time, returning the slice
// of per-rune handled results.
func typeInto(f *form, s string) []bool {
	results := make([]bool, 0, len(s))
	for _, r := range s {
		results = append(results, f.handleRune(r))
	}
	return results
}

// TestFormHandleRuneAppendsAcceptedRunes verifies that handleRune appends only
// characters the focused field's filter accepts, and reports acceptance.
func TestFormHandleRuneAppendsAcceptedRunes(t *testing.T) {
	f := &form{fields: []field{{filter: filterDecimal}}}

	if !f.handleRune('1') {
		t.Fatal("expected '1' to be accepted by filterDecimal")
	}
	if f.handleRune('a') {
		t.Error("expected 'a' to be rejected by filterDecimal")
	}
	if !f.handleRune('.') {
		t.Error("expected '.' to be accepted by filterDecimal")
	}
	if got := f.fields[0].value; got != "1." {
		t.Errorf("field value = %q, want %q", got, "1.")
	}
}

// TestFormHandleRuneRoutesToFocusedField verifies that input lands in the
// currently focused field, not its neighbours.
func TestFormHandleRuneRoutesToFocusedField(t *testing.T) {
	f := &form{fields: []field{
		{filter: filterPrintable},
		{filter: filterPrintable},
	}}

	f.handleRune('a') // focus 0
	f.tab(false)      // focus 1
	f.handleRune('b')

	if f.fields[0].value != "a" {
		t.Errorf("field[0] = %q, want %q", f.fields[0].value, "a")
	}
	if f.fields[1].value != "b" {
		t.Errorf("field[1] = %q, want %q", f.fields[1].value, "b")
	}
}

// TestFormHandleRuneOutOfRangeFocus verifies handleRune is safe when focus is
// out of range (e.g. a malformed form) and reports the rune unhandled.
func TestFormHandleRuneOutOfRangeFocus(t *testing.T) {
	f := &form{fields: []field{{filter: filterPrintable}}, focus: 5}
	if f.handleRune('x') {
		t.Error("expected out-of-range focus to reject input")
	}
}

// TestFormBackspace verifies backspace trims the focused field and is a no-op on
// an empty field.
func TestFormBackspace(t *testing.T) {
	f := &form{fields: []field{{value: "abc", filter: filterPrintable}}}

	f.backspace()
	if f.fields[0].value != "ab" {
		t.Errorf("after backspace, field = %q, want %q", f.fields[0].value, "ab")
	}

	f.fields[0].value = ""
	f.backspace() // must not panic or underflow
	if f.fields[0].value != "" {
		t.Errorf("backspace on empty field = %q, want empty", f.fields[0].value)
	}
}

// TestFormBackspaceTrimsWholeRune verifies backspace removes an entire multibyte
// rune, leaving valid UTF-8 (a byte-trim would corrupt the string).
func TestFormBackspaceTrimsWholeRune(t *testing.T) {
	f := &form{fields: []field{{value: "a中😀", filter: filterPrintable}}}

	f.backspace() // drop the emoji (4 bytes)
	if f.fields[0].value != "a中" {
		t.Errorf("after backspace = %q, want %q", f.fields[0].value, "a中")
	}
	f.backspace() // drop the CJK char (3 bytes)
	if f.fields[0].value != "a" {
		t.Errorf("after second backspace = %q, want %q", f.fields[0].value, "a")
	}
	if !utf8.ValidString(f.fields[0].value) {
		t.Errorf("field is not valid UTF-8 after backspace: %q", f.fields[0].value)
	}
}

// TestFormTabWraps verifies tab cycles forward and backward with wrap-around.
func TestFormTabWraps(t *testing.T) {
	f := &form{fields: make([]field, 3)}

	f.tab(false)
	if f.focus != 1 {
		t.Errorf("after tab, focus = %d, want 1", f.focus)
	}
	f.tab(false)
	f.tab(false) // 1 -> 2 -> 0 (wrap)
	if f.focus != 0 {
		t.Errorf("after wrapping tab, focus = %d, want 0", f.focus)
	}
	f.tab(true) // 0 -> 2 (reverse wrap)
	if f.focus != 2 {
		t.Errorf("after reverse tab, focus = %d, want 2", f.focus)
	}
}

// TestFormValOutOfRange verifies val returns "" for out-of-range indices,
// keeping named accessors safe on a zero-value form.
func TestFormValOutOfRange(t *testing.T) {
	var f form
	if got := f.val(0); got != "" {
		t.Errorf("val on nil fields = %q, want empty", got)
	}
}

// TestDatapointFormDefaults verifies the datapoint form is constructed with the
// expected defaults and the provided value.
func TestDatapointFormDefaults(t *testing.T) {
	d := newDatapointForm("7.5")
	if d.value() != "7.5" {
		t.Errorf("value() = %q, want %q", d.value(), "7.5")
	}
	if d.comment() != "Added via buzz" {
		t.Errorf("comment() = %q, want %q", d.comment(), "Added via buzz")
	}
	if d.date() == "" {
		t.Error("date() should default to today, got empty")
	}
}

// TestDatapointFormFieldFilters verifies each datapoint field accepts/rejects
// the right characters via handleRune — the char-filter ↔ focus interaction the
// issue says only surfaced in manual TUI use.
func TestDatapointFormFieldFilters(t *testing.T) {
	d := newDatapointForm("")
	// Clear the constructor's pre-filled date/comment so each field starts empty.
	d.fields[dpDate].value = ""
	d.fields[dpComment].value = ""

	// Date field (focus 0): digits and dashes only.
	if d.handleRune('a') {
		t.Error("date field should reject letters")
	}
	typeInto(&d.form, "2024-01-15")
	if d.date() != "2024-01-15" {
		t.Errorf("date() = %q, want %q", d.date(), "2024-01-15")
	}

	// Value field (focus 1): digits, decimal, negative.
	d.tab(false)
	if d.handleRune('x') {
		t.Error("value field should reject letters")
	}
	typeInto(&d.form, "-3.5")
	if d.value() != "-3.5" {
		t.Errorf("value() = %q, want %q", d.value(), "-3.5")
	}

	// Comment field (focus 2): any printable rune, including command keys.
	d.tab(false)
	typeInto(&d.form, "trd note 中")
	if d.comment() != "trd note 中" {
		t.Errorf("comment() = %q, want %q", d.comment(), "trd note 中")
	}
}

// TestCreateGoalFormDefaults verifies the create-goal form's default field
// values.
func TestCreateGoalFormDefaults(t *testing.T) {
	c := newCreateGoalForm()
	cases := map[string]struct{ got, want string }{
		"slug":     {c.slug(), ""},
		"title":    {c.title(), ""},
		"goalType": {c.goalType(), "hustler"},
		"gunits":   {c.gunits(), "units"},
		"goaldate": {c.goaldate(), ""},
		"goalval":  {c.goalval(), "0"},
		"rate":     {c.rate(), "1"},
	}
	for name, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s() = %q, want %q", name, tc.got, tc.want)
		}
	}
}

// TestCreateGoalFormFieldFilters verifies the create-goal field filters,
// including the "null" prefix behaviour for the goaldate/goalval/rate fields.
func TestCreateGoalFormFieldFilters(t *testing.T) {
	c := newCreateGoalForm()

	// Slug (focus 0): alphanumeric, dash, underscore; no spaces.
	if c.handleRune(' ') {
		t.Error("slug should reject space")
	}
	typeInto(&c.form, "my-goal_1")
	if c.slug() != "my-goal_1" {
		t.Errorf("slug() = %q, want %q", c.slug(), "my-goal_1")
	}

	// Goal type (focus 2): letters only.
	c.focus = cgGoalType
	c.fields[cgGoalType].value = ""
	if c.handleRune('1') {
		t.Error("goal type should reject digits")
	}
	typeInto(&c.form, "biker")
	if c.goalType() != "biker" {
		t.Errorf("goalType() = %q, want %q", c.goalType(), "biker")
	}

	// Goaldate (focus 4): digits or the literal "null".
	c.focus = cgGoaldate
	if c.handleRune('x') {
		t.Error("goaldate should reject non-digit, non-null chars")
	}
	typeInto(&c.form, "null")
	if c.goaldate() != "null" {
		t.Errorf("goaldate() = %q, want %q", c.goaldate(), "null")
	}
}

// TestCreateGoalFormValidate verifies validate() delegates to the existing
// validator: exactly two of (goaldate, goalval, rate) must be provided.
func TestCreateGoalFormValidate(t *testing.T) {
	c := newCreateGoalForm() // goalval=0, rate=1, goaldate="" => two provided
	c.fields[cgSlug].value = "slug"
	c.fields[cgTitle].value = "Title"
	if got := c.validate(); got != "" {
		t.Errorf("validate() = %q, want no error", got)
	}

	// Provide all three -> invalid.
	c.fields[cgGoaldate].value = "1700000000"
	if got := c.validate(); got == "" {
		t.Error("validate() should fail when all three of goaldate/goalval/rate are set")
	}
}

// TestDatapointFormValidate verifies validate() delegates to the existing
// datapoint validator.
func TestDatapointFormValidate(t *testing.T) {
	d := newDatapointForm("5")
	if got := d.validate(); got != "" {
		t.Errorf("validate() with defaults = %q, want no error", got)
	}

	d.fields[dpValue].value = "not-a-number"
	if got := d.validate(); got == "" {
		t.Error("validate() should fail for non-numeric value")
	}
}
