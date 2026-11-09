package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"maizap/internal/config"
	"maizap/internal/handlers"
	"maizap/internal/services"
	"maizap/internal/storage"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := config.LoadServerConfig(args)
	if err != nil {
		return err
	}

	if _, err := os.Stat(cfg.DataFile); err != nil {
		if _, seedErr := os.Stat(cfg.SeedFile); seedErr != nil {
			log.Printf("Aviso: no se encontró %s ni %s; el servidor inicia sin datos de prueba (ejecute el servidor desde la carpeta app-go)", cfg.DataFile, cfg.SeedFile)
		}
	}

	store, err := storage.OpenJSONStore(cfg.DataFile, cfg.SeedFile)
	if err != nil {
		return err
	}

	users := services.NewUserService(store, time.Now)
	mail := services.NewMailService(store, users, time.Now)
	auth := services.NewLocalAuthenticator(users, cfg.SessionTTL, time.Now)
	api := handlers.NewAPI(users, mail, auth)

	listener, err := net.Listen("tcp", cfg.Address())
	if err != nil {
		return fmt.Errorf("no se pudo abrir %s (¿el puerto está en uso o la IP no pertenece a este equipo?): %w", cfg.Address(), err)
	}

	server := &http.Server{
		Handler:           api.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("MaiZap servidor escuchando en http://%s", cfg.Address())
	log.Printf("Datos: %s | Sesiones expiran tras %s de inactividad", store.Path(), cfg.SessionTTL)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("el servidor se detuvo inesperadamente: %w", err)
		}
		return nil
	case <-stop:
		log.Printf("Apagando servidor...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("no se pudo detener el servidor correctamente: %w", err)
	}
	log.Printf("Servidor detenido")
	return nil
}
