package infraestructura

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthSupabaseProtocoloYErroresSinSecretos(t *testing.T) {
	paso := 0
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("apikey") != "sb_publishable_prueba" {
			t.Error("No se utilizó la clave pública")
		}
		switch paso {
		case 0:
			var datos map[string]string
			_ = json.NewDecoder(r.Body).Decode(&datos)
			if r.URL.RequestURI() != "/auth/v1/token?grant_type=password" || datos["email"] != "admin@colegio.com" || datos["password"] != "clave-privada" || r.Header.Get("Authorization") != "" {
				t.Error("Petición de login incorrecta")
			}
			_, _ = w.Write([]byte(`{"access_token":"acceso","refresh_token":"refresco","expires_in":3600,"user":{"id":"admin","email":"admin@colegio.com"}}`))
		case 1:
			if r.URL.Path != "/auth/v1/user" || r.Header.Get("Authorization") != "Bearer acceso" {
				t.Error("No se validó el token con Supabase")
			}
			_, _ = w.Write([]byte(`{"id":"admin","email":"admin@colegio.com"}`))
		case 2:
			if r.URL.RequestURI() != "/auth/v1/token?grant_type=refresh_token" {
				t.Error("No se renovó mediante refresh_token")
			}
			_, _ = w.Write([]byte(`{"access_token":"nuevo","refresh_token":"rotado","expires_in":3600,"user":{"id":"admin","email":"admin@colegio.com"}}`))
		case 3:
			if r.URL.RequestURI() != "/auth/v1/logout?scope=local" || r.Header.Get("Authorization") != "Bearer nuevo" {
				t.Error("El logout debe limitarse a esta sesión")
			}
			w.WriteHeader(204)
		default:
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"message":"clave-privada-no-mostrar"}`))
		}
		paso++
	}))
	defer servidor.Close()
	auth, err := NuevaAutenticacionSupabase(servidor.URL, "sb_publishable_prueba")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := auth.IniciarSesion(ctx, "admin@colegio.com", "clave-privada"); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.VerificarSesion(ctx, "acceso"); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.RenovarSesion(ctx, "refresco"); err != nil {
		t.Fatal(err)
	}
	if err := auth.CerrarSesion(ctx, "nuevo"); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.IniciarSesion(ctx, "admin@colegio.com", "incorrecta"); !errors.Is(err, ErrCredenciales) || strings.Contains(err.Error(), "no-mostrar") {
		t.Fatal("La respuesta filtró información remota")
	}
	if _, err := NuevaAutenticacionSupabase("http://remoto.test", "sb_publishable_prueba"); err == nil {
		t.Fatal("Se permitió HTTP externo")
	}
	if _, err := NuevaAutenticacionSupabase(servidor.URL, "sb_secret_no_usar"); err == nil {
		t.Fatal("Auth debe utilizar la clave pública")
	}
}
