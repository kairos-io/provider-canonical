package stages

import (
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"net"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/kairos-io/provider-canonical/pkg/domain"
	"github.com/kairos-io/provider-canonical/pkg/fs"
	. "github.com/onsi/gomega"
	"github.com/twpayne/go-vfs/v4/vfst"
)

var testApiserverCrt = `
-----BEGIN CERTIFICATE-----
MIID5zCCAs+gAwIBAgIQG3xDWnjBcebLEpyuLS+1sTANBgkqhkiG9w0BAQsFADAY
MRYwFAYDVQQDEw1rdWJlcm5ldGVzLWNhMB4XDTI1MDUzMDAzNDkzMFoXDTQ1MDUz
MDAzNDkzMFowGTEXMBUGA1UEAxMOa3ViZS1hcGlzZXJ2ZXIwggEiMA0GCSqGSIb3
DQEBAQUAA4IBDwAwggEKAoIBAQC9N1pl2mBJ4pPXoR+OoM/2/mNMJ47SQiZJfJx2
s6IrzDWHDBWaQqOPbNXdAzhT9u9vucPFNz78xeRO3nMuO1XCQGG00QUz+i2rgPa0
kcJgsqDcDDr2i2TWqcDdiKFc1riuA0DXQfCs02CqIq0RW3AKnzNSzGu8VoBtgFQA
g06rlduoOorphPy3hNju7tKQFwrfveK+altVbs0hgJfUIuj3z0Hfpkn6Kvy7M7RJ
PbAHOf+jLB44I4sR7qmoM+HnrXgF8LeomSDDCl6SyMIqI/GfAtuK+/cDiRIWYuBZ
naKGx7zTZNtRDYYiO0TnCspu3cblxMCSACZCDZTEN0bK3gMDAgMBAAGjggEqMIIB
JjAOBgNVHQ8BAf8EBAMCBLAwHQYDVR0lBBYwFAYIKwYBBQUHAwIGCCsGAQUFBwMB
MAwGA1UdEwEB/wQCMAAwHwYDVR0jBBgwFoAUoP2isndWNQkDn5qo9doDaocWEtUw
gcUGA1UdEQSBvTCBuoIKa3ViZXJuZXRlc4ISa3ViZXJuZXRlcy5kZWZhdWx0ghZr
dWJlcm5ldGVzLmRlZmF1bHQuc3Zjgh5rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNs
dXN0ZXKCJGt1YmVybmV0ZXMuZGVmYXVsdC5zdmMuY2x1c3Rlci5sb2NhbIcECgqK
f4cECpi3AYcEfwAAAYcECgqKf4cQAAAAAAAAAAAAAAAAAAAAAYcQ/oAAAAAAAAAC
UFb//rg2xDANBgkqhkiG9w0BAQsFAAOCAQEAikR+DOCg5GfetUxqi96XC7qvugLh
GftiwVZO/7hbs49Rzu169ocny6Geyrt4nuiK3Vv1P3Zo26s/EbVtkjhi0ErgvIUo
N/D87LqOQZuX7QyLY+wpgULURaQbZjv0EPML1CvvoWjwRnWKNksjUL8Q3O4B6oBh
QyuUfvnS93IDWV8FKABWOBe8pdE8Y+m8N+nf6Mx8uFU2v6gABK6WHlnLxvEOVDi6
ku4QJ0MqKI8seSdsnjJaTsIRq3urax6wx81JFJA+y/Z+CJrmIvXOZn4wVKa2FVK4
TyBzIUKr20g3PaNWgmREj8Kng2+KLXOuU9BLNqJ6/9HAAwCyjMdgjYQ8Yw==
-----END CERTIFICATE-----
`

var testCACrt = `
-----BEGIN CERTIFICATE-----
MIIDHDCCAgSgAwIBAgIRAPl1nsnrDBZPVKLsB2iWsYUwDQYJKoZIhvcNAQELBQAw
GDEWMBQGA1UEAxMNa3ViZXJuZXRlcy1jYTAeFw0yNTA1MzAwMzQ5MzBaFw00NTA1
MzAwMzQ5MzBaMBgxFjAUBgNVBAMTDWt1YmVybmV0ZXMtY2EwggEiMA0GCSqGSIb3
DQEBAQUAA4IBDwAwggEKAoIBAQDCTkLqmnxnFq4felqTlt3BOcq2yq0n2MrLW9mc
ZgLzzBxUUk9cRv1M5DRt2A4DHaAsjvmKGiq6z4MEmHCwAcvfX0ZP0qg5hS1ISxnr
p+keTSfvaIAbQxMznN07QgMi2aiFrR2EmcD+a7K0UiiXiPqcs4/b6LzZoLRAKkiE
jnYoU5MrZCcTwiHAsLozn5Gp3kXR8HmwekeddSgM8+QxvZpt6Pjmsrs4Glg6mps3
rwoj7sJOZ7VBpEvUS+65S+Hx7UZiGRkqUK2o9MVyVcMqGalbJRfnVQ+FsdX4Vslg
66l3Ga0JVodS6EEcFlW7dWHlK69FlRnXXWhZDThZeXqptSUdAgMBAAGjYTBfMA4G
A1UdDwEB/wQEAwIChDAdBgNVHSUEFjAUBggrBgEFBQcDAgYIKwYBBQUHAwEwDwYD
VR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUoP2isndWNQkDn5qo9doDaocWEtUwDQYJ
KoZIhvcNAQELBQADggEBAEctHSz9R/bRBIPfmYW5ChhMDmva5kMacpXqYsQwAiZ6
ou69RLHAEk8IWge2oPZpSSBhk84jEc80RGXrHhJYlDA2gDNIiW+01aP0UOqLF8s0
lqBMTBuvQDz4QTBIIQKMhSiFX9v7gEe/i0Pl1im5Ib/Xpl8xsCsxFESA3QW0ZhZd
Mzx255GvToBnVcvY9cNZgtTLk5upS/5V2gjzwoKJBQ4BHgPQ+jHOKRV2Hmk+s2C7
pOlNGSOnmpQKxDNYEPzbyd6OpTte+2iqO6iAtqvxRl/wRf8SdxCCIC0ZVV+Cw3gV
fam+cX/Qv7y99GCSpoKmG0pP7qxZ5EJL5v0hGpsokUc=
-----END CERTIFICATE-----
`

var testCAKey = `
-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAwk5C6pp8ZxauH3pak5bdwTnKtsqtJ9jKy1vZnGYC88wcVFJP
XEb9TOQ0bdgOAx2gLI75ihoqus+DBJhwsAHL319GT9KoOYUtSEsZ66fpHk0n72iA
G0MTM5zdO0IDItmoha0dhJnA/muytFIol4j6nLOP2+i82aC0QCpIhI52KFOTK2Qn
E8IhwLC6M5+Rqd5F0fB5sHpHnXUoDPPkMb2abej45rK7OBpYOpqbN68KI+7CTme1
QaRL1EvuuUvh8e1GYhkZKlCtqPTFclXDKhmpWyUX51UPhbHV+FbJYOupdxmtCVaH
UuhBHBZVu3Vh5SuvRZUZ111oWQ04WXl6qbUlHQIDAQABAoIBABss8vL49Fk+tM+2
PyDRQuaZfJ6gLiOakJJsoDzdj6AldfjdIjhXvWmZqTOLujn5VMOmo4QLMNq71yk3
YNtdBPSS0LStU9XnqHOp/VAWReZ1CBbV2MT3VqIrWE3HZ8TiCE3Z7nzzPCjZSz8p
FoBLKjHsczxgirktXshyoX3YpvHwsAfofHqjHlkLtL367y0x8eBVleU3wwzZI4Ds
1P09dtYBsXXcihBfK4yZ4JDo2zqe/ombSGYLE9cj+L7O0cugfR29WR2Sd/tSVzAk
N1eAkyfdCGA+0UwcN+xANZEGG6LTViuClQgkNaS3jkcGot6pleg1FmtEcsH18d+A
mc85a8kCgYEA2kDm5YXtzXU2r7EyJDYLgrUAEBavLkPyXYlY0GAuywY1f/tazxc9
WUuqe9H6zy5cNnO+I8mv3shc3SHSV8e+OZ99aQv1e3vbZIggpTS7MUaMbg4MW8ic
W8BOcvekYWYsatnv1/hlFchyTEDdcd7JD0zuf0ad/4tigbkM/WSEevMCgYEA4+kU
EEFexncZzXb8SBRUYUv8sepLkPtT+MNjDbpdJlX6pq4RCjXxf4a4yj8tfteR9JJ/
ga3tyDOnkqi/Sk5Q/bpT/1syp7VimFAqhLC4m0JW2vf5D30HEKIMa5pjIvMuzwlz
vaIWpFiwvrlMFaMq6v7EeHtUYjNdohYMfNQKw68CgYAyk4WuPJn92aLBlgtrjsae
FHmeQNN5oi9A87oMF63gSGEPdlz1zond7oXkSaWYa0LdL3cpbex+cOnsKJFI3DW9
vrLeK/JIGkyeAFmoTw7t/U4/lqvQfS2WqXrEc5S5KWczn6tP3fT21kt+Vi263Ii1
Lfu6rM+iT1eVfh9/fNKidwKBgBQP62U26+naiBnvFGwf5gGel8LtlfNQPGcUg/6s
XhDG1safYf6dGwIX0OJ0x0N4JG/8CV9X+St7aI/fbN9Un4qGQWikFYRv0hsIS4Xc
rJN2NoEV/QWhAuMy8Jb0Qy/Lal5tPZP+1bFn4T8YvprU/y0qeg8FBDuUu/RNrpG6
dKwfAoGATzEZVQmIu9/+jDzfpCeHvHghYuZ4fP4+IgHaZz4IZZpHb895OEqverRw
F/b1G7plpVCORHjBUgOLgq716L84pmBwr6nWQGLIftXTwFd/nC5OPo0xwAQDDLNI
RJ48NKPRLpkAcMcXoVo6HglOnmXj425kR9SQeHZQ7C610mLd6XE=
-----END RSA PRIVATE KEY-----
`

func TestReadServiceArgsFile(t *testing.T) {
	g := NewWithT(t)

	t.Run("successfully reads and parses kube-apiserver args file", func(t *testing.T) {
		fileContent := `--advertise-address=10.10.138.127
--allow-privileged=true
--anonymous-auth=false
--authorization-mode=Node,RBAC`

		fs, cleanup, err := vfst.NewTestFS(map[string]interface{}{
			filepath.Join(domain.KubeComponentsArgsPath, "kube-apiserver"): fileContent,
		})
		g.Expect(err).NotTo(HaveOccurred())
		defer cleanup()

		args, err := readServiceArgsFile(fs, "kube-apiserver")

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(*args["--advertise-address"]).To(Equal("10.10.138.127"))
		g.Expect(*args["--allow-privileged"]).To(Equal("true"))
		g.Expect(*args["--anonymous-auth"]).To(Equal("false"))
	})
}

func TestGetApiserverCertFileStage(t *testing.T) {
	g := NewWithT(t)

	t.Run("generates certificate preserving existing SANs and adding new ones", func(t *testing.T) {
		testFS, cleanup, err := vfst.NewTestFS(map[string]interface{}{
			filepath.Join(domain.KubeCertificateDirPath, "apiserver.crt"): string(testApiserverCrt),
			filepath.Join(domain.KubeCertificateDirPath, "ca.crt"):        string(testCACrt),
			filepath.Join(domain.KubeCertificateDirPath, "ca.key"):        string(testCAKey),
		})
		g.Expect(err).NotTo(HaveOccurred())
		defer cleanup()

		originalFS := fs.OSFS
		fs.OSFS = testFS
		defer func() { fs.OSFS = originalFS }()

		existingCertBytes, err := fs.OSFS.ReadFile(filepath.Join(domain.KubeCertificateDirPath, "apiserver.crt"))
		g.Expect(err).NotTo(HaveOccurred())

		existingCertBlock, _ := pem.Decode(existingCertBytes)
		g.Expect(existingCertBlock).NotTo(BeNil(), "Failed to decode existing cert PEM block")

		existingCert, err := x509.ParseCertificate(existingCertBlock.Bytes)
		g.Expect(err).NotTo(HaveOccurred(), "Failed to parse existing certificate")

		// new sans to add
		incomingSans := []string{
			"new.example.com",
			"192.168.1.10",
		}
		apiserverCertPath := filepath.Join(domain.KubeCertificateDirPath, "apiserver.crt")

		stage := getApiserverCertFileStage(incomingSans, apiserverCertPath)

		g.Expect(stage.Name).To(Equal("Regenerate Apiserver Certificates"))
		g.Expect(stage.Files).To(HaveLen(2))

		newCertContent := stage.Files[0].Content
		newCertBlock, _ := pem.Decode([]byte(newCertContent))
		g.Expect(newCertBlock).NotTo(BeNil(), "Failed to decode new cert PEM block")

		newCert, err := x509.ParseCertificate(newCertBlock.Bytes)
		g.Expect(err).NotTo(HaveOccurred(), "Failed to parse new certificate")

		g.Expect(newCert.Subject.CommonName).To(Equal("kube-apiserver"))

		for _, dns := range existingCert.DNSNames {
			g.Expect(newCert.DNSNames).To(ContainElement(dns))
		}

		for _, ip := range existingCert.IPAddresses {
			g.Expect(newCert.IPAddresses).To(ContainElement(ip))
		}

		// verify new sans are added
		g.Expect(newCert.DNSNames).To(ContainElement("new.example.com"))
		expectedIP := net.ParseIP("192.168.1.10")
		var foundIP bool
		for _, ip := range newCert.IPAddresses {
			if ip.Equal(expectedIP) {
				foundIP = true
				break
			}
		}
		g.Expect(foundIP).To(BeTrue())

		caContent, err := fs.OSFS.ReadFile(filepath.Join(domain.KubeCertificateDirPath, "ca.crt"))
		g.Expect(err).NotTo(HaveOccurred())

		caBlock, _ := pem.Decode(caContent)
		g.Expect(caBlock).NotTo(BeNil(), "Failed to decode CA PEM block")

		caCert, err := x509.ParseCertificate(caBlock.Bytes)
		g.Expect(err).NotTo(HaveOccurred(), "Failed to parse CA certificate")

		err = newCert.CheckSignatureFrom(caCert)
		g.Expect(err).NotTo(HaveOccurred(), "New certificate not signed by provided CA")

		g.Expect(stage.Files[0].Path).To(Equal(filepath.Join(domain.KubeCertificateDirPath, "apiserver.crt")))
		g.Expect(stage.Files[0].Permissions).To(Equal(uint32(0600)))

		g.Expect(stage.Files[1].Path).To(Equal(filepath.Join(domain.KubeCertificateDirPath, "apiserver.key")))
		g.Expect(stage.Files[1].Permissions).To(Equal(uint32(0600)))
	})
}

func TestGetArgs(t *testing.T) {
	g := NewWithT(t)

	t.Run("overrides existing args and appends new ones", func(t *testing.T) {
		fileContent := `--advertise-address=10.10.138.127
--allow-privileged=true
--anonymous-auth=false`

		testFS, cleanup, err := vfst.NewTestFS(map[string]interface{}{
			filepath.Join(domain.KubeComponentsArgsPath, "kube-apiserver"): fileContent,
		})
		g.Expect(err).NotTo(HaveOccurred())
		defer cleanup()

		originalFS := fs.OSFS
		fs.OSFS = testFS
		defer func() { fs.OSFS = originalFS }()

		newValue := "10.10.138.200"
		newAuthValue := "true"
		newFeatureValue := "WatchList=false"
		updatedArgs := map[string]*string{
			"--advertise-address": &newValue,
			"--anonymous-auth":    &newAuthValue,
			"--feature-gates":     &newFeatureValue,
		}

		result := getArgs(updatedArgs, "kube-apiserver")
		expectedLines := []string{
			"--advertise-address=10.10.138.200",
			"--allow-privileged=true",
			"--anonymous-auth=true",
			"--feature-gates=WatchList=false",
		}

		resultLines := strings.Split(result, "\n")
		sort.Strings(resultLines)
		sort.Strings(expectedLines)

		g.Expect(resultLines).To(Equal(expectedLines))
	})

	t.Run("handles empty current args file", func(t *testing.T) {
		testFS, cleanup, err := vfst.NewTestFS(map[string]interface{}{
			filepath.Join(domain.KubeComponentsArgsPath, "kube-apiserver"): "",
		})
		g.Expect(err).NotTo(HaveOccurred())
		defer cleanup()

		originalFS := fs.OSFS
		fs.OSFS = testFS
		defer func() { fs.OSFS = originalFS }()

		value := "test-value"
		updatedArgs := map[string]*string{
			"--new-arg": &value,
		}
		result := getArgs(updatedArgs, "kube-apiserver")
		g.Expect(result).To(Equal("--new-arg=test-value"))
	})
}

func TestContainsAnyNonMatch(t *testing.T) {
	g := NewWithT(t)

	t.Run("returns true when source has elements not in target", func(t *testing.T) {
		sources := []string{"a", "b", "c"}
		targets := []string{"a", "b"}
		result := containsAnyNonMatch(sources, targets)
		g.Expect(result).To(BeTrue())
	})

	t.Run("returns false when all source elements are in target", func(t *testing.T) {
		sources := []string{"a", "b"}
		targets := []string{"a", "b", "c"}
		result := containsAnyNonMatch(sources, targets)
		g.Expect(result).To(BeFalse())
	})

	t.Run("returns false for empty source", func(t *testing.T) {
		sources := []string{}
		targets := []string{"a", "b"}
		result := containsAnyNonMatch(sources, targets)
		g.Expect(result).To(BeFalse())
	})

	t.Run("returns true for empty target", func(t *testing.T) {
		sources := []string{"a"}
		targets := []string{}
		result := containsAnyNonMatch(sources, targets)
		g.Expect(result).To(BeTrue())
	})
}
