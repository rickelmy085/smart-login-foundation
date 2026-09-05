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
	if filename == "" {
		filename = ".env"
	}

	abs, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("env file path: %w", err)
	}

	f, err := os.Open(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open env file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
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
		}
	}

	return scanner.Err()
}
