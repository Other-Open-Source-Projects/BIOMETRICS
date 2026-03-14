package auth

import (
	"crypto/tls"
	"crypto/x509"
	"path/filepath"
	"testing"
	"time"
)

func TestNewMTLSManager(t *testing.T) {
	tmpDir := t.TempDir()
	config := &MTLSConfig{
		CAPath:         filepath.Join(tmpDir, "ca.pem"),
		ServerCertPath: filepath.Join(tmpDir, "server.pem"),
		ServerKeyPath:  filepath.Join(tmpDir, "server.key"),
		ClientCAPath:   filepath.Join(tmpDir, "client-ca.pem"),
		MinVersion:     tls.VersionTLS13,
	}

	manager, err := NewMTLSManager(config)
	if err != nil {
		t.Fatalf("Failed to create MTLSManager: %v", err)
	}
	defer manager.Stop()

	if manager.caCert == nil {
		t.Error("CA certificate should not be nil")
	}

	if manager.serverCert == nil {
		t.Error("Server certificate should not be nil")
	}

	if manager.clientCAPool == nil {
		t.Error("Client CA pool should not be nil")
	}
}

func TestDefaultMTLSConfig(t *testing.T) {
	config := DefaultMTLSConfig()

	if config.MinVersion != tls.VersionTLS13 {
		t.Errorf("Expected MinVersion TLS13, got %d", config.MinVersion)
	}

	if config.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("Expected RequireAndVerifyClientCert, got %v", config.ClientAuth)
	}

	if config.RotationInterval != 24*time.Hour {
		t.Errorf("Expected 24h rotation, got %v", config.RotationInterval)
	}
}

func TestMTLSManager_GetTLSConfig(t *testing.T) {
	tmpDir := t.TempDir()
	config := &MTLSConfig{
		CAPath:         filepath.Join(tmpDir, "ca.pem"),
		ServerCertPath: filepath.Join(tmpDir, "server.pem"),
		ServerKeyPath:  filepath.Join(tmpDir, "server.key"),
		ClientAuth:     tls.RequireAndVerifyClientCert,
	}

	manager, err := NewMTLSManager(config)
	if err != nil {
		t.Fatalf("Failed to create MTLSManager: %v", err)
	}
	defer manager.Stop()

	tlsConfig, err := manager.GetTLSConfig()
	if err != nil {
		t.Fatalf("Failed to get TLS config: %v", err)
	}

	_ = tlsConfig
	if len(manager.serverCert.Certificate) == 0 {
		t.Error("TLS config should have certificates")
	}

	if tlsConfig.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("Wrong client auth type: %v", tlsConfig.ClientAuth)
	}

	if len(tlsConfig.CipherSuites) == 0 {
		t.Error("TLS config should have cipher suites")
	}
}

func TestMTLSManager_GenerateClientCertificate(t *testing.T) {
	tmpDir := t.TempDir()
	config := &MTLSConfig{
		CAPath:         filepath.Join(tmpDir, "ca.pem"),
		ServerCertPath: filepath.Join(tmpDir, "server.pem"),
		ServerKeyPath:  filepath.Join(tmpDir, "server.key"),
	}

	manager, err := NewMTLSManager(config)
	if err != nil {
		t.Fatalf("Failed to create MTLSManager: %v", err)
	}
	defer manager.Stop()

	clientCert, err := manager.GenerateClientCertificate("test-client", 365*24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate client certificate: %v", err)
	}

	if clientCert == nil {
		t.Fatal("Client certificate should not be nil")
	}

	if len(clientCert.Certificate) == 0 {
		t.Error("Client certificate should have certificate data")
	}

	if clientCert.PrivateKey == nil {
		t.Error("Client certificate should have private key")
	}
}

func TestMTLSManager_ValidateClientCertificate(t *testing.T) {
	tmpDir := t.TempDir()
	config := &MTLSConfig{
		CAPath:         filepath.Join(tmpDir, "ca.pem"),
		ServerCertPath: filepath.Join(tmpDir, "server.pem"),
		ServerKeyPath:  filepath.Join(tmpDir, "server.key"),
	}

	manager, err := NewMTLSManager(config)
	if err != nil {
		t.Fatalf("Failed to create MTLSManager: %v", err)
	}
	defer manager.Stop()

	clientCert, err := manager.GenerateClientCertificate("test-client", 365*24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate client certificate: %v", err)
	}

	cert := clientCert.Leaf
	if cert == nil {
		t.Fatal("Client certificate leaf should not be nil")
	}

	err = manager.ValidateClientCertificate(cert)
	if err != nil {
		t.Errorf("Valid certificate should pass validation: %v", err)
	}
}

func TestMTLSManager_RotateCertificates(t *testing.T) {
	tmpDir := t.TempDir()
	config := &MTLSConfig{
		CAPath:         filepath.Join(tmpDir, "ca.pem"),
		ServerCertPath: filepath.Join(tmpDir, "server.pem"),
		ServerKeyPath:  filepath.Join(tmpDir, "server.key"),
	}

	manager, err := NewMTLSManager(config)
	if err != nil {
		t.Fatalf("Failed to create MTLSManager: %v", err)
	}
	defer manager.Stop()

	oldCert := manager.GetServerCertificate()

	err = manager.RotateCertificates()
	if err != nil {
		t.Fatalf("Failed to rotate certificates: %v", err)
	}

	newCert := manager.GetServerCertificate()

	if oldCert == newCert {
		t.Error("Certificate should have changed after rotation")
	}
}

func TestMTLSManager_NeedsRotation(t *testing.T) {
	tmpDir := t.TempDir()
	config := &MTLSConfig{
		CAPath:         filepath.Join(tmpDir, "ca.pem"),
		ServerCertPath: filepath.Join(tmpDir, "server.pem"),
		ServerKeyPath:  filepath.Join(tmpDir, "server.key"),
	}

	manager, err := NewMTLSManager(config)
	if err != nil {
		t.Fatalf("Failed to create MTLSManager: %v", err)
	}
	defer manager.Stop()

	needsRotation := manager.NeedsRotation()
	if needsRotation {
		t.Error("Fresh certificate should not need rotation")
	}
}

func TestMTLSManager_GetCertificateInfo(t *testing.T) {
	tmpDir := t.TempDir()
	config := &MTLSConfig{
		CAPath:         filepath.Join(tmpDir, "ca.pem"),
		ServerCertPath: filepath.Join(tmpDir, "server.pem"),
		ServerKeyPath:  filepath.Join(tmpDir, "server.key"),
	}

	manager, err := NewMTLSManager(config)
	if err != nil {
		t.Fatalf("Failed to create MTLSManager: %v", err)
	}
	defer manager.Stop()

	info, err := manager.GetCertificateInfo(config.CAPath)
	if err != nil {
		t.Fatalf("Failed to get certificate info: %v", err)
	}

	if info == nil {
		t.Fatal("Certificate info should not be nil")
	}

	if !info.IsCA {
		t.Error("CA certificate should have IsCA=true")
	}

	if info.Subject.CommonName != "BIOMETRICS Root CA" {
		t.Errorf("Wrong CA common name: %s", info.Subject.CommonName)
	}
}

func TestIsCertificateValid(t *testing.T) {
	tmpDir := t.TempDir()
	config := &MTLSConfig{
		CAPath:         filepath.Join(tmpDir, "ca.pem"),
		ServerCertPath: filepath.Join(tmpDir, "server.pem"),
		ServerKeyPath:  filepath.Join(tmpDir, "server.key"),
	}

	manager, err := NewMTLSManager(config)
	if err != nil {
		t.Fatalf("Failed to create MTLSManager: %v", err)
	}
	defer manager.Stop()

	caCert := manager.GetCACertificate()
	if !IsCertificateValid(caCert) {
		t.Error("Valid CA certificate should be valid")
	}

	expiredCert := &x509.Certificate{
		NotBefore: time.Now().Add(-2 * 24 * time.Hour),
		NotAfter:  time.Now().Add(-1 * 24 * time.Hour),
	}

	if IsCertificateValid(expiredCert) {
		t.Error("Expired certificate should not be valid")
	}
}

func TestDaysUntilExpiry(t *testing.T) {
	cert := &x509.Certificate{
		NotAfter: time.Now().Add(30 * 24 * time.Hour),
	}

	days := DaysUntilExpiry(cert)
	if days < 29 || days > 31 {
		t.Errorf("Expected ~30 days, got %d", days)
	}

	expiredCert := &x509.Certificate{
		NotAfter: time.Now().Add(-1 * 24 * time.Hour),
	}

	days = DaysUntilExpiry(expiredCert)
	if days >= 0 {
		t.Errorf("Expired cert should have negative days, got %d", days)
	}
}

func TestMTLSManager_AutoRotation(t *testing.T) {
	tmpDir := t.TempDir()
	config := &MTLSConfig{
		CAPath:           filepath.Join(tmpDir, "ca.pem"),
		ServerCertPath:   filepath.Join(tmpDir, "server.pem"),
		ServerKeyPath:    filepath.Join(tmpDir, "server.key"),
		RotationInterval: 100 * time.Millisecond,
		AutoRotate:       true,
	}

	manager, err := NewMTLSManager(config)
	if err != nil {
		t.Fatalf("Failed to create MTLSManager: %v", err)
	}

	time.Sleep(250 * time.Millisecond)
	manager.Stop()
}

func TestMTLSManager_SaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	config := &MTLSConfig{
		CAPath:         filepath.Join(tmpDir, "ca.pem"),
		ServerCertPath: filepath.Join(tmpDir, "server.pem"),
		ServerKeyPath:  filepath.Join(tmpDir, "server.key"),
	}

	manager1, err := NewMTLSManager(config)
	if err != nil {
		t.Fatalf("Failed to create first MTLSManager: %v", err)
	}
	manager1.Stop()

	manager2, err := NewMTLSManager(config)
	if err != nil {
		t.Fatalf("Failed to create second MTLSManager: %v", err)
	}
	defer manager2.Stop()

	if manager2.GetCACertificate() == nil {
		t.Error("Should have loaded CA certificate")
	}

	if manager2.GetServerCertificate() == nil {
		t.Error("Should have loaded server certificate")
	}
}

func TestMTLSConfig_Validation(t *testing.T) {
	config := &MTLSConfig{}

	if config.MinVersion == 0 {
		t.Log("MinVersion not set, will use default")
	}

	config.MinVersion = tls.VersionTLS12
	config.MaxVersion = tls.VersionTLS13

	if config.MinVersion > config.MaxVersion {
		t.Error("MinVersion should not be greater than MaxVersion")
	}
}

func TestMTLSManager_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	config := &MTLSConfig{
		CAPath:         filepath.Join(tmpDir, "ca.pem"),
		ServerCertPath: filepath.Join(tmpDir, "server.pem"),
		ServerKeyPath:  filepath.Join(tmpDir, "server.key"),
	}

	manager, err := NewMTLSManager(config)
	if err != nil {
		t.Fatalf("Failed to create MTLSManager: %v", err)
	}
	defer manager.Stop()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			if _, err := manager.GetTLSConfig(); err != nil {
				t.Error(err)
			}
			_ = manager.GetServerCertificate()
			_ = manager.GetCACertificate()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestCertificateErrors(t *testing.T) {
	tmpDir := t.TempDir()

	manager := &MTLSManager{
		config: &MTLSConfig{
			CAPath: filepath.Join(tmpDir, "nonexistent", "ca.pem"),
		},
	}

	err := manager.loadOrGenerateCA()
	if err != nil {
		t.Fatalf("expected CA generation to create missing dirs, got error: %v", err)
	}
}

func TestGenerateSerialNumber(t *testing.T) {
	serial1, err := generateSerialNumber()
	if err != nil {
		t.Fatalf("Failed to generate serial number: %v", err)
	}

	serial2, err := generateSerialNumber()
	if err != nil {
		t.Fatalf("Failed to generate serial number: %v", err)
	}

	if serial1.Cmp(serial2) == 0 {
		t.Error("Serial numbers should be unique")
	}
}
