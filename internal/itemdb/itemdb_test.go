package itemdb

import "testing"

func TestName(t *testing.T) {
	tests := []struct {
		id     int
		want   string
		wantOK bool
	}{
		{id: 0x001, want: "Copper Sword", wantOK: true},
		{id: 0x034, want: "Aura Racket", wantOK: true},
		{id: ArtifactBase, want: "Shuriken", wantOK: true},
		{id: 0x0AA, want: "Mjöllnir", wantOK: true},
		{id: 0x125, want: "Phoenix Down", wantOK: true},
		{id: 0x17D, want: "Striped Apple", wantOK: true},
		{id: -1, wantOK: false},
		{id: Count(), wantOK: false},
	}

	for _, tt := range tests {
		got, ok := Name(tt.id)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("Name(%#x) = %q, %v; want %q, %v", tt.id, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestTableShape(t *testing.T) {
	if got, want := Count(), 501; got != want {
		t.Fatalf("Count() = %d, want %d", got, want)
	}
	for id := 0; id < Count(); id++ {
		if name, _ := Name(id); name == "" {
			t.Errorf("item %#x has an empty name", id)
		}
	}
}
