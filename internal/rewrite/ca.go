package rewrite

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

type CA struct {
	Dir  string
	Cert *x509.Certificate
	Key  *rsa.PrivateKey
}

func (c *CA) CertPath() string { return filepath.Join(c.Dir, "ca.crt") }
func (c *CA) keyPath() string  { return filepath.Join(c.Dir, "ca.key") }

const caCommonName = "Spotify AdFree Local CA"

func LoadOrCreateCA(dir string) (*CA, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	ca := &CA{Dir: dir}
	if _, err := os.Stat(ca.CertPath()); err == nil {
		if err := ca.load(); err != nil {
			return nil, err
		}
		if ca.Cert.Subject.CommonName == caCommonName {
			return ca, nil
		}
		// 旧名称：重建后重新安装
		_ = os.Remove(ca.CertPath())
		_ = os.Remove(ca.keyPath())
	}
	if err := ca.create(); err != nil {
		return nil, err
	}
	return ca, nil
}

func (c *CA) load() error {
	certPEM, err := os.ReadFile(c.CertPath())
	if err != nil {
		return err
	}
	keyPEM, err := os.ReadFile(c.keyPath())
	if err != nil {
		return err
	}
	certBlock, _ := pem.Decode(certPEM)
	keyBlock, _ := pem.Decode(keyPEM)
	if certBlock == nil || keyBlock == nil {
		return fmt.Errorf("invalid CA pem")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return err
	}
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return err
	}
	c.Cert, c.Key = cert, key
	return nil
}

func (c *CA) create() error {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   caCommonName,
			Organization: []string{"Spotify AdFree"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return err
	}
	if err := os.WriteFile(c.CertPath(), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(c.keyPath(), pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0o600); err != nil {
		return err
	}
	c.Cert, c.Key = cert, key
	return nil
}

func (c *CA) Issue(host string) (certPEM, keyPEM []byte, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: host},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{host},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, c.Cert, &key.PublicKey, c.Key)
	if err != nil {
		return nil, nil, err
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return certPEM, keyPEM, nil
}
