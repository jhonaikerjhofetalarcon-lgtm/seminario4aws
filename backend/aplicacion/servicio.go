// Aplicación organiza los casos de uso: registrar, consultar, editar y eliminar.
package aplicacion

import "seminario4/dominio"

// Este servicio conecta las dos partes del ejercicio.
// Cada CRUD tiene su implementación en un archivo separado para poder estudiarlo.
type ServicioAcademico struct {
	repositorio dominio.RepositorioAcademico
	Alumnos     *FachadaAlumnos
	Matriculas  *GestorMatriculas
}

// Esto es inyección de dependencias manual: recibimos el repositorio desde main.go.
func NuevoServicio(repositorio dominio.RepositorioAcademico) *ServicioAcademico {
	return &ServicioAcademico{
		repositorio: repositorio,
		Alumnos:     NuevaFachadaAlumnos(repositorio),
		Matriculas:  NuevoGestorMatriculas(repositorio),
	}
}

func (s *ServicioAcademico) Consultar() (dominio.DatosAcademicos, error) {
	return s.Alumnos.ConsultarConMatriculas()
}

func (s *ServicioAcademico) Reiniciar() error {
	return s.repositorio.Reiniciar()
}
