package openevec

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
)

// AdamStart starts the OpenEVEC controller.
func (openEVEC *OpenEVEC) AdamStart() error {
	cfg := openEVEC.cfg
	command, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot obtain executable path: %w", err)
	}
	log.Infof("Executable path: %s", command)
	if !cfg.Adam.Remote.Redis {
		cfg.Adam.Redis.RemoteURL = ""
	}
	if err := eden.StartAdam(cfg.Adam.Port, cfg.Adam.Dist, cfg.Adam.Force, cfg.Adam.Tag,
		cfg.Adam.Redis.RemoteURL, cfg.Adam.APIv1, cfg.Eden.EnableIPv6, cfg.Eden.IPv6Subnet); err != nil {
		log.Errorf("cannot start adam: %s", err.Error())
	} else {
		log.Infof("Adam is running and accessible on port %d", cfg.Adam.Port)
	}
	return nil
}

// ChangeSigningCert uploads the provided signing certificate to the OpenEVEC
// controller and re-encrypts existing configs against the new cipher context.
//
// If newSignKey is non-nil it is treated as the private key matching
// newSignCert: re-encryption uses the new key for the new cipher context, and
// the new key is installed on disk as signing-key.pem after re-encryption
// completes. If newSignKey is nil the existing on-disk key is reused, which
// matches the historical "rotate cert metadata only" behavior.
func (openEVEC *OpenEVEC) ChangeSigningCert(newSignCert, newSignKey []byte) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDevFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("getControllerAndDevFromConfig: %w", err)
	}

	// we need to re-encrypt existing configs with the new certificate because EVE has support only for one server signing certificate
	err = reencryptConfigs(ctrl, dev, newSignCert, newSignKey)
	if err != nil {
		return fmt.Errorf("failed to reencrypt existing configs: %w", err)
	}

	// Push the re-encrypted configs to adam *before* rotating signing.pem /
	// signing-key.pem on disk. Adam's redis now holds cipher contexts that
	// reference the new cert hash, while adam still signs auth-containers
	// with the old key/cert. EVE's next config pull verifies the auth
	// container against its saved cert and then sees a cipher-context that
	// references an unknown cert hash, which is what triggers
	// SenderStatusCertMiss and the /certs fetch on the device side. The
	// on-disk swap below makes adam's subsequent responses use the new
	// signing material.
	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev: %w", err)
	}

	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		return err
	}
	globalCertsDir := filepath.Join(edenHome, defaults.DefaultCertsDist)
	signingCertPath := filepath.Join(globalCertsDir, "signing.pem")
	signingKeyPath := filepath.Join(globalCertsDir, "signing-key.pem")

	// Both writes go through atomicWriteFile so a crash mid-write cannot
	// leave a truncated signing.pem or signing-key.pem on disk; each file
	// is either fully old or fully new.
	//
	// The two files are still rotated independently, so there is a brief
	// window where adam may serve a (new key, old cert) pair: adam's
	// prepareEnvelope reads the cert and the key from disk in two separate
	// os.ReadFile calls (apiHandlerv2.go:364 and :376). A request that
	// races between our two renames below can therefore produce an
	// auth-container whose signature does not verify against the cert hash
	// it claims. EVE retries config requests on signature mismatch, so a
	// single dropped response is recoverable, but the residual race is
	// real and worth documenting.
	//
	// Writing the key before the cert means adam never serves (new cert,
	// old key) - the inverse skew - which would have the same effect; the
	// ordering choice is largely cosmetic given the residual race above.
	if len(newSignKey) > 0 {
		if err = atomicWriteFile(signingKeyPath, newSignKey, 0600); err != nil {
			return fmt.Errorf("cannot write signing key to %s: %w", signingKeyPath, err)
		}
	}
	if err = atomicWriteFile(signingCertPath, newSignCert, 0644); err != nil {
		// The key on disk is now the new key but the cert is the old one.
		// A subsequent ChangeSigningCert call will read this stale cert as
		// "old" while reading the new key as the "old" controller key,
		// which silently produces a wrong oldCryptoConfig. Surface the
		// inconsistency loudly so the operator knows manual recovery
		// (re-running the full rotation, or reverting signing-key.pem) is
		// required.
		if len(newSignKey) > 0 {
			log.Errorf("INCONSISTENT STATE: signing-key.pem was rotated but signing.pem failed to write to %s. Re-run change-signing-cert to recover.", signingCertPath)
		}
		return fmt.Errorf("cannot write signing cert to %s: %w", signingCertPath, err)
	}

	log.Infof("Signing cert changed successfully")
	return nil
}

// ChangeEncryptCert rotates the controller's ECDH (encryption) cert and key in
// adam, re-encrypting only those configs whose CipherContext references the
// old encrypt cert hash. The signing cert and key are left untouched, so
// auth-container envelopes continue to verify against the device's saved
// signing cert. Configs encrypted with the signing key (the historical
// default in this Eden setup) are left alone, so a rotation here is a no-op
// for apps that were not deployed with --use-encrypt-cert.
//
// If newEncKey is non-nil it is treated as the private key matching newEncCert
// and is installed alongside the cert. If newEncKey is nil the existing key is
// reused (cert-only rotation).
func (openEVEC *OpenEVEC) ChangeEncryptCert(newEncCert, newEncKey []byte) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDevFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("getControllerAndDevFromConfig: %w", err)
	}

	if err := reencryptConfigsForEncryptCert(ctrl, dev, newEncCert, newEncKey); err != nil {
		return fmt.Errorf("failed to reencrypt existing configs: %w", err)
	}

	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev: %w", err)
	}

	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		return err
	}
	globalCertsDir := filepath.Join(edenHome, defaults.DefaultCertsDist)
	encryptCertPath := filepath.Join(globalCertsDir, "encrypt.pem")
	encryptKeyPath := filepath.Join(globalCertsDir, "encrypt-key.pem")

	if len(newEncKey) > 0 {
		if err = atomicWriteFile(encryptKeyPath, newEncKey, 0600); err != nil {
			return fmt.Errorf("cannot write encrypt key to %s: %w", encryptKeyPath, err)
		}
	}
	if err = atomicWriteFile(encryptCertPath, newEncCert, 0644); err != nil {
		if len(newEncKey) > 0 {
			log.Errorf("INCONSISTENT STATE: encrypt-key.pem was rotated but encrypt.pem failed to write to %s. Re-run change-encrypt-cert to recover.", encryptCertPath)
		}
		return fmt.Errorf("cannot write encrypt cert to %s: %w", encryptCertPath, err)
	}

	log.Infof("Encrypt cert changed successfully")
	return nil
}

// reencryptConfigsForEncryptCert re-encrypts only those app/datastore/wireless
// cipher blocks whose owning CipherContext references the old encrypt cert -
// i.e. only configs that were encrypted via WithUseEncryptCert. The new cipher
// context is registered on the device and used as the destination for all
// matching cipher blocks.
func reencryptConfigsForEncryptCert(ctrl controller.Cloud, dev *device.Ctx, newEncCert, newEncKey []byte) error {
	devCert, err := ctrl.GetECDHCert(dev.GetID())
	if err != nil {
		return fmt.Errorf("cannot get device certificate from cloud: %w", err)
	}

	oldEncCert, err := ctrl.EncryptCertGet()
	if err != nil {
		log.Error("cannot get cloud's encrypt certificate. nothing to re-encrypt")
		return nil
	}

	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		return fmt.Errorf("DefaultEdenDir: %w", err)
	}
	keyPath := filepath.Join(edenHome, defaults.DefaultCertsDist, "encrypt-key.pem")
	oldCtrlPrivBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", keyPath, err)
	}

	newCtrlPrivBytes := oldCtrlPrivBytes
	if len(newEncKey) > 0 {
		newCtrlPrivBytes = newEncKey
	}

	oldCryptoConfig, err := utils.GetCommonCryptoConfig(devCert, oldEncCert, oldCtrlPrivBytes)
	if err != nil {
		return fmt.Errorf("GetCommonCryptoConfig (old): %w", err)
	}
	newCryptoConfig, err := utils.GetCommonCryptoConfig(devCert, newEncCert, newCtrlPrivBytes)
	if err != nil {
		return fmt.Errorf("GetCommonCryptoConfig (new): %w", err)
	}

	newCipherCtx, err := utils.CreateCipherCtx(newCryptoConfig)
	if err != nil {
		return fmt.Errorf("CreateCipherCtx: %w", err)
	}
	newCipherCtx = utils.AddCipherCtxToDev(dev, newCipherCtx)

	matchCtxIDs := map[string]struct{}{}
	for _, c := range dev.GetCipherContexts() {
		if controllerCertHashMatches(c.ControllerCertHash, oldEncCert) {
			matchCtxIDs[c.ContextId] = struct{}{}
		}
	}

	matches := func(holder utils.CipherDataHolder) bool {
		cd := holder.GetCipherData()
		if cd == nil {
			return false
		}
		_, ok := matchCtxIDs[cd.CipherContextId]
		return ok
	}

	for _, cfg := range ctrl.ListApplicationInstanceConfig() {
		if !matches(cfg) {
			continue
		}
		if err := utils.ReencryptConfigData(cfg, oldCryptoConfig, newCryptoConfig, newCipherCtx); err != nil {
			return fmt.Errorf("reencryptConfigData (app): %w", err)
		}
	}
	for _, cfg := range ctrl.ListDataStore() {
		if !matches(cfg) {
			continue
		}
		if err := utils.ReencryptConfigData(cfg, oldCryptoConfig, newCryptoConfig, newCipherCtx); err != nil {
			return fmt.Errorf("reencryptConfigData (datastore): %w", err)
		}
	}
	for _, networkConfigID := range dev.GetNetworks() {
		networkConfig, err := ctrl.GetNetworkConfig(networkConfigID)
		if err != nil {
			return fmt.Errorf("GetNetworkConfig: %w", err)
		}
		if networkConfig == nil || networkConfig.Wireless == nil {
			continue
		}
		for _, cfg := range networkConfig.Wireless.CellularCfg {
			for _, ap := range cfg.AccessPoints {
				if !matches(ap) {
					continue
				}
				if err := utils.ReencryptConfigData(ap, oldCryptoConfig, newCryptoConfig, newCipherCtx); err != nil {
					return fmt.Errorf("reencryptConfigData (cellular): %w", err)
				}
			}
		}
		for _, cfg := range networkConfig.Wireless.WifiCfg {
			if !matches(cfg) {
				continue
			}
			if err := utils.ReencryptConfigData(cfg, oldCryptoConfig, newCryptoConfig, newCipherCtx); err != nil {
				return fmt.Errorf("reencryptConfigData (wifi): %w", err)
			}
		}
	}

	return nil
}

// controllerCertHashMatches returns true iff the first 16 bytes of the
// CipherContext.ControllerCertHash equal the first 16 bytes of sha256 of the
// trimmed PEM-encoded cert (the same way Eden computed it at encrypt time -
// see utils.GetCommonCryptoConfig and utils.CreateCipherCtx).
func controllerCertHashMatches(ctxHash []byte, certPEM []byte) bool {
	if len(ctxHash) == 0 || len(certPEM) == 0 {
		return false
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(string(certPEM))))
	expect := sum[:16]
	if len(ctxHash) < len(expect) {
		return false
	}
	for i := range expect {
		if ctxHash[i] != expect[i] {
			return false
		}
	}
	return true
}

// atomicWriteFile writes data to path via a tmpfile in the same directory
// followed by an atomic os.Rename. The tmpfile is fsynced before rename so
// a crash post-rename cannot lose the new content. The destination's mode
// is set to perm regardless of umask.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create tmp in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	// On any error below, ensure the tmpfile is cleaned up.
	defer func() {
		// os.Remove on a non-existent file (after a successful rename)
		// returns ENOENT, which we ignore.
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write %s: %w", tmpName, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", tmpName, err)
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("chmod %s to %o: %w", tmpName, perm, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmpName, path, err)
	}
	return nil
}

func reencryptConfigs(ctrl controller.Cloud, dev *device.Ctx, newSignCert, newSignKey []byte) error {
	// get device certificate from the controller
	devCert, err := ctrl.GetECDHCert(dev.GetID())
	if err != nil {
		return fmt.Errorf("cannot get device certificate from cloud: %w", err)
	}

	// get signing certificate from the controller
	oldSignCert, err := ctrl.SigningCertGet()
	if err != nil {
		log.Error("cannot get cloud's signing certificate. will use plaintext")
		return nil
	}

	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		return fmt.Errorf("DefaultEdenDir: %w", err)
	}
	keyPath := filepath.Join(edenHome, defaults.DefaultCertsDist, "signing-key.pem")
	oldCtrlPrivBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", keyPath, err)
	}

	// The new ECDH shared secret must be derived from the *new* private key
	// when a key rotation is requested; otherwise the cipher context written
	// for the new cert would still be readable with the previous key.
	newCtrlPrivBytes := oldCtrlPrivBytes
	if len(newSignKey) > 0 {
		newCtrlPrivBytes = newSignKey
	}

	oldCryptoConfig, err := utils.GetCommonCryptoConfig(devCert, oldSignCert, oldCtrlPrivBytes)
	if err != nil {
		return fmt.Errorf("GetCommonCryptoConfig (old): %w", err)
	}

	newCryptoConfig, err := utils.GetCommonCryptoConfig(devCert, newSignCert, newCtrlPrivBytes)
	if err != nil {
		return fmt.Errorf("GetCommonCryptoConfig (new): %w", err)
	}

	cipherCtx, err := utils.CreateCipherCtx(newCryptoConfig)
	if err != nil {
		return fmt.Errorf("CreateCipherCtx: %w", err)
	}
	// add cipher context to device or return a matching existing one
	cipherCtx = utils.AddCipherCtxToDev(dev, cipherCtx)

	// Only re-encrypt configs whose CipherContext references the OLD signing
	// cert hash. Configs encrypted with the encrypt cert (--use-encrypt-cert)
	// are owned by change-encrypt-cert and are left untouched here, so the
	// two rotations can coexist without one breaking the other.
	matchCtxIDs := map[string]struct{}{}
	for _, c := range dev.GetCipherContexts() {
		if controllerCertHashMatches(c.ControllerCertHash, oldSignCert) {
			matchCtxIDs[c.ContextId] = struct{}{}
		}
	}
	matches := func(holder utils.CipherDataHolder) bool {
		cd := holder.GetCipherData()
		if cd == nil {
			return false
		}
		_, ok := matchCtxIDs[cd.CipherContextId]
		return ok
	}

	// re-encrypt all app configs with the new signing certificate
	appConfigs := ctrl.ListApplicationInstanceConfig()
	for _, config := range appConfigs {
		if !matches(config) {
			continue
		}
		if err = utils.ReencryptConfigData(config, oldCryptoConfig, newCryptoConfig, cipherCtx); err != nil {
			return fmt.Errorf("reencryptConfigData: %w", err)
		}
	}

	// re-encrypt all datastore configs with the new signing certificate
	dsConfigs := ctrl.ListDataStore()
	for _, config := range dsConfigs {
		if !matches(config) {
			continue
		}
		if err = utils.ReencryptConfigData(config, oldCryptoConfig, newCryptoConfig, cipherCtx); err != nil {
			return fmt.Errorf("reencryptConfigData: %w", err)
		}
	}

	// re-encrypt all wireless configs with the new signing certificate
	for _, networkConfigID := range dev.GetNetworks() {
		networkConfig, err := ctrl.GetNetworkConfig(networkConfigID)
		if err != nil {
			return fmt.Errorf("GetNetworkConfig: %w", err)
		}
		if networkConfig != nil && networkConfig.Wireless != nil {
			for _, config := range networkConfig.Wireless.CellularCfg {
				for _, ap := range config.AccessPoints {
					if !matches(ap) {
						continue
					}
					if err = utils.ReencryptConfigData(ap, oldCryptoConfig, newCryptoConfig, cipherCtx); err != nil {
						return fmt.Errorf("reencryptConfigData: %w", err)
					}
				}
			}
			for _, config := range networkConfig.Wireless.WifiCfg {
				if !matches(config) {
					continue
				}
				if err = utils.ReencryptConfigData(config, oldCryptoConfig, newCryptoConfig, cipherCtx); err != nil {
					return fmt.Errorf("reencryptConfigData: %w", err)
				}
			}
		}
	}

	return nil
}
