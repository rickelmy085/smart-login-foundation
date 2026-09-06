package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadEnvFile carrega variáveis de ambiente de um arquivo .env,
// apenas se ainda não estiverem definidas no ambiente do processo.
// Isso permite rodar `go run .` sem exportar manualmente as vars.
func LoadEnvFile(filename string) error {
	fmt.Printf("[CONFIG] LoadEnvFile iniciado filename=%s\n", filename)
	if filename == "" {
		filename = ".env"
	}

	abs, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("env file path: %w", err)
	}
	fmt.Printf("[CONFIG] LoadEnvFile abs path=%s\n", abs)

	f, err := os.Open(abs)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("[CONFIG] LoadEnvFile arquivo %s nao existe, ignorando\n", abs)
			return nil
		}
		return fmt.Errorf("open env file: %w", err)
	}
	defer f.Close()
	fmt.Printf("[CONFIG] LoadEnvFile arquivo aberto %s\n", abs)

	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Só define se não estiver já definida no ambiente.
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
			count++
		}
	}
	fmt.Printf("[CONFIG] LoadEnvFile finalizado vars carregadas=%d\n", count)
	return scanner.Err()
}
