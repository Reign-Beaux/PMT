package domain

import (
	"errors"
	"testing"

	"pmt/internal/kernel"
)

func TestChangeProductVersionOnlyInWorking(t *testing.T) {
	for _, status := range []Status{StatusPlanning, StatusStandBy, StatusInDev, StatusStopped, StatusCancelled} {
		t.Run(string(status), func(t *testing.T) {
			p := Project{Status: status, ProductVersion: "1.0.0"}

			_, err := p.ChangeProductVersion("1.1.0", "motivo", kernel.Owner)

			if !errors.Is(err, ErrProductVersionNotWorking) || kindOf(err) != kernel.KindConflict {
				t.Fatalf("err = %v", err)
			}
			if p.ProductVersion != "1.0.0" {
				t.Errorf("versión = %q", p.ProductVersion)
			}
		})
	}
}

func TestChangeProductVersionValidates(t *testing.T) {
	tests := []struct {
		name    string
		current string
		next    string
		reason  string
		want    error
	}{
		{"sube el parche", "1.0.0", "1.0.1", "bug", nil},
		{"sube el menor", "1.0.9", "1.1.0", "etapa", nil},
		{"sube el mayor", "1.9.9", "2.0.0", "cambio grande", nil},
		{"compara números, no texto", "1.9.0", "1.10.0", "etapa", nil},
		{"actual sin forma acepta cualquiera", "beta", "0.0.1", "inicio", nil},
		{"actual vacía acepta cualquiera", "", "0.1.0", "inicio", nil},
		{"recorta espacios", "1.0.0", " 1.0.1 ", "bug", nil},
		{"igual", "1.0.0", "1.0.0", "x", ErrProductVersionNotGreater},
		{"menor", "1.2.0", "1.1.9", "x", ErrProductVersionNotGreater},
		{"dos partes", "1.0.0", "1.1", "x", ErrProductVersionFormat},
		{"cuatro partes", "1.0.0", "1.1.0.0", "x", ErrProductVersionFormat},
		{"cero a la izquierda", "1.0.0", "1.01.0", "x", ErrProductVersionFormat},
		{"con signo", "1.0.0", "+2.0.0", "x", ErrProductVersionFormat},
		{"con prefijo v", "1.0.0", "v2.0.0", "x", ErrProductVersionFormat},
		{"con sufijo", "1.0.0", "2.0.0-beta", "x", ErrProductVersionFormat},
		{"vacía", "1.0.0", "", "x", ErrProductVersionFormat},
		{"desborda", "1.0.0", "99999999999999999999.0.0", "x", ErrProductVersionFormat},
		{"motivo vacío", "1.0.0", "1.0.1", "  ", ErrProductVersionReasonRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Project{Status: StatusWorking, ProductVersion: tt.current}

			change, err := p.ChangeProductVersion(tt.next, tt.reason, kernel.Actor{Name: "pp", Profile: kernel.ProfilePP})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, quiero %v", err, tt.want)
			}
			if tt.want != nil {
				if kindOf(err) != kernel.KindInvalid {
					t.Errorf("err = %v, quiero validación", err)
				}
				if p.ProductVersion != tt.current {
					t.Errorf("versión = %q", p.ProductVersion)
				}
				return
			}
			if change.From != tt.current || change.To != p.ProductVersion || change.ActorName != "pp" || change.ActorProfile != "PP" || change.Reason != tt.reason {
				t.Errorf("registro = %+v, versión = %q", change, p.ProductVersion)
			}
		})
	}
}

func kindOf(err error) kernel.ErrorKind {
	var business *kernel.Error
	if errors.As(err, &business) {
		return business.Kind
	}
	return 0
}
