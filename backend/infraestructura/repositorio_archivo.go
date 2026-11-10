// Infraestructura implementa el contrato del dominio usando un archivo JSON.
package infraestructura

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"seminario4/dominio"
)

type RepositorioArchivo struct {
	ruta string
	// Mutex permite atender varias peticiones sin que escriban a la vez.
	// Este ejemplo se ejecuta en UN proceso de Go, no en varios servidores.
	mutex sync.Mutex
}

func NuevoRepositorioArchivo(ruta string) (*RepositorioArchivo, error) {
	repositorio := &RepositorioArchivo{ruta: ruta}
	if err := os.MkdirAll(filepath.Dir(ruta), 0755); err != nil {
		return nil, err
	}
	if _, err := os.Stat(ruta); os.IsNotExist(err) {
		if err := repositorio.escribir(datosDeEjemplo()); err != nil {
			return nil, err
		}
	}
	// Si el archivo existe pero está dañado, avisamos; nunca lo reemplazamos en silencio.
	if _, err := repositorio.leer(); err != nil {
		return nil, err
	}
	return repositorio, nil
}

func datosDeEjemplo() dominio.DatosAcademicos {
	return dominio.DatosAcademicos{
		Alumnos: []dominio.Alumno{
			{ID: 1, Nombre: "Juan Pérez", Edad: 20},
			{ID: 2, Nombre: "Ana López", Edad: 19},
		},
		Matriculas: []dominio.Matricula{
			{ID: 1, AlumnoID: 1, Curso: "Programación I", Modalidad: "Presencial"},
			{ID: 2, AlumnoID: 2, Curso: "Desarrollo Web", Modalidad: "Virtual"},
		},
		SiguienteAlumno: 3, SiguienteMatricula: 3,
	}
}

func (r *RepositorioArchivo) leer() (dominio.DatosAcademicos, error) {
	var datos dominio.DatosAcademicos
	contenido, err := os.ReadFile(r.ruta)
	if err == nil {
		err = json.Unmarshal(contenido, &datos)
	}
	if err != nil {
		return datos, fmt.Errorf("%w: %v", dominio.ErrAlmacenamiento, err)
	}
	return datos, nil
}

func (r *RepositorioArchivo) escribir(datos dominio.DatosAcademicos) error {
	// Primero escribimos en un archivo temporal. Si falla, conservamos el anterior.
	temporal, err := os.CreateTemp(filepath.Dir(r.ruta), "academia-*.tmp")
	if err != nil {
		return fmt.Errorf("%w: %v", dominio.ErrAlmacenamiento, err)
	}
	defer os.Remove(temporal.Name())
	codificador := json.NewEncoder(temporal)
	codificador.SetIndent("", "  ")
	err = codificador.Encode(datos)
	errCierre := temporal.Close()
	if err == nil {
		err = errCierre
	}
	if err == nil {
		err = os.Rename(temporal.Name(), r.ruta)
	}
	if err != nil {
		return fmt.Errorf("%w: %v", dominio.ErrAlmacenamiento, err)
	}
	return nil
}

// Este ayudante mantiene juntos leer + cambiar + guardar bajo el mismo bloqueo.
// cambio es una función que recibimos para indicar qué registro se modificará.
func (r *RepositorioArchivo) modificar(cambio func(*dominio.DatosAcademicos) error) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	datos, err := r.leer()
	if err != nil {
		return err
	}
	if err := cambio(&datos); err != nil {
		return err
	}
	return r.escribir(datos)
}

func (r *RepositorioArchivo) Consultar() (dominio.DatosAcademicos, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.leer()
}

func (r *RepositorioArchivo) GuardarAlumno(alumno dominio.Alumno) (dominio.Alumno, error) {
	err := r.modificar(func(datos *dominio.DatosAcademicos) error {
		if alumno.ID == 0 {
			alumno.ID = datos.SiguienteAlumno
			datos.SiguienteAlumno++
			datos.Alumnos = append(datos.Alumnos, alumno)
			return nil
		}
		for indice, actual := range datos.Alumnos {
			if actual.ID == alumno.ID {
				datos.Alumnos[indice] = alumno
				return nil
			}
		}
		return dominio.ErrNoEncontrado
	})
	return alumno, err
}

func (r *RepositorioArchivo) EliminarAlumno(id int) error {
	return r.modificar(func(datos *dominio.DatosAcademicos) error {
		indiceAlumno := -1
		for indice, alumno := range datos.Alumnos {
			if alumno.ID == id {
				indiceAlumno = indice
				break
			}
		}
		if indiceAlumno == -1 {
			return dominio.ErrNoEncontrado
		}
		datos.Alumnos = append(datos.Alumnos[:indiceAlumno], datos.Alumnos[indiceAlumno+1:]...)
		// Borrado en cascada: al eliminar un alumno quitamos también sus matrículas.
		matriculasRestantes := []dominio.Matricula{}
		for _, matricula := range datos.Matriculas {
			if matricula.AlumnoID != id {
				matriculasRestantes = append(matriculasRestantes, matricula)
			}
		}
		datos.Matriculas = matriculasRestantes
		return nil
	})
}

func (r *RepositorioArchivo) GuardarMatricula(matricula dominio.Matricula) (dominio.Matricula, error) {
	err := r.modificar(func(datos *dominio.DatosAcademicos) error {
		indiceMatricula := -1
		for indice, actual := range datos.Matriculas {
			if actual.ID == matricula.ID {
				indiceMatricula = indice
			}
		}
		if matricula.ID != 0 && indiceMatricula == -1 {
			return dominio.ErrNoEncontrado
		}
		alumnoExiste := false
		for _, alumno := range datos.Alumnos {
			if alumno.ID == matricula.AlumnoID {
				alumnoExiste = true
			}
		}
		if !alumnoExiste {
			return dominio.ErrAlumnoNoExiste
		}
		for _, actual := range datos.Matriculas {
			if actual.ID != matricula.ID && actual.AlumnoID == matricula.AlumnoID && actual.Curso == matricula.Curso {
				return dominio.ErrDuplicado
			}
		}
		if matricula.ID == 0 {
			matricula.ID = datos.SiguienteMatricula
			datos.SiguienteMatricula++
			datos.Matriculas = append(datos.Matriculas, matricula)
		} else {
			datos.Matriculas[indiceMatricula] = matricula
		}
		return nil
	})
	return matricula, err
}

func (r *RepositorioArchivo) EliminarMatricula(id int) error {
	return r.modificar(func(datos *dominio.DatosAcademicos) error {
		for indice, matricula := range datos.Matriculas {
			if matricula.ID == id {
				datos.Matriculas = append(datos.Matriculas[:indice], datos.Matriculas[indice+1:]...)
				return nil
			}
		}
		return dominio.ErrNoEncontrado
	})
}

func (r *RepositorioArchivo) Reiniciar() error {
	return r.modificar(func(datos *dominio.DatosAcademicos) error {
		*datos = datosDeEjemplo()
		return nil
	})
}
