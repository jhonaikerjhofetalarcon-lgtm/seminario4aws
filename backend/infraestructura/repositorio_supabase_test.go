package infraestructura

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"seminario4/dominio"
)

// Probamos el protocolo HTTP sin necesitar una cuenta ni credenciales reales.
func prepararSupabase(t *testing.T, atender http.HandlerFunc) *RepositorioSupabase {
	t.Helper()
	servidor := httptest.NewServer(atender)
	t.Cleanup(servidor.Close)
	repositorio, err := NuevoRepositorioSupabase(servidor.URL, "sb_secret_prueba_solo_local")
	if err != nil {
		t.Fatal(err)
	}
	return repositorio
}

func TestSupabasePeticionesCRUD(t *testing.T) {
	paso := 0
	casos := []struct{ metodo, ruta, cuerpo, respuesta string }{
		{"POST", "/rest/v1/rpc/consultar_sistema_academico", `{}`, `{"alumnos":[],"matriculas":[]}`},
		{"POST", "/rest/v1/alumnos", `{"nombre":"María García","edad":20}`, `[{"id":3,"nombre":"María García","edad":20}]`},
		{"PATCH", "/rest/v1/alumnos?id=eq.3", `{"nombre":"María Torres","edad":21}`, `[{"id":3,"nombre":"María Torres","edad":21}]`},
		{"POST", "/rest/v1/matriculas", `{"alumno_id":3,"curso":"Desarrollo Web","modalidad":"Virtual"}`, `[{"id":4,"alumno_id":3,"curso":"Desarrollo Web","modalidad":"Virtual"}]`},
		{"PATCH", "/rest/v1/matriculas?id=eq.4", `{"alumno_id":3,"curso":"Base de Datos","modalidad":"Presencial"}`, `[{"id":4,"alumno_id":3,"curso":"Base de Datos","modalidad":"Presencial"}]`},
		{"DELETE", "/rest/v1/matriculas?id=eq.4", ``, `[{"id":4}]`},
		{"DELETE", "/rest/v1/alumnos?id=eq.3", ``, `[{"id":3}]`},
	}
	repositorio := prepararSupabase(t, func(w http.ResponseWriter, r *http.Request) {
		if paso >= len(casos) {
			t.Error("Se recibió una petición adicional")
			w.WriteHeader(500)
			return
		}
		caso := casos[paso]
		paso++
		if r.Method != caso.metodo || r.URL.RequestURI() != caso.ruta {
			t.Errorf("Petición inesperada: %s %s", r.Method, r.URL.RequestURI())
		}
		if r.Header.Get("apikey") != "sb_secret_prueba_solo_local" || r.Header.Get("Authorization") != "" {
			t.Error("Las claves nuevas deben viajar solo en apikey")
		}
		if r.Header.Get("Prefer") != "return=representation" {
			t.Error("Falta la representación de la fila")
		}
		if caso.cuerpo != "" {
			var recibido, esperado map[string]any
			if err := json.NewDecoder(r.Body).Decode(&recibido); err != nil {
				t.Error(err)
			}
			_ = json.Unmarshal([]byte(caso.cuerpo), &esperado)
			a, _ := json.Marshal(recibido)
			b, _ := json.Marshal(esperado)
			if string(a) != string(b) {
				t.Errorf("Cuerpo inesperado: %s", a)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(caso.respuesta))
	})
	if _, err := repositorio.Consultar(); err != nil {
		t.Fatal(err)
	}
	if alumno, err := repositorio.GuardarAlumno(dominio.Alumno{Nombre: "María García", Edad: 20}); err != nil || alumno.ID != 3 {
		t.Fatalf("Registrar: %+v %v", alumno, err)
	}
	if _, err := repositorio.GuardarAlumno(dominio.Alumno{ID: 3, Nombre: "María Torres", Edad: 21}); err != nil {
		t.Fatal(err)
	}
	if _, err := repositorio.GuardarMatricula(dominio.Matricula{AlumnoID: 3, Curso: "Desarrollo Web", Modalidad: "Virtual"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repositorio.GuardarMatricula(dominio.Matricula{ID: 4, AlumnoID: 3, Curso: "Base de Datos", Modalidad: "Presencial"}); err != nil {
		t.Fatal(err)
	}
	if err := repositorio.EliminarMatricula(4); err != nil {
		t.Fatal(err)
	}
	if err := repositorio.EliminarAlumno(3); err != nil {
		t.Fatal(err)
	}
	if paso != len(casos) {
		t.Fatalf("Solo se hicieron %d peticiones", paso)
	}
}

func TestSupabaseErroresYRespuestasVacias(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		estado   int
		cuerpo   string
		esperado error
	}{
		{"duplicado", 409, `{"code":"23505"}`, dominio.ErrDuplicado},
		{"alumno ausente", 409, `{"code":"23503"}`, dominio.ErrAlumnoNoExiste},
		{"no autorizado", 401, `{"message":"sb_secret_no_mostrar"}`, dominio.ErrAlmacenamiento},
		{"sin permisos", 403, `{}`, dominio.ErrAlmacenamiento},
		{"sin función SQL", 404, `{"code":"PGRST202"}`, dominio.ErrAlmacenamiento},
		{"JSON roto", 200, `{`, dominio.ErrAlmacenamiento},
		{"respuesta sin tablas", 200, `{}`, dominio.ErrAlmacenamiento},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			repositorio := prepararSupabase(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(caso.estado)
				_, _ = w.Write([]byte(caso.cuerpo))
			})
			_, err := repositorio.Consultar()
			if !errors.Is(err, caso.esperado) {
				t.Fatalf("Error inesperado: %v", err)
			}
			if strings.Contains(err.Error(), "sb_secret_no_mostrar") {
				t.Fatal("Se filtró información remota")
			}
		})
	}
	repositorio := prepararSupabase(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`[]`)) })
	if err := repositorio.EliminarAlumno(999); !errors.Is(err, dominio.ErrNoEncontrado) {
		t.Fatal(err)
	}
	if _, err := repositorio.GuardarAlumno(dominio.Alumno{ID: 999}); !errors.Is(err, dominio.ErrNoEncontrado) {
		t.Fatal(err)
	}
	if _, err := repositorio.GuardarMatricula(dominio.Matricula{ID: 999}); !errors.Is(err, dominio.ErrNoEncontrado) {
		t.Fatal(err)
	}
	if _, err := repositorio.GuardarAlumno(dominio.Alumno{}); !errors.Is(err, dominio.ErrAlmacenamiento) {
		t.Fatal(err)
	}
}

func TestSupabaseClavesYRedirecciones(t *testing.T) {
	for _, clave := range []string{"", "sb_publishable_publica", "sb_secret_REEMPLAZAR", "clave-cualquiera"} {
		if _, err := NuevoRepositorioSupabase("https://ejemplo.supabase.co", clave); err == nil {
			t.Fatal("Se aceptó una clave que no es privada")
		}
	}
	for _, direccion := range []string{"http://ejemplo.supabase.co", "https://usuario:clave@ejemplo.supabase.co", "https://ejemplo.supabase.co?clave=secreto", "https://ejemplo.supabase.co/rest/v1"} {
		if _, err := NuevoRepositorioSupabase(direccion, "sb_secret_prueba"); err == nil {
			t.Fatal("Se aceptó una URL inválida")
		}
	}
	legada := "eyJhbGciOiJIUzI1NiJ9." + base64.RawURLEncoding.EncodeToString([]byte(`{"role":"service_role"}`)) + ".firma"
	repositorio := prepararSupabase(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+legada {
			t.Error("Falta Authorization para la clave JWT antigua")
		}
		_, _ = w.Write([]byte(`{"alumnos":[],"matriculas":[]}`))
	})
	repositorio.clave = legada
	if _, err := repositorio.Consultar(); err != nil {
		t.Fatal(err)
	}
	anónima := "eyJhbGciOiJIUzI1NiJ9." + base64.RawURLEncoding.EncodeToString([]byte(`{"role":"anon"}`)) + ".firma"
	if clavePrivadaValida(anónima) {
		t.Fatal("Se aceptó el rol anon")
	}
	destinoVisitado := false
	destino := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { destinoVisitado = true }))
	defer destino.Close()
	repositorio = prepararSupabase(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destino.URL, 307) })
	if _, err := repositorio.Consultar(); err == nil || destinoVisitado {
		t.Fatal("Se siguió una redirección con credenciales")
	}
}

func TestCargarEntorno(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), ".env")
	for _, nombre := range []string{"ACADEMIA_PRUEBA_A", "ACADEMIA_PRUEBA_B", "ACADEMIA_PRUEBA_C"} {
		anterior, existia := os.LookupEnv(nombre)
		_ = os.Unsetenv(nombre)
		t.Cleanup(func() {
			if existia {
				_ = os.Setenv(nombre, anterior)
			} else {
				_ = os.Unsetenv(nombre)
			}
		})
	}
	_ = os.Setenv("ACADEMIA_PRUEBA_C", "del-entorno")
	contenido := "# comentario\nACADEMIA_PRUEBA_A=valor\nACADEMIA_PRUEBA_B='con espacios'\nACADEMIA_PRUEBA_C=del-archivo\n"
	if err := os.WriteFile(ruta, []byte(contenido), 0600); err != nil {
		t.Fatal(err)
	}
	if err := CargarEntorno(ruta); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("ACADEMIA_PRUEBA_A") != "valor" || os.Getenv("ACADEMIA_PRUEBA_B") != "con espacios" || os.Getenv("ACADEMIA_PRUEBA_C") != "del-entorno" {
		t.Fatal("Las variables no se cargaron como se esperaba")
	}
	if err := os.WriteFile(ruta, []byte("ACADEMIA_PRUEBA_A='sin cierre"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := CargarEntorno(ruta); err == nil {
		t.Fatal("Se aceptaron comillas sin cierre")
	}
	if err := CargarEntorno(filepath.Join(t.TempDir(), "no-existe")); err != nil {
		t.Fatal(err)
	}
}
