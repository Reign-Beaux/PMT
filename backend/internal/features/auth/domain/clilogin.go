package domain

import (
	"crypto/rand"
	"math/big"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

// Reglas del inicio de sesión del CLI pmt por navegador.
const (
	// CLILoginTTL es lo que vale un código desde que el CLI lo pide.
	CLILoginTTL = 10 * time.Minute
	// MaxPendingCLILogins es el tope de códigos pendientes y vigentes en toda la
	// instancia: pedir un código es público, así que nadie puede llenar la tabla.
	MaxPendingCLILogins = 20
	// CLIPollInterval son los segundos entre sondeos que se piden al CLI.
	CLIPollInterval = 2
	// maxMachineLength deja el nombre del token ("pmt <máquina>") en 100 caracteres o menos.
	maxMachineLength = 60
)

// Estados guardados de un inicio de sesión del CLI.
const (
	CLIStatePending    = "pending"
	CLIStateAuthorized = "authorized"
	CLIStateCancelled  = "cancelled"
	CLIStateConsumed   = "consumed"
)

// Estados que ven el CLI y la pantalla /cli. Un código entregado cuenta como
// caducado: se usa una vez.
const (
	CLIStatusPending    = "pending"
	CLIStatusAuthorized = "authorized"
	CLIStatusCancelled  = "cancelled"
	CLIStatusExpired    = "expired"
)

var (
	// ErrCLILoginNotFound indica que el código o el secreto no existen.
	ErrCLILoginNotFound = kernel.NotFound("código no encontrado")
	// ErrCLILoginClosed indica que el código ya se autorizó, se canceló o caducó.
	ErrCLILoginClosed = kernel.Conflict("el código ya se usó, se canceló o caducó")
	// ErrTooManyCLILogins indica que se llegó al tope de códigos pendientes.
	ErrTooManyCLILogins = kernel.Conflict("hay demasiados inicios de sesión pendientes; intenta en unos minutos")
	// ErrMachineInvalid indica que el nombre de la máquina está vacío, es muy largo o tiene caracteres de control.
	ErrMachineInvalid = kernel.Invalid("el nombre de la máquina debe tener entre 1 y 60 caracteres, sin caracteres de control")
	// ErrPollSecretRequired indica que el sondeo no trae el secreto.
	ErrPollSecretRequired = kernel.Invalid("falta el secreto de sondeo")
	// ErrCLICodeTaken indica que el código generado ya existe: se genera otro.
	ErrCLICodeTaken = kernel.Conflict("el código ya existe")
)

var cliCodePattern = regexp.MustCompile(`^[A-Z]{4}-[0-9]{4}$`)

// Letras del código sin I ni O, que se confunden con 1 y 0.
const cliCodeLetters = "ABCDEFGHJKLMNPQRSTUVWXYZ"

// CLILogin es un inicio de sesión del CLI como lo ve la pantalla /cli.
type CLILogin struct {
	ID        uuid.UUID
	Code      string
	Machine   string
	State     string
	ExpiresAt time.Time
}

// Status es el estado visible a una hora dada.
func (l CLILogin) Status(now time.Time) string {
	return CLILoginStatus(l.State, l.ExpiresAt, now)
}

// NewCLILogin es lo que se guarda al pedir un código: solo el hash del secreto.
type NewCLILogin struct {
	ID        uuid.UUID
	Code      string
	PollHash  []byte
	Machine   string
	ExpiresAt time.Time
}

// IssuedToken es el token que se entrega al sondear un código autorizado. Se
// genera antes de abrir la transacción y se guarda solo si se entrega.
type IssuedToken struct {
	ID    uuid.UUID
	Token string
}

// CLIPoll es el resultado de un sondeo. UserID, Username y TokenID solo vienen
// con CLIStatusAuthorized.
type CLIPoll struct {
	Status   string
	UserID   uuid.UUID
	Username string
	TokenID  uuid.UUID
}

// CLILoginStatus traduce el estado guardado al visible: lo vencido y lo ya
// entregado responden caducado.
func CLILoginStatus(state string, expires, now time.Time) string {
	switch {
	case state == CLIStateCancelled:
		return CLIStatusCancelled
	case state == CLIStateConsumed || !expires.After(now):
		return CLIStatusExpired
	case state == CLIStateAuthorized:
		return CLIStatusAuthorized
	default:
		return CLIStatusPending
	}
}

// NewCLICode genera un código corto para que la persona lo compare: cuatro
// letras, guion y cuatro dígitos (ABCD-1234).
func NewCLICode() (string, error) {
	var b strings.Builder
	for i := range 9 {
		if i == 4 {
			b.WriteByte('-')
			continue
		}
		alphabet := cliCodeLetters
		if i > 4 {
			alphabet = "0123456789"
		}
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		b.WriteByte(alphabet[n.Int64()])
	}
	return b.String(), nil
}

// NormalizeCLICode acepta el código con minúsculas, espacios o sin guion.
// Uno que no tiene la forma de un código responde que no existe.
func NormalizeCLICode(code string) (string, error) {
	code = strings.ToUpper(strings.Join(strings.Fields(code), ""))
	if len(code) == 8 && !strings.Contains(code, "-") {
		code = code[:4] + "-" + code[4:]
	}
	if !cliCodePattern.MatchString(code) {
		return "", ErrCLILoginNotFound
	}
	return code, nil
}

// ValidateMachine quita espacios y exige de 1 a 60 caracteres sin control.
func ValidateMachine(machine string) (string, error) {
	machine = strings.TrimSpace(machine)
	if machine == "" || utf8.RuneCountInString(machine) > maxMachineLength {
		return "", ErrMachineInvalid
	}
	for _, r := range machine {
		if unicode.IsControl(r) {
			return "", ErrMachineInvalid
		}
	}
	return machine, nil
}

// CLITokenName es el nombre del token que crea el CLI, como los de la cuenta.
func CLITokenName(machine string) string {
	return "pmt " + machine
}

// CLILoginURL es la pantalla del frontend donde se autoriza el código.
func CLILoginURL(base, code string) string {
	return strings.TrimRight(base, "/") + "/cli?codigo=" + code
}
