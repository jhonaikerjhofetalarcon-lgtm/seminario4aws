package aplicacion

import "seminario4/dominio"

// CRUD 2 conserva sus cuatro operaciones. Visitor añade las indicaciones en las
// respuestas de crear, consultar y actualizar. Eliminar sigue siendo un borrado.
type GestorMatriculas struct {
	repositorio dominio.RepositorioAcademico
}

func NuevoGestorMatriculas(repositorio dominio.RepositorioAcademico) *GestorMatriculas {
	return &GestorMatriculas{repositorio: repositorio}
}

// Esto es un DTO: un objeto de respuesta. La indicación se calcula, no se guarda.
// Al incluir Matricula, su ID, alumno_id, curso y modalidad aparecen en el JSON.
type DetalleMatricula struct {
	dominio.Matricula
	Indicacion string `json:"indicacion"`
}

func (g *GestorMatriculas) Consultar() ([]DetalleMatricula, error) {
	datos, err := g.repositorio.Consultar()
	if err != nil {
		return nil, err
	}
	return g.Describir(datos.Matriculas), nil
}

func (g *GestorMatriculas) Guardar(id, alumnoID int, curso, modalidad string) (DetalleMatricula, error) {
	matricula, err := dominio.NuevaMatricula(alumnoID, curso, modalidad)
	if err != nil {
		return DetalleMatricula{}, err
	}
	matricula.ID = id
	guardada, err := g.repositorio.GuardarMatricula(matricula)
	if err != nil {
		return DetalleMatricula{}, err
	}
	return describirMatricula(guardada), nil
}

func (g *GestorMatriculas) Eliminar(id int) error {
	return g.repositorio.EliminarMatricula(id)
}

// También usamos esta función para el resumen inicial, leyendo un solo archivo.
func (g *GestorMatriculas) Describir(matriculas []dominio.Matricula) []DetalleMatricula {
	detalles := []DetalleMatricula{}
	for _, matricula := range matriculas {
		detalles = append(detalles, describirMatricula(matricula))
	}
	return detalles
}

func describirMatricula(matricula dominio.Matricula) DetalleMatricula {
	if matricula.Modalidad == "" {
		matricula.Modalidad = "Presencial"
	}
	// Aquí se usa Visitor de verdad:
	// 1. Elegimos el elemento (presencial o virtual).
	// 2. Creamos el visitante con la operación que queremos aplicar.
	// 3. Aceptar llama a VisitarPresencial o VisitarVirtual según el elemento.
	elemento := dominio.CrearElementoMatricula(matricula)
	visitante := VisitanteIndicaciones{}
	indicacion := elemento.Aceptar(visitante)
	return DetalleMatricula{Matricula: matricula, Indicacion: indicacion}
}
