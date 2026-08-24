package storage

import (
	"path/filepath"
	"testing"

	"github.com/macrocanvas/macrocanvas/internal/macro"
)

func TestMacroRoundTrip(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	m := macro.P10Sample()
	if err := db.UpsertMacro(m); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetMacro(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != m.Name || len(got.Nodes) != len(m.Nodes) {
		t.Fatalf("%+v", got)
	}
	list, err := db.ListMacros()
	if err != nil || len(list) != 1 {
		t.Fatalf("list %v %v", list, err)
	}
	if err := db.SaveRun("r1", m.ID, "running", map[string]string{"x": "1"}); err != nil {
		t.Fatal(err)
	}
	ids, err := db.IncompleteRuns()
	if err != nil || len(ids) != 1 {
		t.Fatalf("incomplete %v %v", ids, err)
	}
	if err := db.MarkAbandoned("r1"); err != nil {
		t.Fatal(err)
	}
	ids, _ = db.IncompleteRuns()
	if len(ids) != 0 {
		t.Fatal(ids)
	}
	_ = filepath.Join(dir, "macrocanvas.db")
}

func TestImportBounds(t *testing.T) {
	_, errs := macro.DecodeAndValidate([]byte(`{"name":"x","precision":"nope","nodes":[],"edges":[]}`))
	if len(errs) == 0 {
		t.Fatal("expected precision error")
	}
}
