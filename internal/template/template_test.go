package template

import "testing"

func TestRender(t *testing.T) {
	vars := map[string]string{"version": "1.24"}
	got, err := Render("apt-get install nginx={{ .version }}", vars)
	if err != nil {
		t.Fatal(err)
	}
	if got != "apt-get install nginx=1.24" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestRenderMissingKey(t *testing.T) {
	_, err := Render("{{ .missing }}", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}
