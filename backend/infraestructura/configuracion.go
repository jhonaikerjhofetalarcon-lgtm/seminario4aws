package infraestructura

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Go no lee .env automáticamente. Este lector pequeño admite NOMBRE=valor,
// comentarios en líneas separadas y valores entre comillas simples o dobles.
// No ejecuta comandos ni reemplaza variables: solo lee texto.
func CargarEntorno(ruta string) error {
	archivo, err := os.Open(ruta)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("No se pudo abrir la configuración .env.")
	}
	defer archivo.Close()
	lector := bufio.NewScanner(archivo)
	linea := 0
	for lector.Scan() {
		linea++
		texto := strings.TrimSpace(strings.TrimPrefix(lector.Text(), "\uFEFF"))
		if texto == "" || strings.HasPrefix(texto, "#") {
			continue
		}
		nombre, valor, encontrado := strings.Cut(texto, "=")
		nombre = strings.TrimSpace(nombre)
		if !encontrado || !nombreVariableValido(nombre) {
			return fmt.Errorf("Revisa el formato de .env en la línea %d.", linea)
		}
		valor = strings.TrimSpace(valor)
		if strings.HasPrefix(valor, "\"") || strings.HasPrefix(valor, "'") {
			if len(valor) < 2 || valor[len(valor)-1] != valor[0] {
				return fmt.Errorf("Revisa las comillas de .env en la línea %d.", linea)
			}
			valor = valor[1 : len(valor)-1]
		}
		// El entorno del proceso tiene prioridad sobre el archivo local.
		if _, existe := os.LookupEnv(nombre); !existe {
			if err := os.Setenv(nombre, valor); err != nil {
				return fmt.Errorf("No se pudo cargar una variable de .env.")
			}
		}
	}
	if lector.Err() != nil {
		return fmt.Errorf("No se pudo leer la configuración .env.")
	}
	return nil
}

func nombreVariableValido(nombre string) bool {
	if nombre == "" {
		return false
	}
	for indice, letra := range nombre {
		if letra == '_' || letra >= 'A' && letra <= 'Z' || letra >= 'a' && letra <= 'z' {
			continue
		}
		if indice > 0 && letra >= '0' && letra <= '9' {
			continue
		}
		return false
	}
	return true
}

// Docker monta el secreto en tiempo de ejecución; nunca se copia a la imagen.
func ClaveSupabase() (string, error) {
	if ruta := os.Getenv("SUPABASE_SECRET_KEY_FILE"); ruta != "" {
		contenido, err := os.ReadFile(ruta)
		if err != nil {
			return "", fmt.Errorf("No se pudo leer el archivo privado de Supabase.")
		}
		clave := strings.TrimSpace(string(contenido))
		if clave == "" {
			return "", fmt.Errorf("El archivo privado de Supabase está vacío.")
		}
		return clave, nil
	}
	if clave := os.Getenv("SUPABASE_SECRET_KEY"); clave != "" {
		return clave, nil
	}
	return os.Getenv("SUPABASE_SERVICE_ROLE_KEY"), nil
}
