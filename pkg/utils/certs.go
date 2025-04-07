package utils

import (
	"crypto/x509"
	"encoding/pem"
	"github.com/kairos-io/provider-canonical/pkg/fs"
	"github.com/pkg/errors"
)

func GetCertSans(certPath string) ([]string, error) {
	var sans []string

	certBytes, err := fs.OSFS.ReadFile(certPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read private key file")
	}

	// Decode the PEM caCertBlock
	certBlock, _ := pem.Decode(certBytes)
	if certBlock == nil || certBlock.Type != "CERTIFICATE" {
		return nil, errors.New("tls: failed to decode certificate")
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, errors.New("tls: failed to parse certificate: " + err.Error())
	}

	for _, dns := range cert.DNSNames {
		sans = append(sans, dns)
	}

	for _, ip := range cert.IPAddresses {
		sans = append(sans, ip.String())
	}
	return sans, nil
}
