package presentacion

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"seminario4/dominio"
)

type authDePrueba struct {
	usuario   dominio.UsuarioAutenticado
	fallo     error
	renovadas int
	cerradas  int
}

func (a *authDePrueba) IniciarSesion(_ context.Context, correo, clave string) (dominio.SesionAutenticada, error) {
	if a.fallo != nil {
		return dominio.SesionAutenticada{}, a.fallo
	}
	if correo != "admin@colegio.com" || clave != "clave-de-prueba" {
		return dominio.SesionAutenticada{}, dominio.ErrCredenciales
	}
	return dominio.SesionAutenticada{Acceso: "token-solo-servidor", Refresco: "refresco-solo-servidor", VenceEn: 3600, Usuario: a.usuario}, nil
}
func (a *authDePrueba) VerificarSesion(context.Context, string) (dominio.UsuarioAutenticado, error) {
	return a.usuario, a.fallo
}
func (a *authDePrueba) RenovarSesion(context.Context, string) (dominio.SesionAutenticada, error) {
	a.renovadas++
	return dominio.SesionAutenticada{Acceso: "renovado", Refresco: "rotado", VenceEn: 3600, Usuario: a.usuario}, a.fallo
}
func (a *authDePrueba) CerrarSesion(context.Context, string) error { a.cerradas++; return a.fallo }

func prepararAcceso(t *testing.T, segura bool) (*controlAcceso, *authDePrueba, *int) {
	t.Helper()
	auth := &authDePrueba{usuario: dominio.UsuarioAutenticado{ID: "admin-autorizado", Correo: "admin@colegio.com"}}
	llamadas := new(int)
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { *llamadas++; w.WriteHeader(200) })
	h, err := ConAutenticacion(api, auth, "admin-autorizado", "http://localhost:3000", segura)
	if err != nil {
		t.Fatal(err)
	}
	return h.(*controlAcceso), auth, llamadas
}

func peticionAcceso(c *controlAcceso, metodo, ruta, cuerpo string, cookie *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
	r.RemoteAddr = "192.0.2.10:12345"
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	c.ServeHTTP(w, r)
	return w
}
func entrarPrueba(t *testing.T, c *controlAcceso) *http.Cookie {
	t.Helper()
	w := peticionAcceso(c, "POST", "/api/auth/login", `{"correo":"admin@colegio.com","clave":"clave-de-prueba"}`, nil)
	if w.Code != 200 {
		t.Fatalf("Login: %d", w.Code)
	}
	return w.Result().Cookies()[0]
}

func TestAccesoExigeSesionParaTodasLasOperaciones(t *testing.T) {
	c, _, llamadas := prepararAcceso(t, false)
	for _, caso := range []struct{ metodo, ruta string }{{"GET", "/api/resumen"}, {"GET", "/api/alumnos"}, {"POST", "/api/alumnos"}, {"PUT", "/api/alumnos/1"}, {"DELETE", "/api/matriculas/1"}, {"GET", "/api/auth/sesion"}} {
		if w := peticionAcceso(c, caso.metodo, caso.ruta, "{}", nil); w.Code != 401 {
			t.Fatalf("Se permitio %s %s", caso.metodo, caso.ruta)
		}
	}
	for _, token := range []string{"inventada", strings.Repeat("a", 43)} {
		if w := peticionAcceso(c, "GET", "/api/resumen", "", &http.Cookie{Name: nombreCookie, Value: token}); w.Code != 401 {
			t.Fatal("Se aceptó una cookie inventada")
		}
	}
	if *llamadas != 0 {
		t.Fatal("Se llamó a la API sin autorización")
	}
}

func TestLoginCookiePrivadaYCierreInvalidaReuso(t *testing.T) {
	c, auth, llamadas := prepararAcceso(t, true)
	w := peticionAcceso(c, "POST", "/api/auth/login", `{"correo":"admin@colegio.com","clave":"clave-de-prueba"}`, nil)
	if w.Code != 200 || strings.Contains(w.Body.String(), "token") || strings.Contains(w.Body.String(), "refresco") {
		t.Fatal("El login filtró tokens o falló")
	}
	galleta := w.Result().Cookies()[0]
	if !galleta.HttpOnly || !galleta.Secure || galleta.Path != "/api" || galleta.SameSite != http.SameSiteLaxMode || galleta.MaxAge != 28800 {
		t.Fatal("La cookie no tiene las protecciones esperadas")
	}
	if w := peticionAcceso(c, "GET", "/api/resumen", "", galleta); w.Code != 200 || *llamadas != 1 {
		t.Fatal("No funcionó la sesión válida")
	}
	if w := peticionAcceso(c, "GET", "/api/auth/sesion", "", galleta); w.Code != 200 || !strings.Contains(w.Body.String(), "admin@colegio.com") {
		t.Fatal("No devolvió el usuario actual")
	}
	if w := peticionAcceso(c, "POST", "/api/auth/logout", "", galleta); w.Code != 204 || w.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("No se cerró la sesión")
	}
	if w := peticionAcceso(c, "GET", "/api/resumen", "", galleta); w.Code != 401 {
		t.Fatal("Se pudo reutilizar una sesión cerrada")
	}
	if auth.cerradas != 1 {
		t.Fatal("No se revocó la sesión remota")
	}
}

func TestLoginRechazaCredencialesYCuentasNoAutorizadas(t *testing.T) {
	c, auth, _ := prepararAcceso(t, false)
	if w := peticionAcceso(c, "POST", "/api/auth/login", `{"correo":"admin@colegio.com","clave":"incorrecta"}`, nil); w.Code != 401 || len(c.sesiones) != 0 {
		t.Fatal("Se aceptó una contraseña incorrecta")
	}
	auth.usuario.ID = "otro-usuario"
	if w := peticionAcceso(c, "POST", "/api/auth/login", `{"correo":"admin@colegio.com","clave":"clave-de-prueba"}`, nil); w.Code != 403 || len(c.sesiones) != 0 || auth.cerradas != 1 {
		t.Fatal("Una cuenta sin permiso pudo ingresar")
	}
	if _, err := ConAutenticacion(c.api, auth, "", c.origen, false); err == nil {
		t.Fatal("Se permitió un servidor sin lista de administradores")
	}
}

func TestSesionesExpiranRenuevanYNoOcultanCaidas(t *testing.T) {
	c, auth, _ := prepararAcceso(t, false)
	galleta := entrarPrueba(t, c)
	r := httptest.NewRequest("GET", "/api/resumen", nil)
	r.AddCookie(galleta)
	s, _ := c.buscar(r)
	s.accesoVence = time.Now().Add(-time.Second)
	if w := peticionAcceso(c, "GET", "/api/resumen", "", galleta); w.Code != 200 || auth.renovadas != 1 {
		t.Fatal("No se renovó el token privado")
	}
	auth.fallo = dominio.ErrServicioAutenticacion
	if w := peticionAcceso(c, "GET", "/api/resumen", "", galleta); w.Code != 503 || len(c.sesiones) != 1 {
		t.Fatal("Una caída remota debe conservar la sesión y rechazar la operación")
	}
	auth.fallo = nil
	s.vence = time.Now().Add(-time.Second)
	if w := peticionAcceso(c, "GET", "/api/resumen", "", galleta); w.Code != 401 || len(c.sesiones) != 0 {
		t.Fatal("Se aceptó una sesión vencida")
	}
}

func TestLoginProtegeOrigenCORSYLimitaIntentosPorIP(t *testing.T) {
	c, _, _ := prepararAcceso(t, false)
	r := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"correo":"admin@colegio.com","clave":"clave-de-prueba"}`))
	r.Header.Set("Origin", "https://ajeno.test")
	w := httptest.NewRecorder()
	c.ServeHTTP(w, r)
	if w.Code != 403 || len(c.sesiones) != 0 {
		t.Fatal("Se permitió un login de otro origen")
	}
	r = httptest.NewRequest("OPTIONS", "/api/auth/login", nil)
	r.Header.Set("Origin", c.origen)
	w = httptest.NewRecorder()
	c.ServeHTTP(w, r)
	if w.Code != 204 || w.Header().Get("Access-Control-Allow-Credentials") != "true" || w.Header().Get("Access-Control-Allow-Origin") != c.origen {
		t.Fatal("CORS no permite las cookies del frontend")
	}
	for i := 0; i < 10; i++ {
		r = httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"correo":"admin@colegio.com","clave":"incorrecta"}`))
		r.RemoteAddr = "192.0.2.20:" + strings.Repeat("1", i+1)
		w = httptest.NewRecorder()
		c.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatal("El límite se activó antes de diez intentos")
		}
	}
	r.RemoteAddr = "192.0.2.20:9999"
	w = httptest.NewRecorder()
	c.ServeHTTP(w, r)
	if w.Code != 429 || w.Header().Get("Retry-After") != "60" {
		t.Fatal("Cambiar de puerto permitió evadir el límite por IP")
	}
}
