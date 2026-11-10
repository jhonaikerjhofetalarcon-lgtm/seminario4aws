package presentacion

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"seminario4/dominio"
)

const duracionSesion = 8 * time.Hour
const nombreCookie = "aula_sesion"

type sesionPrivada struct {
	mu          sync.Mutex
	tokens      dominio.SesionAutenticada
	vence       time.Time
	accesoVence time.Time
	revocada    bool
}

type ventanaIntentos struct {
	veces int
	hasta time.Time
}

type controlAcceso struct {
	api        http.Handler
	auth       dominio.Autenticador
	permitidos map[string]bool
	origen     string
	segura     bool
	mu         sync.Mutex
	sesiones   map[[32]byte]*sesionPrivada
	intentos   map[string]ventanaIntentos
}

// Solo main entrega este manejador al servidor: todas las rutas /api quedan
// protegidas, salvo los endpoints de inicio, consulta y cierre de sesión.
func ConAutenticacion(api http.Handler, auth dominio.Autenticador, permitidos, origen string, segura bool) (http.Handler, error) {
	c := &controlAcceso{api: api, auth: auth, permitidos: map[string]bool{}, origen: origen, segura: segura, sesiones: map[[32]byte]*sesionPrivada{}, intentos: map[string]ventanaIntentos{}}
	for _, id := range strings.Split(permitidos, ",") {
		if id = strings.TrimSpace(id); id != "" {
			c.permitidos[id] = true
		}
	}
	if auth == nil || len(c.permitidos) == 0 {
		return nil, errors.New("Configura AUTH_USUARIOS_PERMITIDOS con los IDs autorizados de Supabase Auth.")
	}
	return c, nil
}

func (c *controlAcceso) cookie(w http.ResponseWriter, valor string, edad int) {
	http.SetCookie(w, &http.Cookie{Name: nombreCookie, Value: valor, Path: "/api", MaxAge: edad, HttpOnly: true, Secure: c.segura, SameSite: http.SameSiteLaxMode})
}

func (c *controlAcceso) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/") {
		c.api.ServeHTTP(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "Origin")
	if r.Header.Get("Origin") == c.origen {
		w.Header().Set("Access-Control-Allow-Origin", c.origen)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	}
	if (r.Method == http.MethodOptions || (r.Method != http.MethodGet && r.Method != http.MethodHead)) && r.Header.Get("Origin") != "" && r.Header.Get("Origin") != c.origen {
		responder(w, http.StatusForbidden, map[string]string{"error": "Origen no permitido."})
		return
	}
	if r.Method == http.MethodOptions {
		if r.Header.Get("Origin") != c.origen {
			responder(w, http.StatusForbidden, map[string]string{"error": "Origen no permitido."})
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	switch r.URL.Path {
	case "/api/auth/login":
		if r.Method == http.MethodPost {
			c.entrar(w, r)
			return
		}
	case "/api/auth/logout":
		if r.Method == http.MethodPost {
			c.salir(w, r)
			return
		}
	}
	usuario, valido := c.verificar(w, r)
	if !valido {
		return
	}
	if r.URL.Path == "/api/auth/sesion" && r.Method == http.MethodGet {
		responder(w, http.StatusOK, map[string]any{"usuario": usuario})
		return
	}
	c.api.ServeHTTP(w, r)
}

func (c *controlAcceso) permitirIntento(clave string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	ahora := time.Now()
	for id, v := range c.intentos {
		if ahora.After(v.hasta) {
			delete(c.intentos, id)
		}
	}
	v := c.intentos[clave]
	if v.hasta.IsZero() {
		v.hasta = ahora.Add(time.Minute)
	}
	if v.veces >= 10 || len(c.intentos) >= 10000 {
		return false
	}
	v.veces++
	c.intentos[clave] = v
	return true
}

func (c *controlAcceso) entrar(w http.ResponseWriter, r *http.Request) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	if !c.permitirIntento(ip) {
		w.Header().Set("Retry-After", "60")
		responder(w, http.StatusTooManyRequests, map[string]string{"error": "Demasiados intentos. Espera un minuto."})
		return
	}
	var entrada struct {
		Correo string `json:"correo"`
		Clave  string `json:"clave"`
	}
	if !leerJSON(w, r, &entrada) {
		return
	}
	if len(entrada.Correo) > 254 || len(entrada.Clave) > 1024 || entrada.Clave == "" || !strings.Contains(entrada.Correo, "@") {
		responder(w, http.StatusBadRequest, map[string]string{"error": "Escribe un correo y una contraseña válidos."})
		return
	}
	tokens, err := c.auth.IniciarSesion(r.Context(), strings.ToLower(strings.TrimSpace(entrada.Correo)), entrada.Clave)
	if err != nil {
		c.errorAuth(w, err)
		return
	}
	if !c.permitidos[tokens.Usuario.ID] {
		_ = c.auth.CerrarSesion(r.Context(), tokens.Acceso)
		responder(w, http.StatusForbidden, map[string]string{"error": "Esta cuenta no tiene acceso al sistema."})
		return
	}
	valor := make([]byte, 32)
	if _, err := rand.Read(valor); err != nil {
		c.errorAuth(w, dominio.ErrServicioAutenticacion)
		return
	}
	id := base64.RawURLEncoding.EncodeToString(valor)
	ahora := time.Now()
	c.mu.Lock()
	for k, s := range c.sesiones {
		if ahora.After(s.vence) {
			delete(c.sesiones, k)
		}
	}
	if anterior, err := r.Cookie(nombreCookie); err == nil {
		delete(c.sesiones, sha256.Sum256([]byte(anterior.Value)))
	}
	if len(c.sesiones) >= 1000 {
		c.mu.Unlock()
		c.errorAuth(w, dominio.ErrServicioAutenticacion)
		return
	}
	c.sesiones[sha256.Sum256([]byte(id))] = &sesionPrivada{tokens: tokens, vence: ahora.Add(duracionSesion), accesoVence: ahora.Add(time.Duration(tokens.VenceEn) * time.Second)}
	c.mu.Unlock()
	c.cookie(w, id, int(duracionSesion.Seconds()))
	responder(w, http.StatusOK, map[string]any{"usuario": tokens.Usuario})
}

func (c *controlAcceso) buscar(r *http.Request) (*sesionPrivada, [32]byte) {
	galleta, err := r.Cookie(nombreCookie)
	if err != nil || len(galleta.Value) != 43 {
		return nil, [32]byte{}
	}
	id := sha256.Sum256([]byte(galleta.Value))
	c.mu.Lock()
	s := c.sesiones[id]
	c.mu.Unlock()
	return s, id
}

func (c *controlAcceso) verificar(w http.ResponseWriter, r *http.Request) (dominio.UsuarioAutenticado, bool) {
	s, id := c.buscar(r)
	if s == nil {
		c.sinSesion(w)
		return dominio.UsuarioAutenticado{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.revocada || time.Now().After(s.vence) {
		c.eliminar(id)
		c.sinSesion(w)
		return dominio.UsuarioAutenticado{}, false
	}
	if time.Now().Add(30 * time.Second).After(s.accesoVence) {
		nueva, err := c.auth.RenovarSesion(r.Context(), s.tokens.Refresco)
		if err != nil {
			if errors.Is(err, dominio.ErrCredenciales) {
				c.eliminar(id)
				c.sinSesion(w)
			} else {
				c.errorAuth(w, err)
			}
			return dominio.UsuarioAutenticado{}, false
		}
		if nueva.Usuario.ID != s.tokens.Usuario.ID {
			c.eliminar(id)
			c.sinSesion(w)
			return dominio.UsuarioAutenticado{}, false
		}
		s.tokens = nueva
		s.accesoVence = time.Now().Add(time.Duration(nueva.VenceEn) * time.Second)
	}
	usuario, err := c.auth.VerificarSesion(r.Context(), s.tokens.Acceso)
	if err != nil {
		if errors.Is(err, dominio.ErrCredenciales) {
			c.eliminar(id)
			c.sinSesion(w)
		} else {
			c.errorAuth(w, err)
		}
		return dominio.UsuarioAutenticado{}, false
	}
	if usuario.ID != s.tokens.Usuario.ID || !c.permitidos[usuario.ID] {
		c.eliminar(id)
		c.sinSesion(w)
		return dominio.UsuarioAutenticado{}, false
	}
	return usuario, true
}

func (c *controlAcceso) eliminar(id [32]byte) {
	c.mu.Lock()
	delete(c.sesiones, id)
	c.mu.Unlock()
}

func (c *controlAcceso) sinSesion(w http.ResponseWriter) {
	c.cookie(w, "", -1)
	responder(w, http.StatusUnauthorized, map[string]string{"error": "Inicia sesión para continuar."})
}

func (c *controlAcceso) salir(w http.ResponseWriter, r *http.Request) {
	s, id := c.buscar(r)
	if s != nil {
		c.eliminar(id)
		s.mu.Lock()
		s.revocada = true
		_ = c.auth.CerrarSesion(r.Context(), s.tokens.Acceso)
		s.mu.Unlock()
	}
	c.cookie(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

func (c *controlAcceso) errorAuth(w http.ResponseWriter, err error) {
	if errors.Is(err, dominio.ErrCredenciales) {
		responder(w, http.StatusUnauthorized, map[string]string{"error": "Correo o contraseña incorrectos."})
		return
	}
	responder(w, http.StatusServiceUnavailable, map[string]string{"error": "No se pudo verificar el acceso. Intenta nuevamente."})
}
