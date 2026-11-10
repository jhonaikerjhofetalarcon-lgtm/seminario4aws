package presentacion_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"seminario4/aplicacion"
	"seminario4/dominio"
	"seminario4/infraestructura"
	"seminario4/presentacion"
)

// Estas pruebas envían peticiones reales al manejador HTTP sin abrir un puerto.
// t.TempDir crea una carpeta temporal: nunca usamos los datos de la aplicación.
func preparar(t *testing.T) (http.Handler, string) {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "academia.json")
	repositorio, err := infraestructura.NuevoRepositorioArchivo(ruta)
	if err != nil {
		t.Fatal(err)
	}
	return presentacion.NuevoManejador(aplicacion.NuevoServicio(repositorio), "http://localhost:3000"), ruta
}

func solicitar(t *testing.T, manejador http.Handler, metodo, ruta, cuerpo string, esperado int) *httptest.ResponseRecorder {
	t.Helper()
	peticion := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
	peticion.Header.Set("Content-Type", "application/json")
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != esperado {
		t.Fatalf("%s %s: esperado %d, recibido %d: %s", metodo, ruta, esperado, respuesta.Code, respuesta.Body.String())
	}
	return respuesta
}

func consultarDatos(t *testing.T, manejador http.Handler) dominio.DatosAcademicos {
	t.Helper()
	respuesta := solicitar(t, manejador, "GET", "/api/resumen", "", 200)
	var datos dominio.DatosAcademicos
	if err := json.Unmarshal(respuesta.Body.Bytes(), &datos); err != nil {
		t.Fatal(err)
	}
	return datos
}

func TestFlujoCompletoCRUDYPersistencia(t *testing.T) {
	manejador, ruta := preparar(t)
	solicitar(t, manejador, "GET", "/api/alumnos", "", 200)
	solicitar(t, manejador, "GET", "/api/matriculas", "", 200)
	solicitar(t, manejador, "POST", "/api/alumnos", `{"nombre":"  María   García ","edad":20}`, 201)
	datos := consultarDatos(t, manejador)
	if len(datos.Alumnos) != 3 || datos.Alumnos[2].Nombre != "María García" || datos.Alumnos[2].ID != 3 {
		t.Fatalf("El alumno no se guardó con nombre normalizado e ID: %+v", datos.Alumnos)
	}
	solicitar(t, manejador, "POST", "/api/matriculas", `{"alumno_id":3,"curso":"Desarrollo Web"}`, 201)
	solicitar(t, manejador, "PUT", "/api/matriculas/3", `{"alumno_id":3,"curso":"Base de Datos"}`, 200)
	solicitar(t, manejador, "PUT", "/api/alumnos/3", `{"nombre":"María Torres","edad":21}`, 200)
	// Recreamos el repositorio para simular un reinicio del backend.
	reabierto, err := infraestructura.NuevoRepositorioArchivo(ruta)
	if err != nil {
		t.Fatal(err)
	}
	manejador = presentacion.NuevoManejador(aplicacion.NuevoServicio(reabierto), "http://localhost:3000")
	datos = consultarDatos(t, manejador)
	if datos.Alumnos[2].Nombre != "María Torres" || datos.Matriculas[2].Curso != "Base de Datos" || datos.Matriculas[2].AlumnoID != 3 {
		t.Fatalf("No persistieron los cambios relacionados: %+v", datos)
	}
	// Eliminar solo una matrícula conserva al alumno.
	solicitar(t, manejador, "DELETE", "/api/matriculas/3", "", 204)
	if len(consultarDatos(t, manejador).Alumnos) != 3 {
		t.Fatal("Eliminar una matrícula eliminó un alumno")
	}
	solicitar(t, manejador, "POST", "/api/matriculas", `{"alumno_id":3,"curso":"Desarrollo Web"}`, 201)
	solicitar(t, manejador, "POST", "/api/matriculas", `{"alumno_id":3,"curso":"Base de Datos"}`, 201)
	solicitar(t, manejador, "DELETE", "/api/alumnos/3", "", 204)
	datos = consultarDatos(t, manejador)
	if len(datos.Alumnos) != 2 || len(datos.Matriculas) != 2 {
		t.Fatalf("El borrado en cascada dejó registros: %+v", datos)
	}
	solicitar(t, manejador, "POST", "/api/alumnos", `{"nombre":"Pedro Díaz","edad":25}`, 201)
	if consultarDatos(t, manejador).Alumnos[2].ID != 4 {
		t.Fatal("Se reutilizó un ID eliminado")
	}
	solicitar(t, manejador, "POST", "/api/ejemplos", "", 204)
	datos = consultarDatos(t, manejador)
	if len(datos.Alumnos) != 2 || len(datos.Matriculas) != 2 || datos.Alumnos[0].Nombre != "Juan Pérez" {
		t.Fatal("No se restauraron los ejemplos iniciales")
	}
}

func TestValidacionesYReferencias(t *testing.T) {
	manejador, _ := preparar(t)
	casos := []struct {
		nombre, metodo, ruta, cuerpo string
		estado                       int
	}{
		{"nombre vacío", "POST", "/api/alumnos", `{"nombre":"   ","edad":20}`, 400},
		{"edad inválida", "POST", "/api/alumnos", `{"nombre":"Luis Díaz","edad":0}`, 400},
		{"edad decimal", "POST", "/api/alumnos", `{"nombre":"Luis Díaz","edad":20.5}`, 400},
		{"JSON incompleto", "POST", "/api/alumnos", `{"nombre":`, 400},
		{"campo desconocido", "POST", "/api/alumnos", `{"nombre":"Luis Díaz","edad":20,"otro":true}`, 400},
		{"JSON doble", "POST", "/api/alumnos", `{"nombre":"Luis Díaz","edad":20} {}`, 400},
		{"alumno inexistente", "POST", "/api/matriculas", `{"alumno_id":999,"curso":"Base de Datos"}`, 400},
		{"curso inexistente", "POST", "/api/matriculas", `{"alumno_id":1,"curso":"Curso falso"}`, 400},
		{"matrícula repetida", "POST", "/api/matriculas", `{"alumno_id":1,"curso":"Programación I"}`, 409},
		{"editar alumno ausente", "PUT", "/api/alumnos/999", `{"nombre":"Luis Díaz","edad":20}`, 404},
		{"editar matrícula ausente", "PUT", "/api/matriculas/999", `{"alumno_id":1,"curso":"Base de Datos"}`, 404},
		{"ID inválido", "DELETE", "/api/alumnos/abc", "", 400},
		{"borrar alumno ausente", "DELETE", "/api/alumnos/999", "", 404},
		{"borrar matrícula ausente", "DELETE", "/api/matriculas/999", "", 404},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			solicitar(t, manejador, caso.metodo, caso.ruta, caso.cuerpo, caso.estado)
		})
	}
	if datos := consultarDatos(t, manejador); len(datos.Alumnos) != 2 || len(datos.Matriculas) != 2 {
		t.Fatal("Una operación rechazada modificó los datos")
	}
	// Una matrícula puede guardarse sin cambios, pero no convertirse en otra repetida.
	solicitar(t, manejador, "PUT", "/api/matriculas/1", `{"alumno_id":1,"curso":"Programación I"}`, 200)
	solicitar(t, manejador, "PUT", "/api/matriculas/2", `{"alumno_id":1,"curso":"Programación I"}`, 409)
}

func TestMatriculasConcurrentesNoSeDuplican(t *testing.T) {
	manejador, _ := preparar(t)
	var grupo sync.WaitGroup
	estados := make(chan int, 8)
	for intento := 0; intento < 8; intento++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, httptest.NewRequest("POST", "/api/matriculas", strings.NewReader(`{"alumno_id":1,"curso":"Base de Datos"}`)))
			estados <- respuesta.Code
		}()
	}
	grupo.Wait()
	close(estados)
	creadas := 0
	for estado := range estados {
		if estado == 201 {
			creadas++
		} else if estado != 409 {
			t.Fatalf("Respuesta inesperada: %d", estado)
		}
	}
	if creadas != 1 || len(consultarDatos(t, manejador).Matriculas) != 3 {
		t.Fatal("Peticiones simultáneas generaron matrículas duplicadas")
	}
}

func TestArchivoDanadoNoSeReemplaza(t *testing.T) {
	manejador, ruta := preparar(t)
	if err := os.WriteFile(ruta, []byte("archivo dañado"), 0600); err != nil {
		t.Fatal(err)
	}
	solicitar(t, manejador, "GET", "/api/resumen", "", 500)
	if _, err := infraestructura.NuevoRepositorioArchivo(ruta); err == nil {
		t.Fatal("El repositorio aceptó un archivo dañado")
	}
	contenido, err := os.ReadFile(ruta)
	if err != nil || string(contenido) != "archivo dañado" {
		t.Fatal("Se reemplazó el archivo dañado en lugar de avisar")
	}
}

func TestCORS(t *testing.T) {
	manejador, _ := preparar(t)
	for _, origen := range []string{"http://localhost:3000", "http://otro-origen.test"} {
		peticion := httptest.NewRequest("OPTIONS", "/api/alumnos", nil)
		peticion.Header.Set("Origin", origen)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticion)
		if origen == "http://localhost:3000" && (respuesta.Code != 204 || respuesta.Header().Get("Access-Control-Allow-Origin") != origen) {
			t.Fatal("No se permitió el origen del frontend")
		}
		if origen != "http://localhost:3000" && (respuesta.Code != 403 || respuesta.Header().Get("Access-Control-Allow-Origin") != "") {
			t.Fatal("Se permitió un origen desconocido")
		}
	}
}

func TestSaludNoExponeDatos(t *testing.T) {
	// La salud del proceso no consulta alumnos ni necesita credenciales.
	manejador := presentacion.NuevoManejador(nil, "http://localhost:3000")
	respuesta := solicitar(t, manejador, "GET", "/healthz", "", 200)
	var salud map[string]string
	if err := json.Unmarshal(respuesta.Body.Bytes(), &salud); err != nil {
		t.Fatal(err)
	}
	if len(salud) != 1 || salud["estado"] != "ok" {
		t.Fatal("La ruta de salud expuso información inesperada")
	}
}

func TestEscrituraDesdeOtroOrigenNoModificaDatos(t *testing.T) {
	manejador, _ := preparar(t)
	peticion := httptest.NewRequest("POST", "/api/alumnos", strings.NewReader(`{"nombre":"Alumno externo","edad":20}`))
	peticion.Header.Set("Origin", "https://otro-sitio.test")
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusForbidden || len(consultarDatos(t, manejador).Alumnos) != 2 {
		t.Fatal("Una petición de otro sitio pudo modificar los datos")
	}
}

func TestVisitorEnLasRespuestasDelCRUD(t *testing.T) {
	manejador, ruta := preparar(t)
	solicitar(t, manejador, "POST", "/api/alumnos", `{"nombre":"María García","edad":20}`, 201)
	respuesta := solicitar(t, manejador, "POST", "/api/matriculas", `{"alumno_id":3,"curso":"Desarrollo Web","modalidad":"Presencial"}`, 201)
	var creada aplicacion.DetalleMatricula
	if err := json.Unmarshal(respuesta.Body.Bytes(), &creada); err != nil {
		t.Fatal(err)
	}
	if creada.Modalidad != "Presencial" || creada.Indicacion != "Asiste al instituto para el curso Desarrollo Web." {
		t.Fatalf("No se aplicó el visitante presencial: %+v", creada)
	}
	respuesta = solicitar(t, manejador, "PUT", "/api/matriculas/3", `{"alumno_id":3,"curso":"Desarrollo Web","modalidad":"Virtual"}`, 200)
	var actualizada aplicacion.DetalleMatricula
	if err := json.Unmarshal(respuesta.Body.Bytes(), &actualizada); err != nil {
		t.Fatal(err)
	}
	if actualizada.Modalidad != "Virtual" || actualizada.Indicacion != "Ingresa al aula virtual para el curso Desarrollo Web." {
		t.Fatalf("No se aplicó el visitante virtual: %+v", actualizada)
	}
	respuesta = solicitar(t, manejador, "GET", "/api/matriculas", "", 200)
	var lista []aplicacion.DetalleMatricula
	if err := json.Unmarshal(respuesta.Body.Bytes(), &lista); err != nil {
		t.Fatal(err)
	}
	if lista[2].Indicacion != actualizada.Indicacion {
		t.Fatal("Consultar no ejecutó Visitor")
	}
	// La modalidad sí se guarda, la indicación se genera cada vez que se consulta.
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(contenido), "indicacion") {
		t.Fatal("Se guardó un texto que debería generar el visitante")
	}
	solicitar(t, manejador, "POST", "/api/matriculas", `{"alumno_id":3,"curso":"Base de Datos","modalidad":"Desconocida"}`, 400)
	solicitar(t, manejador, "POST", "/api/matriculas", `{"alumno_id":3,"curso":"Desarrollo Web","modalidad":"Presencial"}`, 409)
	solicitar(t, manejador, "DELETE", "/api/matriculas/3", "", 204)
}

func TestMatriculasAnterioresSinModalidad(t *testing.T) {
	manejador, ruta := preparar(t)
	datos := consultarDatos(t, manejador)
	for indice := range datos.Matriculas {
		datos.Matriculas[indice].Modalidad = ""
	}
	// Conservamos los contadores al simular un archivo de la primera versión.
	datos.SiguienteAlumno = 3
	datos.SiguienteMatricula = 3
	contenido, err := json.Marshal(datos)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, contenido, 0600); err != nil {
		t.Fatal(err)
	}
	respuesta := solicitar(t, manejador, "GET", "/api/matriculas", "", 200)
	var lista []aplicacion.DetalleMatricula
	if err := json.Unmarshal(respuesta.Body.Bytes(), &lista); err != nil {
		t.Fatal(err)
	}
	for _, matricula := range lista {
		if matricula.Modalidad != "Presencial" || !strings.HasPrefix(matricula.Indicacion, "Asiste al instituto") {
			t.Fatalf("No se conservó compatibilidad con el archivo anterior: %+v", matricula)
		}
	}
}
