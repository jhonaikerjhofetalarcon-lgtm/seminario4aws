package aplicacion

import "seminario4/dominio"

// CRUD 2 — PATRÓN VISITOR: este es el visitante concreto.
// La operación "generar indicación" está separada de los tipos de matrícula.
// Para otra operación podemos crear otro visitante sin cambiar Aceptar.
type VisitanteIndicaciones struct{}

func (v VisitanteIndicaciones) VisitarPresencial(matricula dominio.MatriculaPresencial) string {
	return "Asiste al instituto para el curso " + matricula.Datos.Curso + "."
}

func (v VisitanteIndicaciones) VisitarVirtual(matricula dominio.MatriculaVirtual) string {
	return "Ingresa al aula virtual para el curso " + matricula.Datos.Curso + "."
}
