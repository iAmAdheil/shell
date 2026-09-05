package session

import "testing"

// The exported tests cover what a caller sees: distinct, stable colours. This
// one is inside the package, because the rule it checks — colours go out in
// join order — can only be stated against the palette itself.
func TestColorsGoOutInJoinOrder(t *testing.T) {
	s := &Session{colors: map[string]string{}}

	first := s.colorForLocked("acct-1")
	second := s.colorForLocked("acct-2")

	if first != palette[0] {
		t.Errorf("first User got %q, want %q", first, palette[0])
	}
	if second != palette[1] {
		t.Errorf("second User got %q, want %q", second, palette[1])
	}
	if again := s.colorForLocked("acct-1"); again != first {
		t.Errorf("asking twice gave %q then %q, want the same colour", first, again)
	}
}

// A Session with more Users than colours must still answer for every one.
func TestThePaletteStartsAgainWhenItRunsOut(t *testing.T) {
	s := &Session{colors: map[string]string{}}

	for i := range palette {
		s.colorForLocked(string(rune('a' + i)))
	}

	if got := s.colorForLocked("one-too-many"); got != palette[0] {
		t.Errorf("the User after the palette ran out got %q, want %q", got, palette[0])
	}
}
