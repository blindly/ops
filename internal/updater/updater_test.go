package updater

import (
	"runtime"
	"strings"
	"testing"
)

func TestAssetName(t *testing.T) {
	name := assetName()
	if name == "" {
		t.Fatal("expected non-empty asset name")
	}
	if !strings.Contains(name, runtime.GOOS) {
		t.Fatalf("asset name %q missing OS %q", name, runtime.GOOS)
	}
	if !strings.Contains(name, runtime.GOARCH) {
		t.Fatalf("asset name %q missing arch %q", name, runtime.GOARCH)
	}
}

func TestFindAsset(t *testing.T) {
	want := assetName()
	r := &Release{
		Assets: []Asset{
			{Name: "ops-linux-amd64"},
			{Name: "ops-darwin-arm64"},
			{Name: want},
			{Name: "ops-windows-amd64.exe"},
		},
	}
	a := FindAsset(r)
	if a == nil {
		t.Fatalf("expected to find asset %s", want)
	}
	if a.Name != want {
		t.Fatalf("found wrong asset: %s", a.Name)
	}
}

func TestFindAssetMissing(t *testing.T) {
	r := &Release{
		Assets: []Asset{
			{Name: "ops-linux-amd64"},
		},
	}
	// If we're not linux/amd64, this should return nil.
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		if a := FindAsset(r); a != nil {
			t.Fatalf("expected nil for non-matching assets")
		}
	}
}
