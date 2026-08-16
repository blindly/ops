package ssh

import (
	"strings"
	"testing"
)

func TestUploadPath(t *testing.T) {
	path, err := uploadPath("bash", "install")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, "/tmp/ops-") || !strings.HasSuffix(path, ".sh") {
		t.Fatalf("unexpected bash path: %s", path)
	}
	if !strings.Contains(path, "-install") {
		t.Fatalf("bash path missing script name: %s", path)
	}

	path, err = uploadPath("powershell", "win")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, `C:\Windows\Temp\ops-`) || !strings.HasSuffix(path, ".ps1") {
		t.Fatalf("unexpected powershell path: %s", path)
	}
	if !strings.Contains(path, "-win") {
		t.Fatalf("powershell path missing script name: %s", path)
	}

	path, err = uploadPath("pwsh", "win")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, `C:\Windows\Temp\ops-`) || !strings.HasSuffix(path, ".ps1") {
		t.Fatalf("unexpected pwsh path: %s", path)
	}
	if !strings.Contains(path, "-win") {
		t.Fatalf("pwsh path missing script name: %s", path)
	}

	// Unknown interpreters default to POSIX shell behavior.
	path, err = uploadPath("python3", "bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, "/tmp/ops-") || !strings.HasSuffix(path, ".sh") {
		t.Fatalf("unexpected default path: %s", path)
	}
}

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		// No special characters: wrap in single quotes.
		"simple": "'simple'",
		// Empty string still needs quoting to stay a single argument.
		"": "''",
		// Spaces and shell metacharacters are contained within quotes.
		"my script.sh": "'my script.sh'",
		"a; rm -rf /":  "'a; rm -rf /'",
		"$(whoami)":    "'$(whoami)'",
		// Embedded single quotes use the POSIX escape idiom: close,
		// embed a quoted quote, reopen.
		"it's":      "'it'\"'\"'s'",
		"o'reilly":  "'o'\"'\"'reilly'",
		"'leading":  "''\"'\"'leading'",
		"trailing'": "'trailing'\"'\"''",
		"a'b'c":     "'a'\"'\"'b'\"'\"'c'",
	}
	for in, want := range cases {
		if got := ShellQuote(in); got != want {
			t.Errorf("ShellQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPowerShellQuote(t *testing.T) {
	cases := map[string]string{
		// No special characters: wrap in single quotes.
		"simple": "'simple'",
		"":       "''",
		// Spaces and metacharacters are contained within quotes.
		"my script.ps1":       "'my script.ps1'",
		"a; Remove-Item C:\\": "'a; Remove-Item C:\\'",
		// PowerShell escapes a single quote by doubling it.
		"it's":      "'it''s'",
		"o'reilly":  "'o''reilly'",
		"'leading":  "'''leading'",
		"trailing'": "'trailing'''",
		"a'b'c":     "'a''b''c'",
	}
	for in, want := range cases {
		if got := PowerShellQuote(in); got != want {
			t.Errorf("PowerShellQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCleanupCommand(t *testing.T) {
	// POSIX interpreter: rm -f with a shell-quoted path; names with spaces
	// or metacharacters must not break out of the quoted argument.
	got := cleanupCommand("bash", "/tmp/ops-deadbeef-install.sh")
	want := "rm -f '/tmp/ops-deadbeef-install.sh'"
	if got != want {
		t.Errorf("posix cleanup = %q, want %q", got, want)
	}

	spaced := cleanupCommand("bash", "/tmp/ops-deadbeef-my script.sh")
	if !strings.Contains(spaced, "'/tmp/ops-deadbeef-my script.sh'") {
		t.Errorf("spaced name not safely quoted: %q", spaced)
	}
	if strings.Count(spaced, "'") != 2 {
		t.Errorf("spaced name has unbalanced quotes: %q", spaced)
	}
	// Ensure a shell metacharacter in the name cannot escape quoting.
	injected := cleanupCommand("bash", "/tmp/x; rm -rf /")
	if strings.Contains(injected, "; rm") && !strings.HasPrefix(injected, "rm -f '/tmp/x; rm -rf /'") {
		t.Errorf("metacharacter not safely quoted: %q", injected)
	}

	// PowerShell interpreter: Remove-Item -Force with a quoted path.
	for _, interp := range []string{"powershell", "pwsh"} {
		got := cleanupCommand(interp, `C:\Windows\Temp\ops-deadbeef-win.ps1`)
		want := "Remove-Item -Force 'C:\\Windows\\Temp\\ops-deadbeef-win.ps1'"
		if got != want {
			t.Errorf("%s cleanup = %q, want %q", interp, got, want)
		}
	}

	// PowerShell interpreter: embedded single quotes in the path must be
	// escaped by doubling, not by the POSIX '"'"' idiom.
	for _, interp := range []string{"powershell", "pwsh"} {
		got := cleanupCommand(interp, `C:\Windows\Temp\it's.ps1`)
		want := "Remove-Item -Force 'C:\\Windows\\Temp\\it''s.ps1'"
		if got != want {
			t.Errorf("%s embedded-quote cleanup = %q, want %q", interp, got, want)
		}
		if strings.Contains(got, `'"'"'`) {
			t.Errorf("%s cleanup used POSIX escape idiom: %q", interp, got)
		}
	}

	// Unknown interpreters fall back to the POSIX command.
	if got := cleanupCommand("python3", "/tmp/ops-x.sh"); !strings.HasPrefix(got, "rm -f ") {
		t.Errorf("unknown interpreter did not fall back to rm: %q", got)
	}
}
