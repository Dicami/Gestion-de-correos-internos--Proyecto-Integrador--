package config

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultHost       = "127.0.0.1"
	DefaultPort       = 8080
	DefaultDataFile   = "data/maizap.json"
	DefaultSeedFile   = "data/seed.json"
	DefaultSessionTTL = 30 * time.Minute
	DefaultTimeout    = 5 * time.Second
)

type ServerConfig struct {
	Host       string
	Port       int
	DataFile   string
	SeedFile   string
	SessionTTL time.Duration
}

type ClientConfig struct {
	Host    string
	Port    int
	Timeout time.Duration
}

func LoadServerConfig(args []string) (ServerConfig, error) {
	cfg := ServerConfig{}
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	port, portErr := envInt("MAIZAP_PORT", DefaultPort)
	ttl, ttlErr := envDuration("MAIZAP_SESSION_TTL", DefaultSessionTTL)
	fs.StringVar(&cfg.Host, "host", envString("MAIZAP_HOST", DefaultHost), "dirección IP en la que escucha el servidor")
	fs.IntVar(&cfg.Port, "port", port, "puerto TCP del servidor")
	fs.StringVar(&cfg.DataFile, "data", envString("MAIZAP_DATA", DefaultDataFile), "archivo JSON donde se guardan los datos")
	fs.StringVar(&cfg.SeedFile, "seed", envString("MAIZAP_SEED", DefaultSeedFile), "archivo JSON con datos iniciales de prueba")
	fs.DurationVar(&cfg.SessionTTL, "session-ttl", ttl, "tiempo de inactividad antes de expirar una sesión")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if portErr != nil && !flagSet(fs, "port") {
		return cfg, portErr
	}
	if ttlErr != nil && !flagSet(fs, "session-ttl") {
		return cfg, ttlErr
	}
	if err := validateEndpoint(cfg.Host, cfg.Port); err != nil {
		return cfg, err
	}
	if strings.TrimSpace(cfg.DataFile) == "" {
		return cfg, fmt.Errorf("el archivo de datos no puede estar vacío")
	}
	if cfg.SessionTTL < time.Minute {
		return cfg, fmt.Errorf("la duración de sesión debe ser de al menos 1m (recibido %s)", cfg.SessionTTL)
	}
	return cfg, nil
}

func LoadClientConfig(args []string) (ClientConfig, error) {
	cfg := ClientConfig{}
	fs := flag.NewFlagSet("client", flag.ContinueOnError)
	port, portErr := envInt("MAIZAP_SERVER_PORT", DefaultPort)
	timeout, timeoutErr := envDuration("MAIZAP_TIMEOUT", DefaultTimeout)
	fs.StringVar(&cfg.Host, "host", envString("MAIZAP_SERVER_HOST", DefaultHost), "dirección IP del servidor MaiZap")
	fs.IntVar(&cfg.Port, "port", port, "puerto TCP del servidor MaiZap")
	fs.DurationVar(&cfg.Timeout, "timeout", timeout, "tiempo máximo de espera por solicitud")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if portErr != nil && !flagSet(fs, "port") {
		return cfg, portErr
	}
	if timeoutErr != nil && !flagSet(fs, "timeout") {
		return cfg, timeoutErr
	}
	if err := validateEndpoint(cfg.Host, cfg.Port); err != nil {
		return cfg, err
	}
	if cfg.Timeout <= 0 {
		return cfg, fmt.Errorf("el tiempo de espera debe ser positivo")
	}
	return cfg, nil
}

func (c ServerConfig) Address() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

func (c ClientConfig) Address() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

func (c ClientConfig) BaseURL() string {
	return "http://" + c.Address()
}

func validateEndpoint(host string, port int) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("la dirección IP no puede estar vacía")
	}
	if host != "localhost" && net.ParseIP(host) == nil {
		return fmt.Errorf("la dirección %q no es una IP válida", host)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("el puerto %d está fuera del rango 1-65535", port)
	}
	return nil
}

func flagSet(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func envString(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback, fmt.Errorf("la variable %s debe ser un número entero (recibido %q)", key, v)
	}
	return n, nil
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback, fmt.Errorf("la variable %s debe ser una duración como 30m o 1h (recibido %q)", key, v)
	}
	return d, nil
}
