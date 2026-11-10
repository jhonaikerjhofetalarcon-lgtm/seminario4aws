package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"seminario4/aplicacion"
	"seminario4/dominio"
	"seminario4/infraestructura"
	"seminario4/presentacion"
)

// main conecta las capas. Es el punto de entrada del backend.
func main() {
	if err := infraestructura.CargarEntorno(".env"); err != nil {
		log.Fatal(err)
	}
	ruta := os.Getenv("ARCHIVO_DATOS")
	if ruta == "" {
		ruta = "datos/academia.json"
	}
	origen := os.Getenv("ORIGEN_FRONTEND")
	if origen == "" {
		origen = "http://localhost:3000"
	}
	var repositorio dominio.RepositorioAcademico
	almacenamiento := strings.ToLower(strings.TrimSpace(os.Getenv("ALMACENAMIENTO")))
	clave, err := infraestructura.ClaveSupabase()
	if err != nil {
		log.Fatal(err)
	}
	if almacenamiento == "" {
		almacenamiento = "archivo"
		if os.Getenv("SUPABASE_URL") != "" || clave != "" {
			almacenamiento = "supabase"
		}
	}
	switch almacenamiento {
	case "supabase":
		repositorio, err = infraestructura.NuevoRepositorioSupabase(os.Getenv("SUPABASE_URL"), clave)
		if err == nil {
			// Verificamos acceso y tablas antes de anunciar que el backend está listo.
			_, err = repositorio.Consultar()
		}
	case "archivo":
		repositorio, err = infraestructura.NuevoRepositorioArchivo(ruta)
	default:
		log.Fatal("ALMACENAMIENTO debe ser archivo o supabase.")
	}
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Almacenamiento activo: %s", almacenamiento)
	servicio := aplicacion.NuevoServicio(repositorio)
	auth, err := infraestructura.NuevaAutenticacionSupabase(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_PUBLISHABLE_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	manejador, err := presentacion.ConAutenticacion(
		presentacion.NuevoManejador(servicio, origen), auth,
		os.Getenv("AUTH_USUARIOS_PERMITIDOS"), origen, strings.HasPrefix(origen, "https://"),
	)
	if err != nil {
		log.Fatal(err)
	}
	direccion := os.Getenv("DIRECCION_HTTP")
	if direccion == "" {
		direccion = "127.0.0.1:8080"
	}
	servidor := &http.Server{
		Addr:              direccion,
		Handler:           manejador,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
	ctx, cancelar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelar()
	errores := make(chan error, 1)
	go func() { errores <- servidor.ListenAndServe() }()
	log.Printf("Backend académico escuchando en %s", direccion)
	select {
	case err := <-errores:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	case <-ctx.Done():
		apagado, detener := context.WithTimeout(context.Background(), 15*time.Second)
		defer detener()
		if err := servidor.Shutdown(apagado); err != nil {
			log.Print("No se pudo completar el apagado ordenado del servidor.")
			_ = servidor.Close()
		}
	}
}
