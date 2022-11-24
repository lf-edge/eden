package openevec

import (
	"fmt"
	"os"

	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
)

func RegistryStart(cfg *RegistryConfig) error {
	command, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot obtain executable path: %w", err)
	}
	log.Infof("Executable path: %s", command)
	if err := eden.StartRegistry(cfg.Port, cfg.Tag, cfg.Dist); err != nil {
		return fmt.Errorf("cannot start registry: %w", err)
	}
	log.Infof("registry is running and accessible on port %d", cfg.Port)
	return nil
}

func RegistryLoad(ref string, cfg *RegistryConfig) error {
	registry := fmt.Sprintf("%s:%d", cfg.IP, cfg.Port)
	hash, err := utils.LoadRegistry(ref, registry)
	if err != nil {
		return fmt.Errorf("failed to load image %s: %w", ref, err)
	}
	fmt.Printf("image %s loaded with manifest hash %s\n", ref, hash)
	return nil
}
