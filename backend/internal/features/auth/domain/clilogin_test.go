package domain

import (
	"testing"
	"time"
)

func TestNewCLICodeTieneLaFormaYNoUsaIniO(t *testing.T) {
	seen := map[string]bool{}
	for range 200 {
		code, err := NewCLICode()
		if err != nil {
			t.Fatal(err)
		}
		if !cliCodePattern.MatchString(code) {
			t.Fatalf("código %q", code)
		}
		for _, r := range code[:4] {
			if r == 'I' || r == 'O' {
				t.Fatalf("código con I u O: %q", code)
			}
		}
		seen[code] = true
	}
	if len(seen) < 190 {
		t.Fatalf("códigos repetidos: %d distintos de 200", len(seen))
	}
}

func TestNormalizeCLICode(t *testing.T) {
	for in, want := range map[string]string{"ABCD-1234": "ABCD-1234", " abcd-1234 ": "ABCD-1234", "abcd1234": "ABCD-1234", "AB CD-12 34": "ABCD-1234"} {
		if got, err := NormalizeCLICode(in); err != nil || got != want {
			t.Errorf("%q → %q, %v", in, got, err)
		}
	}
	for _, in := range []string{"", "ABCD-123", "ABC1-1234", "ABCD_1234", "../x"} {
		if _, err := NormalizeCLICode(in); err != ErrCLILoginNotFound {
			t.Errorf("%q debería no existir: %v", in, err)
		}
	}
}

func TestCLILoginStatus(t *testing.T) {
	now := time.Now()
	later, before := now.Add(time.Minute), now.Add(-time.Minute)
	for _, tc := range []struct {
		state   string
		expires time.Time
		want    string
	}{
		{CLIStatePending, later, CLIStatusPending},
		{CLIStateAuthorized, later, CLIStatusAuthorized},
		{CLIStateCancelled, later, CLIStatusCancelled},
		{CLIStateConsumed, later, CLIStatusExpired},
		{CLIStatePending, before, CLIStatusExpired},
		{CLIStateAuthorized, before, CLIStatusExpired},
		{CLIStateCancelled, before, CLIStatusCancelled},
		{CLIStatePending, now, CLIStatusExpired},
	} {
		if got := CLILoginStatus(tc.state, tc.expires, now); got != tc.want {
			t.Errorf("%s: %s, se esperaba %s", tc.state, got, tc.want)
		}
	}
}

func TestValidateMachineYNombreDelToken(t *testing.T) {
	if m, err := ValidateMachine("  laptop  "); err != nil || CLITokenName(m) != "pmt laptop" {
		t.Fatalf("%q %v", m, err)
	}
	long := ""
	for range 60 {
		long += "ñ"
	}
	name, err := ValidateMachine(long)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateTokenName(CLITokenName(name)); err != nil {
		t.Fatalf("el nombre del token con la máquina más larga no vale: %v", err)
	}
	if CLILoginURL("https://pmt.example.com/", "ABCD-1234") != "https://pmt.example.com/cli?codigo=ABCD-1234" {
		t.Fatal("url")
	}
}
