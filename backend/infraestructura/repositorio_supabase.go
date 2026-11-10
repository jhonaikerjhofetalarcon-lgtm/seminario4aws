package infraestructura

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"seminario4/dominio"
)

// Repository: cumple el mismo contrato que RepositorioArchivo.
// Facade, Visitor y React no necesitan conocer cómo se comunica con Supabase.
type RepositorioSupabase struct {
	direccion string
	clave     string
	cliente   *http.Client
}

func NuevoRepositorioSupabase(direccion, clave string) (*RepositorioSupabase, error) {
	direccion = strings.TrimRight(strings.TrimSpace(direccion), "/")
	clave = strings.TrimSpace(clave)
	datosURL, err := url.Parse(direccion)
	if err != nil || datosURL.Host == "" || datosURL.User != nil || datosURL.RawQuery != "" || datosURL.Fragment != "" || datosURL.Path != "" {
		return nil, errors.New("Configura SUPABASE_URL con la URL raíz de tu proyecto.")
	}
	local := datosURL.Hostname() == "localhost" || datosURL.Hostname() == "127.0.0.1" || datosURL.Hostname() == "::1"
	if datosURL.Scheme != "https" && !(datosURL.Scheme == "http" && local) {
		return nil, errors.New("La URL de Supabase debe usar HTTPS.")
	}
	// Aceptamos la clave secreta actual o la antigua service_role.
	// Una clave pública no sirve para este repositorio privado del backend.
	if !clavePrivadaValida(clave) {
		return nil, errors.New("Configura SUPABASE_SECRET_KEY con una clave secreta de servidor, o SUPABASE_SERVICE_ROLE_KEY con service_role.")
	}
	return &RepositorioSupabase{
		direccion: direccion + "/rest/v1",
		clave:     clave,
		cliente: &http.Client{
			Timeout: 8 * time.Second,
			// No seguimos redirecciones para no enviar la clave a otro destino.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

func clavePrivadaValida(clave string) bool {
	if strings.HasPrefix(clave, "sb_secret_") && clave != "sb_secret_REEMPLAZAR" {
		return true
	}
	partes := strings.Split(clave, ".")
	if len(partes) != 3 {
		return false
	}
	contenido, err := base64.RawURLEncoding.DecodeString(partes[1])
	if err != nil {
		return false
	}
	var datos struct {
		Rol string `json:"role"`
	}
	return json.Unmarshal(contenido, &datos) == nil && datos.Rol == "service_role"
}

// net/http y encoding/json son parte de Go: no instalamos un SDK adicional.
func (r *RepositorioSupabase) solicitar(metodo, ruta string, datos, destino any) error {
	contenido, err := json.Marshal(datos)
	if err != nil {
		return fmt.Errorf("%w: no se pudo preparar la operación.", dominio.ErrAlmacenamiento)
	}
	var cuerpo io.Reader
	if datos != nil {
		cuerpo = bytes.NewReader(contenido)
	}
	peticion, err := http.NewRequest(metodo, r.direccion+ruta, cuerpo)
	if err != nil {
		return fmt.Errorf("%w: no se pudo preparar la petición.", dominio.ErrAlmacenamiento)
	}
	peticion.Header.Set("apikey", r.clave)
	// Las claves sb_secret_ no son JWT: solo se envían en apikey.
	if !strings.HasPrefix(r.clave, "sb_secret_") {
		peticion.Header.Set("Authorization", "Bearer "+r.clave)
	}
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	peticion.Header.Set("Prefer", "return=representation")
	respuesta, err := r.cliente.Do(peticion)
	if err != nil {
		return fmt.Errorf("%w: no se pudo conectar con Supabase.", dominio.ErrAlmacenamiento)
	}
	defer respuesta.Body.Close()
	lector := io.LimitReader(respuesta.Body, 8*1024*1024)
	if respuesta.StatusCode < 200 || respuesta.StatusCode >= 300 {
		var fallo struct {
			Codigo string `json:"code"`
		}
		// No mostramos cuerpos remotos ni cabeceras para evitar filtrar secretos.
		_ = json.NewDecoder(lector).Decode(&fallo)
		switch fallo.Codigo {
		case "23505":
			return dominio.ErrDuplicado
		case "23503":
			return dominio.ErrAlumnoNoExiste
		case "23514", "23502", "22P02":
			return errors.New("Los datos no cumplen las validaciones de la base de datos.")
		}
		if respuesta.StatusCode == 401 || respuesta.StatusCode == 403 {
			return fmt.Errorf("%w: revisa la clave y los permisos de Supabase.", dominio.ErrAlmacenamiento)
		}
		return fmt.Errorf("%w: Supabase respondió %d; revisa la conexión y el script de tablas.", dominio.ErrAlmacenamiento, respuesta.StatusCode)
	}
	if destino != nil {
		if err := json.NewDecoder(lector).Decode(destino); err != nil {
			return fmt.Errorf("%w: la respuesta de Supabase no tiene el formato esperado.", dominio.ErrAlmacenamiento)
		}
	}
	return nil
}

func (r *RepositorioSupabase) Consultar() (dominio.DatosAcademicos, error) {
	// Una función SQL obtiene ambas listas en la misma consulta y sin el límite
	// de filas de una consulta de tabla: devuelve un único objeto JSON.
	var datos dominio.DatosAcademicos
	err := r.solicitar(http.MethodPost, "/rpc/consultar_sistema_academico", struct{}{}, &datos)
	if err == nil && (datos.Alumnos == nil || datos.Matriculas == nil) {
		err = fmt.Errorf("%w: la consulta no devolvió las dos listas esperadas.", dominio.ErrAlmacenamiento)
	}
	if datos.Alumnos == nil {
		datos.Alumnos = []dominio.Alumno{}
	}
	if datos.Matriculas == nil {
		datos.Matriculas = []dominio.Matricula{}
	}
	return datos, err
}

func (r *RepositorioSupabase) GuardarAlumno(alumno dominio.Alumno) (dominio.Alumno, error) {
	metodo, ruta := http.MethodPost, "/alumnos"
	if alumno.ID != 0 {
		metodo, ruta = http.MethodPatch, "/alumnos?id=eq."+strconv.Itoa(alumno.ID)
	}
	datos := struct {
		Nombre string `json:"nombre"`
		Edad   int    `json:"edad"`
	}{alumno.Nombre, alumno.Edad}
	var guardados []dominio.Alumno
	err := r.solicitar(metodo, ruta, datos, &guardados)
	if err != nil {
		return dominio.Alumno{}, err
	}
	if len(guardados) == 0 && alumno.ID != 0 {
		return dominio.Alumno{}, dominio.ErrNoEncontrado
	}
	if len(guardados) != 1 || guardados[0].ID < 1 {
		return dominio.Alumno{}, fmt.Errorf("%w: Supabase no devolvió el alumno guardado.", dominio.ErrAlmacenamiento)
	}
	return guardados[0], nil
}

func (r *RepositorioSupabase) GuardarMatricula(matricula dominio.Matricula) (dominio.Matricula, error) {
	metodo, ruta := http.MethodPost, "/matriculas"
	if matricula.ID != 0 {
		metodo, ruta = http.MethodPatch, "/matriculas?id=eq."+strconv.Itoa(matricula.ID)
	}
	datos := struct {
		AlumnoID  int    `json:"alumno_id"`
		Curso     string `json:"curso"`
		Modalidad string `json:"modalidad"`
	}{matricula.AlumnoID, matricula.Curso, matricula.Modalidad}
	var guardadas []dominio.Matricula
	err := r.solicitar(metodo, ruta, datos, &guardadas)
	if err != nil {
		return dominio.Matricula{}, err
	}
	if len(guardadas) == 0 && matricula.ID != 0 {
		return dominio.Matricula{}, dominio.ErrNoEncontrado
	}
	if len(guardadas) != 1 || guardadas[0].ID < 1 {
		return dominio.Matricula{}, fmt.Errorf("%w: Supabase no devolvió la matrícula guardada.", dominio.ErrAlmacenamiento)
	}
	return guardadas[0], nil
}

func (r *RepositorioSupabase) eliminar(tabla string, id int) error {
	var eliminados []struct {
		ID int `json:"id"`
	}
	err := r.solicitar(http.MethodDelete, "/"+tabla+"?id=eq."+strconv.Itoa(id), nil, &eliminados)
	if err != nil {
		return err
	}
	if len(eliminados) == 0 {
		return dominio.ErrNoEncontrado
	}
	return nil
}

func (r *RepositorioSupabase) EliminarAlumno(id int) error {
	// PostgreSQL elimina sus matrículas mediante la clave foránea ON DELETE CASCADE.
	return r.eliminar("alumnos", id)
}

func (r *RepositorioSupabase) EliminarMatricula(id int) error {
	return r.eliminar("matriculas", id)
}

func (r *RepositorioSupabase) Reiniciar() error {
	return errors.New("El reinicio de ejemplos solo está disponible en el almacenamiento local.")
}
