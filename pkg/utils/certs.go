package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/kairos-io/provider-canonical/pkg/fs"
	"github.com/pkg/errors"
)

func GetExistingIpAndDnsSans(certPath string) ([]string, []net.IP, error) {
	certBytes, err := fs.OSFS.ReadFile(certPath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read private key file")
	}

	// Decode the PEM caCertBlock
	certBlock, _ := pem.Decode(certBytes)
	if certBlock == nil || certBlock.Type != "CERTIFICATE" {
		return nil, nil, errors.New("tls: failed to decode certificate")
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, errors.New("tls: failed to parse certificate: " + err.Error())
	}
	return cert.DNSNames, cert.IPAddresses, nil
}

func GetAllSans(certPath string) ([]string, error) {
	var sans []string

	dns, ip, err := GetExistingIpAndDnsSans(certPath)
	if err != nil {
		return nil, err
	}

	sans = append(sans, dns...)

	for _, ip := range ip {
		sans = append(sans, ip.String())
	}
	return sans, nil
}

func SplitIPAndDNSSANs(extraSANs []string) ([]string, []net.IP) {
	var ipSANs []net.IP
	var dnsSANs []string

	for _, san := range extraSANs {
		if san == "" {
			continue
		}

		if ip := net.ParseIP(san); ip != nil {
			ipSANs = append(ipSANs, ip)
		} else {
			dnsSANs = append(dnsSANs, san)
		}
	}

	return dnsSANs, ipSANs
}

// GenerateSerialNumber returns a random number that can be used for the SerialNumber field in an x509 certificate.
func GenerateSerialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	return serialNumber, nil
}

func GenerateCertificate(subject pkix.Name, notBefore, notAfter time.Time, ca bool, dnsSANs []string, ipSANs []net.IP) (*x509.Certificate, error) {
	serialNumber, err := GenerateSerialNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number for certificate template: %w", err)
	}

	cert := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               subject,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IPAddresses:           ipSANs,
		DNSNames:              dnsSANs,
		BasicConstraintsValid: true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	if ca {
		cert.IsCA = true
		cert.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign
	} else {
		cert.IsCA = false
		cert.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment | x509.KeyUsageDigitalSignature
	}

	return cert, nil
}

func SignCertificate(certificate *x509.Certificate, bits int, parent *x509.Certificate, pub any, priv any) (string, string, error) {
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate RSA private key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if keyPEM == nil {
		return "", "", fmt.Errorf("failed to encode private key PEM")
	}

	// Determine which public key to use in the certificate and which private key to use for signing
	var certPubKey any
	var signPrivKey any

	if pub == nil && priv == nil {
		// Self-signed certificate: use the newly generated key for both
		certPubKey = &key.PublicKey
		signPrivKey = key
	} else {
		// CA-signed certificate: use the new key's public key in the certificate,
		// but sign with the CA's private key
		certPubKey = &key.PublicKey
		signPrivKey = priv
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, certificate, parent, certPubKey, signPrivKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign certificate: %w", err)
	}
	crtPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if crtPEM == nil {
		return "", "", fmt.Errorf("failed to encode certificate PEM")
	}

	// Validate that the certificate and private key match before returning
	if err := ValidateCertificateKeyPair(string(crtPEM), string(keyPEM)); err != nil {
		return "", "", fmt.Errorf("certificate and private key mismatch detected: %w", err)
	}

	return string(crtPEM), string(keyPEM), nil
}

func LoadCertificate(certPEM string, keyPEM string) (*x509.Certificate, *rsa.PrivateKey, error) {
	decodedCert, _ := pem.Decode([]byte(certPEM))
	if decodedCert == nil {
		return nil, nil, fmt.Errorf("failed to parse certificate PEM")
	}
	cert, err := x509.ParseCertificate(decodedCert.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	if keyPEM == "" {
		return cert, nil, nil
	}

	key, err := loadRSAPrivateKey(keyPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load RSA private key: %w", err)
	}

	return cert, key, nil
}

func loadRSAPrivateKey(keyPEM string) (*rsa.PrivateKey, error) {
	pb, _ := pem.Decode([]byte(keyPEM))
	if pb == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}
	switch pb.Type {
	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(pb.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
		}
		return key, nil
	case "PRIVATE KEY":
		parsed, err := x509.ParsePKCS8PrivateKey(pb.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		v, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key")
		}
		return v, nil
	}
	return nil, fmt.Errorf("unknown private key block type %q", pb.Type)
}

// ValidateCertificateKeyPair verifies that a certificate and private key match.
// This is a critical validation to prevent certificate/key mismatches that would
// cause TLS handshake failures.
func ValidateCertificateKeyPair(certPEM, keyPEM string) error {
	cert, key, err := LoadCertificate(certPEM, keyPEM)
	if err != nil {
		return fmt.Errorf("failed to load certificate and key for validation: %w", err)
	}

	if cert == nil || key == nil {
		return fmt.Errorf("certificate or private key is nil")
	}

	// Extract the public key from the certificate
	certPubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("certificate public key is not RSA")
	}

	// Compare the modulus and exponent of the certificate's public key with the private key's public key
	if certPubKey.N.Cmp(key.N) != 0 {
		return fmt.Errorf("certificate public key modulus does not match private key modulus")
	}

	if certPubKey.E != key.E {
		return fmt.Errorf("certificate public key exponent does not match private key exponent")
	}

	return nil
}
