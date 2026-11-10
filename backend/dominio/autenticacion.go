package dominio

import (
	"context"
	"errors"
)

var ErrCredenciales = errors.New("Correo o contraseña incorrectos.")
var ErrServicioAutenticacion = errors.New("No se pudo verificar el acceso. Intenta nuevamente.")

type UsuarioAutenticado struct {
	ID     string `json:"id"`
	Correo string `json:"email"`
}

// Los tokens solo existen en Go: no se incluyen en respuestas al navegador.
type SesionAutenticada struct {
	Acceso   string             `json:"access_token"`
	Refresco string             `json:"refresh_token"`
	VenceEn  int                `json:"expires_in"`
	Usuario  UsuarioAutenticado `json:"user"`
}

type Autenticador interface {
	IniciarSesion(context.Context, string, string) (SesionAutenticada, error)
	VerificarSesion(context.Context, string) (UsuarioAutenticado, error)
	RenovarSesion(context.Context, string) (SesionAutenticada, error)
	CerrarSesion(context.Context, string) error
}
