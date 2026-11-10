package dominio_test

import (
	"seminario4/dominio"
	"testing"
)

// Este visitante de prueba demuestra otra operación con los mismos elementos.
// No modificamos Aceptar: ahora queremos etiquetas en lugar de indicaciones.
type VisitanteEtiquetas struct{}

func (v VisitanteEtiquetas) VisitarPresencial(m dominio.MatriculaPresencial) string {
	return "En campus: " + m.Datos.Curso
}

func (v VisitanteEtiquetas) VisitarVirtual(m dominio.MatriculaVirtual) string {
	return "En línea: " + m.Datos.Curso
}

func TestElementosAceptanOtroVisitante(t *testing.T) {
	var elementos = []dominio.ElementoMatricula{
		dominio.MatriculaPresencial{Datos: dominio.Matricula{Curso: "Desarrollo Web"}},
		dominio.MatriculaVirtual{Datos: dominio.Matricula{Curso: "Base de Datos"}},
	}
	esperados := []string{"En campus: Desarrollo Web", "En línea: Base de Datos"}
	visitante := VisitanteEtiquetas{}
	for indice, elemento := range elementos {
		if resultado := elemento.Aceptar(visitante); resultado != esperados[indice] {
			t.Fatalf("El elemento eligió otro método: %q", resultado)
		}
	}
}
