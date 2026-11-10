package infraestructura

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"seminario4/dominio"
)

var ErrCredenciales = dominio.ErrCredenciales
var ErrServicioAutenticacion = dominio.ErrServicioAutenticacion

type AutenticacionSupabase struct {
	direccion string
	publica   string
	cliente   *http.Client
}

func NuevaAutenticacionSupabase(direccion, publica string) (*AutenticacionSupabase, error) {
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(direccion), "/"))
	if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost"))) {
		return nil, errors.New("Configura la URL raíz de Supabase para el login.")
	}
	if strings.TrimSpace(publica) == "" || clavePrivadaValida(publica) {
		return nil, errors.New("Configura SUPABASE_PUBLISHABLE_KEY con la clave pública del proyecto.")
	}
	return &AutenticacionSupabase{
		direccion: u.String() + "/auth/v1",
		publica:   strings.TrimSpace(publica),
		cliente: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}},
	}, nil
}

func (a *AutenticacionSupabase) solicitar(ctx context.Context, metodo, ruta, acceso string, datos, destino any) error {
	var cuerpo io.Reader
	if datos != nil {
		contenido, err := json.Marshal(datos)
		if err != nil {
			return ErrServicioAutenticacion
		}
		cuerpo = bytes.NewReader(contenido)
	}
	r, err := http.NewRequestWithContext(ctx, metodo, a.direccion+ruta, cuerpo)
	if err != nil {
		return ErrServicioAutenticacion
	}
	r.Header.Set("apikey", a.publica)
	r.Header.Set("Content-Type", "application/json")
	if acceso != "" {
		r.Header.Set("Authorization", "Bearer "+acceso)
	}
	respuesta, err := a.cliente.Do(r)
	if err != nil {
		return ErrServicioAutenticacion
	}
	defer respuesta.Body.Close()
	if respuesta.StatusCode == 400 || respuesta.StatusCode == 401 || respuesta.StatusCode == 403 {
		return ErrCredenciales
	}
	if respuesta.StatusCode < 200 || respuesta.StatusCode >= 300 {
		return ErrServicioAutenticacion
	}
	if destino != nil && json.NewDecoder(io.LimitReader(respuesta.Body, 1024*1024)).Decode(destino) != nil {
		return ErrServicioAutenticacion
	}
	return nil
}

func (a *AutenticacionSupabase) token(ctx context.Context, ruta string, datos any) (dominio.SesionAutenticada, error) {
	var sesion dominio.SesionAutenticada
	err := a.solicitar(ctx, http.MethodPost, ruta, "", datos, &sesion)
	if err == nil && (sesion.Acceso == "" || sesion.Refresco == "" || sesion.Usuario.ID == "" || sesion.Usuario.Correo == "" || sesion.VenceEn <= 0) {
		err = ErrServicioAutenticacion
	}
	return sesion, err
}

func (a *AutenticacionSupabase) IniciarSesion(ctx context.Context, correo, clave string) (dominio.SesionAutenticada, error) {
	return a.token(ctx, "/token?grant_type=password", map[string]string{"email": correo, "password": clave})
}

func (a *AutenticacionSupabase) RenovarSesion(ctx context.Context, refresco string) (dominio.SesionAutenticada, error) {
	return a.token(ctx, "/token?grant_type=refresh_token", map[string]string{"refresh_token": refresco})
}

func (a *AutenticacionSupabase) VerificarSesion(ctx context.Context, acceso string) (dominio.UsuarioAutenticado, error) {
	var usuario dominio.UsuarioAutenticado
	err := a.solicitar(ctx, http.MethodGet, "/user", acceso, nil, &usuario)
	if err == nil && (usuario.ID == "" || usuario.Correo == "") {
		err = ErrCredenciales
	}
	return usuario, err
}

func (a *AutenticacionSupabase) CerrarSesion(ctx context.Context, acceso string) error {
	return a.solicitar(ctx, http.MethodPost, "/logout?scope=local", acceso, nil, nil)
}
