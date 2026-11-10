// El dominio contiene los datos y las reglas del negocio. No conoce HTTP ni archivos.
package dominio

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// Una entidad tiene identidad propia. Dos alumnos pueden llamarse igual y tener otro ID.
type Alumno struct {
	ID     int    `json:"id"`
	Nombre string `json:"nombre"`
	Edad   int    `json:"edad"`
}

// AlumnoID es la referencia (clave foránea) que conecta los dos CRUD.
type Matricula struct {
	ID        int    `json:"id"`
	AlumnoID  int    `json:"alumno_id"`
	Curso     string `json:"curso"`
	Modalidad string `json:"modalidad"`
}

var ErrNoEncontrado = errors.New("El registro no existe.")
var ErrDuplicado = errors.New("El alumno ya está matriculado en ese curso.")
var ErrAlumnoNoExiste = errors.New("El alumno elegido no está registrado.")
var ErrAlmacenamiento = errors.New("No se pudieron leer o guardar los datos.")

// Esto es una función fábrica: construye un alumno y valida sus datos.
// No es el patrón Factory Method clásico (que usa herencia).
func NuevoAlumno(nombre string, edad int) (Alumno, error) {
	nombre = strings.Join(strings.Fields(nombre), " ")
	if utf8.RuneCountInString(nombre) < 3 || utf8.RuneCountInString(nombre) > 100 {
		return Alumno{}, errors.New("El nombre debe tener entre 3 y 100 caracteres.")
	}
	if edad < 1 || edad > 120 {
		return Alumno{}, errors.New("La edad debe ser un número entero entre 1 y 120.")
	}
	return Alumno{Nombre: nombre, Edad: edad}, nil
}

// Usamos una lista pequeña de cursos, sin agregar un tercer CRUD.
func CursosDisponibles() []string {
	return []string{"Programación I", "Desarrollo Web", "Base de Datos", "Diseño de Interfaces"}
}

func NuevaMatricula(alumnoID int, curso, modalidad string) (Matricula, error) {
	if alumnoID < 1 {
		return Matricula{}, errors.New("Debes elegir un alumno registrado.")
	}
	// Las matrículas de la primera versión no tenían modalidad. Usamos Presencial.
	if modalidad == "" {
		modalidad = "Presencial"
	}
	if modalidad != "Presencial" && modalidad != "Virtual" {
		return Matricula{}, errors.New("La modalidad debe ser Presencial o Virtual.")
	}
	for _, disponible := range CursosDisponibles() {
		if curso == disponible {
			return Matricula{AlumnoID: alumnoID, Curso: curso, Modalidad: modalidad}, nil
		}
	}
	return Matricula{}, errors.New("Selecciona uno de los cursos disponibles.")
}

// Esto es una interfaz Repository: dice QUÉ podemos hacer, sin decidir CÓMO guardar.
// Un solo repositorio mantiene juntas las operaciones relacionadas de este ejercicio.
type RepositorioAcademico interface {
	Consultar() (DatosAcademicos, error)
	GuardarAlumno(Alumno) (Alumno, error)
	EliminarAlumno(int) error
	GuardarMatricula(Matricula) (Matricula, error)
	EliminarMatricula(int) error
	Reiniciar() error
}

type DatosAcademicos struct {
	Alumnos    []Alumno    `json:"alumnos"`
	Matriculas []Matricula `json:"matriculas"`
	// Guardamos los contadores para no reutilizar IDs después de eliminar registros.
	SiguienteAlumno    int `json:"siguiente_alumno"`
	SiguienteMatricula int `json:"siguiente_matricula"`
}
