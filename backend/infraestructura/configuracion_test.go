package infraestructura

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaveSupabaseArchivoYVariables(t *testing.T) {
	t.Setenv("SUPABASE_SECRET_KEY_FILE", "")
	t.Setenv("SUPABASE_SECRET_KEY", "sb_secret_variable_de_prueba")
	t.Setenv("SUPABASE_SERVICE_ROLE_KEY", "legada_de_prueba")
	clave, err := ClaveSupabase()
	if err != nil || clave != "sb_secret_variable_de_prueba" {
		t.Fatal("No se respetó la variable privada principal")
	}
	t.Setenv("SUPABASE_SECRET_KEY", "")
	clave, err = ClaveSupabase()
	if err != nil || clave != "legada_de_prueba" {
		t.Fatal("No se respetó la variable de compatibilidad")
	}
	ruta := filepath.Join(t.TempDir(), "clave.txt")
	if err := os.WriteFile(ruta, []byte("sb_secret_archivo_de_prueba\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SUPABASE_SECRET_KEY_FILE", ruta)
	t.Setenv("SUPABASE_SECRET_KEY", "sb_secret_variable_de_prueba")
	clave, err = ClaveSupabase()
	if err != nil || clave != "sb_secret_archivo_de_prueba" {
		t.Fatal("El archivo explícito debe tener prioridad y admitir un salto final")
	}
	if err := os.WriteFile(ruta, []byte(" \n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ClaveSupabase(); err == nil {
		t.Fatal("Se aceptó un secreto vacío")
	}
	t.Setenv("SUPABASE_SECRET_KEY_FILE", ruta+"-no-existe")
	if _, err := ClaveSupabase(); err == nil || strings.Contains(err.Error(), ruta) {
		t.Fatal("El error debe rechazar el archivo ausente sin filtrar su ruta")
	}
}
