package utils

import (
	"net"

	"github.com/kairos-io/provider-canonical/pkg/domain"
)

func IsProxyConfigured(proxyMap map[string]string) bool {
	return len(proxyMap["HTTP_PROXY"]) > 0 || len(proxyMap["HTTPS_PROXY"]) > 0
}

func GetDefaultNoProxy(clusterCtx *domain.ClusterContext) string {
	var noProxy string

	clusterCidr := clusterCtx.ClusterCidr
	serviceCidr := clusterCtx.ServiceCidr

	if len(clusterCidr) > 0 {
		noProxy = clusterCidr
	}

	if len(serviceCidr) > 0 {
		noProxy = noProxy + "," + serviceCidr + "," + getFirstIpServiceCidr(serviceCidr)
	}
	return noProxy + "," + domain.K8sNoProxy
}

func getFirstIpServiceCidr(serviceCidr string) string {
	return listIPs(serviceCidr)[0]
}

func listIPs(cidr string) []string {
	ip, ipnet, _ := net.ParseCIDR(cidr)

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incIP(ip) {
		ips = append(ips, ip.String())
	}

	// Remove network and broadcast addresses if applicable
	if len(ips) > 2 {
		return ips[1 : len(ips)-1]
	}

	return ips
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] != 0 {
			break
		}
	}
}

func GetNoProxyConfig(clusterCtx *domain.ClusterContext) string {
	defaultNoProxy := GetDefaultNoProxy(clusterCtx)
	userNoProxy := clusterCtx.EnvConfig["NO_PROXY"]
	if len(userNoProxy) > 0 {
		return defaultNoProxy + "," + userNoProxy
	}
	return defaultNoProxy
}
