// Presentación traduce las peticiones del navegador a llamadas de la aplicación.
package presentacion

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"seminario4/aplicacion"
	"seminario4/dominio"
)

type Controlador struct {
	servicio *aplicacion.ServicioAcademico
}

func NuevoManejador(servicio *aplicacion.ServicioAcademico, origen string) http.Handler {
	c := &Controlador{servicio: servicio}
	rutas := http.NewServeMux()
	// Salud del proceso: no devuelve alumnos, credenciales ni datos de la base.
	rutas.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		responder(w, http.StatusOK, map[string]string{"estado": "ok"})
	})
	rutas.HandleFunc("GET /api/resumen", c.consultar)
	rutas.HandleFunc("GET /api/alumnos", c.consultarAlumnos)
	rutas.HandleFunc("GET /api/matriculas", c.consultarMatriculas)
	rutas.HandleFunc("POST /api/alumnos", c.guardarAlumno)
	rutas.HandleFunc("PUT /api/alumnos/{id}", c.guardarAlumno)
	rutas.HandleFunc("DELETE /api/alumnos/{id}", c.eliminarAlumno)
	rutas.HandleFunc("POST /api/matriculas", c.guardarMatricula)
	rutas.HandleFunc("PUT /api/matriculas/{id}", c.guardarMatricula)
	rutas.HandleFunc("DELETE /api/matriculas/{id}", c.eliminarMatricula)
	rutas.HandleFunc("POST /api/ejemplos", c.reiniciar)

	// CORS permite que Next.js (puerto 3000) hable con Go (puerto 8080).
	// CORS no reemplaza la autenticación; este proyecto es una práctica local.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "Origin")
		if r.Header.Get("Origin") == origen {
			w.Header().Set("Access-Control-Allow-Origin", origen)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			if r.Header.Get("Origin") != origen {
				responder(w, http.StatusForbidden, map[string]string{"error": "Origen no permitido."})
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// Al publicar con acceso compartido, bloqueamos escrituras de otro sitio.
		// Las herramientas sin Origin siguen funcionando detrás del proxy privado.
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Header.Get("Origin") != "" && r.Header.Get("Origin") != origen {
			responder(w, http.StatusForbidden, map[string]string{"error": "Origen no permitido."})
			return
		}
		rutas.ServeHTTP(w, r)
	})
}

func responder(w http.ResponseWriter, estado int, contenido any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(estado)
	if contenido != nil {
		if err := json.NewEncoder(w).Encode(contenido); err != nil {
			log.Print(err)
		}
	}
}

func responderError(w http.ResponseWriter, err error) {
	estado := http.StatusBadRequest
	if errors.Is(err, dominio.ErrNoEncontrado) {
		estado = http.StatusNotFound
	}
	if errors.Is(err, dominio.ErrDuplicado) {
		estado = http.StatusConflict
	}
	if errors.Is(err, dominio.ErrAlmacenamiento) {
		log.Print(err)
		responder(w, http.StatusInternalServerError, map[string]string{"error": dominio.ErrAlmacenamiento.Error()})
		return
	}
	responder(w, estado, map[string]string{"error": err.Error()})
}

func leerID(w http.ResponseWriter, r *http.Request) (int, bool) {
	if r.Method == http.MethodPost {
		return 0, true
	}
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		responder(w, http.StatusBadRequest, map[string]string{"error": "El ID debe ser un número entero positivo."})
		return 0, false
	}
	return id, true
}

func leerJSON(w http.ResponseWriter, r *http.Request, destino any) bool {
	// Limitamos el tamaño y rechazamos campos desconocidos para detectar errores.
	lector := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	lector.DisallowUnknownFields()
	if err := lector.Decode(destino); err != nil {
		responder(w, http.StatusBadRequest, map[string]string{"error": "Envía un JSON válido con los campos y tipos esperados."})
		return false
	}
	if err := lector.Decode(&struct{}{}); err != io.EOF {
		responder(w, http.StatusBadRequest, map[string]string{"error": "Envía un solo objeto JSON."})
		return false
	}
	return true
}

func (c *Controlador) consultar(w http.ResponseWriter, r *http.Request) {
	datos, err := c.servicio.Consultar()
	if err != nil {
		responderError(w, err)
		return
	}
	responder(w, http.StatusOK, map[string]any{
		"alumnos": datos.Alumnos, "matriculas": c.servicio.Matriculas.Describir(datos.Matriculas), "cursos": dominio.CursosDisponibles(),
	})
}

func (c *Controlador) consultarAlumnos(w http.ResponseWriter, r *http.Request) {
	alumnos, err := c.servicio.Alumnos.Consultar()
	if err != nil {
		responderError(w, err)
		return
	}
	responder(w, http.StatusOK, alumnos)
}

func (c *Controlador) consultarMatriculas(w http.ResponseWriter, r *http.Request) {
	matriculas, err := c.servicio.Matriculas.Consultar()
	if err != nil {
		responderError(w, err)
		return
	}
	responder(w, http.StatusOK, matriculas)
}

func (c *Controlador) guardarAlumno(w http.ResponseWriter, r *http.Request) {
	id, valido := leerID(w, r)
	if !valido {
		return
	}
	var entrada struct {
		Nombre string `json:"nombre"`
		Edad   int    `json:"edad"`
	}
	if !leerJSON(w, r, &entrada) {
		return
	}
	// CRUD 1 usa Facade: el HTTP solo llama a la acción pública que necesita.
	var alumno dominio.Alumno
	var err error
	if id == 0 {
		alumno, err = c.servicio.Alumnos.Registrar(entrada.Nombre, entrada.Edad)
	} else {
		alumno, err = c.servicio.Alumnos.Actualizar(id, entrada.Nombre, entrada.Edad)
	}
	if err != nil {
		responderError(w, err)
		return
	}
	estado := http.StatusOK
	if id == 0 {
		estado = http.StatusCreated
	}
	responder(w, estado, alumno)
}

func (c *Controlador) guardarMatricula(w http.ResponseWriter, r *http.Request) {
	id, valido := leerID(w, r)
	if !valido {
		return
	}
	var entrada struct {
		AlumnoID  int    `json:"alumno_id"`
		Curso     string `json:"curso"`
		Modalidad string `json:"modalidad"`
	}
	if !leerJSON(w, r, &entrada) {
		return
	}
	// CRUD 2 genera la indicación de la respuesta por medio del Visitor.
	matricula, err := c.servicio.Matriculas.Guardar(id, entrada.AlumnoID, entrada.Curso, entrada.Modalidad)
	if err != nil {
		responderError(w, err)
		return
	}
	estado := http.StatusOK
	if id == 0 {
		estado = http.StatusCreated
	}
	responder(w, estado, matricula)
}

func (c *Controlador) eliminarAlumno(w http.ResponseWriter, r *http.Request) {
	id, valido := leerID(w, r)
	if !valido {
		return
	}
	if err := c.servicio.Alumnos.Eliminar(id); err != nil {
		responderError(w, err)
		return
	}
	responder(w, http.StatusNoContent, nil)
}

func (c *Controlador) eliminarMatricula(w http.ResponseWriter, r *http.Request) {
	id, valido := leerID(w, r)
	if !valido {
		return
	}
	if err := c.servicio.Matriculas.Eliminar(id); err != nil {
		responderError(w, err)
		return
	}
	responder(w, http.StatusNoContent, nil)
}

func (c *Controlador) reiniciar(w http.ResponseWriter, r *http.Request) {
	if err := c.servicio.Reiniciar(); err != nil {
		responderError(w, err)
		return
	}
	responder(w, http.StatusNoContent, nil)
}
