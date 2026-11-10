package dominio

// PATRÓN VISITOR — parte 1: los objetos que se pueden visitar.
// Una interfaz es un contrato. Cualquier tipo que tenga Aceptar cumple este contrato.
type ElementoMatricula interface {
	Aceptar(visitante VisitanteMatricula) string
}

// PATRÓN VISITOR — parte 2: el contrato del visitante.
// Exige un método por cada tipo concreto de matrícula.
type VisitanteMatricula interface {
	VisitarPresencial(matricula MatriculaPresencial) string
	VisitarVirtual(matricula MatriculaVirtual) string
}

// En Go usamos composición, no herencia: cada tipo contiene los datos de Matricula.
type MatriculaPresencial struct {
	Datos Matricula
}

type MatriculaVirtual struct {
	Datos Matricula
}

// Aceptar NO calcula la indicación. Le entrega el trabajo al visitante adecuado.
func (m MatriculaPresencial) Aceptar(visitante VisitanteMatricula) string {
	return visitante.VisitarPresencial(m)
}

func (m MatriculaVirtual) Aceptar(visitante VisitanteMatricula) string {
	return visitante.VisitarVirtual(m)
}

// Este ayudante elige el objeto concreto al leer el archivo JSON.
// La condición solo construye el tipo; la indicación vive en el visitante.
func CrearElementoMatricula(datos Matricula) ElementoMatricula {
	if datos.Modalidad == "Virtual" {
		return MatriculaVirtual{Datos: datos}
	}
	return MatriculaPresencial{Datos: datos}
}
