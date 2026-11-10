package aplicacion

import "seminario4/dominio"

// CRUD 1 — PATRÓN FACADE (fachada).
// Es una puerta de entrada con cuatro acciones fáciles de usar.
// El controlador HTTP no necesita saber cómo validar ni cómo guardar los alumnos.
type FachadaAlumnos struct {
	repositorio dominio.RepositorioAcademico
}

// Recibimos el repositorio desde afuera: esto es inyección de dependencias.
func NuevaFachadaAlumnos(repositorio dominio.RepositorioAcademico) *FachadaAlumnos {
	return &FachadaAlumnos{repositorio: repositorio}
}

// C: crear. La fachada coordina dos partes: la validación del dominio y el guardado.
func (f *FachadaAlumnos) Registrar(nombre string, edad int) (dominio.Alumno, error) {
	return f.guardar(0, nombre, edad)
}

// R: consultar. El controlador recibe la lista y no conoce el archivo JSON.
func (f *FachadaAlumnos) Consultar() ([]dominio.Alumno, error) {
	datos, err := f.ConsultarConMatriculas()
	return datos.Alumnos, err
}

// La pantalla inicial necesita los alumnos junto con sus relaciones.
// Los leemos juntos para obtener una única fotografía del archivo.
func (f *FachadaAlumnos) ConsultarConMatriculas() (dominio.DatosAcademicos, error) {
	return f.repositorio.Consultar()
}

// U: actualizar. Conservamos el ID y validamos los nuevos datos.
func (f *FachadaAlumnos) Actualizar(id int, nombre string, edad int) (dominio.Alumno, error) {
	return f.guardar(id, nombre, edad)
}

// D: eliminar. El repositorio también borra las matrículas relacionadas.
func (f *FachadaAlumnos) Eliminar(id int) error {
	return f.repositorio.EliminarAlumno(id)
}

// Este método privado evita repetir la validación de crear y actualizar.
func (f *FachadaAlumnos) guardar(id int, nombre string, edad int) (dominio.Alumno, error) {
	alumno, err := dominio.NuevoAlumno(nombre, edad)
	if err != nil {
		return dominio.Alumno{}, err
	}
	alumno.ID = id
	return f.repositorio.GuardarAlumno(alumno)
}
