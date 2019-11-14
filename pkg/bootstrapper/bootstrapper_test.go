package bootstrapper

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTranslateFile tests decoding and transforming ignition file sources
func TestTranslateFile(t *testing.T) {
	type args struct {
		input  string
		lambda translationFunc
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "No translation function",
			args: args{
				input:  "data:,-----BEGIN%20CERTIFICATE-----%0AMIIDEDCCAfigAwIBAgIIKH9ePWRYTs8wDQYJKoZIhvcNAQELBQAwJjESMBAGA1UE%0ACxMJb3BlbnNoaWZ0MRAwDgYDVQQDEwdyb290LWNhMB4XDTE5MDkxMDE0MjkzMloX%0ADTI5MDkwNzE0MjkzMlowJjESMBAGA1UECxMJb3BlbnNoaWZ0MRAwDgYDVQQDEwdy%0Ab290LWNhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4FcWyu6Nsdb5%0A%2Bw0r1101FTPjw2W392K8mJgm8tI852WxnVdC41vpkpreNZhHpef2LYemRbX3LVv5%0AEw3Ovuaz%2FKcsVASg5MpP0XgzFUhHT1UgAdFvh08GtUGZedXb9di66TJHnYoVrSsJ%0Ad%2FuZnRIT7dsR%2BVdmMhB0N2vcBsLOilG3XaR24h3UmeB8cqkKxzmaG2dKf1Z1MiyM%0AkP%2Fy73wzKEMtWPjNA%2BJaJdNf4n7Mh57fwO9IMrmMQWZP7d%2B8kFMnfQygXPopqFQR%0ADhOjG1D52hzExHWD08ShnossHJWt9ETo2eb9D1djf3E%2BwCZ7HQV8J5V6WlO8wR0R%0AC8fjKImLjQIDAQABo0IwQDAOBgNVHQ8BAf8EBAMCAqQwDwYDVR0TAQH%2FBAUwAwEB%0A%2FzAdBgNVHQ4EFgQUUEKZ3tCtmqwA26fFx0N%2Bd%2BAxxOkwDQYJKoZIhvcNAQELBQAD%0AggEBAAqAeBN7G5S1hsDiNd2lZwI5eNuGGk5T5tOEwCIuKHaSxnwkmn7qKymjsm42%0A%2BSKzN63i%2FSreK8CONW6Xp8kUNQW3J6iziRQD11uR8jZVoezqCW7%2BfWZmD4VBrUqI%0AFbrOEMZbc9vPxvpbN%2FinzKJoSLUGTtzN7CjsLmf4XdTFtEr9qBPpOFb0i3gaYn%2Fx%0AK58cZ7SBbK9oyk%2FCF2St%2F9TR7unuNFDq1TPsjSKxJMC%2FsTyEcW6ABCOjcqu94eWt%0AUHfH1Be25D8kcN0%2FtdrJt4NgawQINUr0QIkSsY%2B3hh8AUHSvyCbiiCrt%2Fn7jjF7G%0ArqLuyNO%2BhCh%2FZclPL%2BUiGJH1dlQ%3D%0A-----END%20CERTIFICATE-----",
				lambda: nil,
			},
			want: []byte(`-----BEGIN CERTIFICATE-----
MIIDEDCCAfigAwIBAgIIKH9ePWRYTs8wDQYJKoZIhvcNAQELBQAwJjESMBAGA1UE
CxMJb3BlbnNoaWZ0MRAwDgYDVQQDEwdyb290LWNhMB4XDTE5MDkxMDE0MjkzMloX
DTI5MDkwNzE0MjkzMlowJjESMBAGA1UECxMJb3BlbnNoaWZ0MRAwDgYDVQQDEwdy
b290LWNhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4FcWyu6Nsdb5
+w0r1101FTPjw2W392K8mJgm8tI852WxnVdC41vpkpreNZhHpef2LYemRbX3LVv5
Ew3Ovuaz/KcsVASg5MpP0XgzFUhHT1UgAdFvh08GtUGZedXb9di66TJHnYoVrSsJ
d/uZnRIT7dsR+VdmMhB0N2vcBsLOilG3XaR24h3UmeB8cqkKxzmaG2dKf1Z1MiyM
kP/y73wzKEMtWPjNA+JaJdNf4n7Mh57fwO9IMrmMQWZP7d+8kFMnfQygXPopqFQR
DhOjG1D52hzExHWD08ShnossHJWt9ETo2eb9D1djf3E+wCZ7HQV8J5V6WlO8wR0R
C8fjKImLjQIDAQABo0IwQDAOBgNVHQ8BAf8EBAMCAqQwDwYDVR0TAQH/BAUwAwEB
/zAdBgNVHQ4EFgQUUEKZ3tCtmqwA26fFx0N+d+AxxOkwDQYJKoZIhvcNAQELBQAD
ggEBAAqAeBN7G5S1hsDiNd2lZwI5eNuGGk5T5tOEwCIuKHaSxnwkmn7qKymjsm42
+SKzN63i/SreK8CONW6Xp8kUNQW3J6iziRQD11uR8jZVoezqCW7+fWZmD4VBrUqI
FbrOEMZbc9vPxvpbN/inzKJoSLUGTtzN7CjsLmf4XdTFtEr9qBPpOFb0i3gaYn/x
K58cZ7SBbK9oyk/CF2St/9TR7unuNFDq1TPsjSKxJMC/sTyEcW6ABCOjcqu94eWt
UHfH1Be25D8kcN0/tdrJt4NgawQINUr0QIkSsY+3hh8AUHSvyCbiiCrt/n7jjF7G
rqLuyNO+hCh/ZclPL+UiGJH1dlQ=
-----END CERTIFICATE-----`),
		},
		{
			name: "Using translation function",
			args: args{
				input: "data:,-----BEGIN%20CERTIFICATE-----%0AMIIDEDCCAfigAwIBAgIIKH9ePWRYTs9wDQYJKoZIhvcNAQELBQAwJjESMBAGA1UE%0ACxMJb3BlbnNoaWZ0MRAwDgYDVQQDEwdyb290LWNhMB4XDTE5MDkxMDE0MjkzMloX%0ADTI5MDkwNzE0MjkzMlowJjESMBAGA1UECxMJb3BlbnNoaWZ0MRAwDgYDVQQDEwdy%0Ab290LWNhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4FcWyu6Nsdb5%0A%2Bw0r1101FTPjw2W392K8mJgm8tI852WxnVdC41vpkpreNZhHpef2LYemRbX3LVv5%0AEw3Ovuaz%2FKcsVASg5MpP0XgzFUhHT1UgAdFvh08GtUGZedXb9di66TJHnYoVrSsJ%0Ad%2FuZnRIT7dsR%2BVdmMhB0N2vcBsLOilG3XaR24h3UmeB8cqkKxzmaG2dKf1Z1MiyM%0AkP%2Fy73wzKEMtWPjNA%2BJaJdNf4n7Mh57fwO9IMrmMQWZP7d%2B8kFMnfQygXPopqFQR%0ADhOjG1D52hzExHWD08ShnossHJWt9ETo2eb9D1djf3E%2BwCZ7HQV8J5V6WlO8wR0R%0AC8fjKImLjQIDAQABo0IwQDAOBgNVHQ8BAf8EBAMCAqQwDwYDVR0TAQH%2FBAUwAwEB%0A%2FzAdBgNVHQ4EFgQUUEKZ3tCtmqwA26fFx0N%2Bd%2BAxxOkwDQYJKoZIhvcNAQELBQAD%0AggEBAAqAeBN7G5S1hsDiNd2lZwI5eNuGGk5T5tOEwCIuKHaSxnwkmn7qKymjsm42%0A%2BSKzN63i%2FSreK8CONW6Xp8kUNQW3J6iziRQD11uR8jZVoezqCW7%2BfWZmD4VBrUqI%0AFbrOEMZbc9vPxvpbN%2FinzKJoSLUGTtzN7CjsLmf4XdTFtEr9qBPpOFb0i3gaYn%2Fx%0AK58cZ7SBbK9oyk%2FCF2St%2F9TR7unuNFDq1TPsjSKxJMC%2FsTyEcW6ABCOjcqu94eWt%0AUHfH1Be25D8kcN0%2FtdrJt4NgawQINUr0QIkSsY%2B3hh8AUHSvyCbiiCrt%2Fn7jjF7G%0ArqLuyNO%2BhCh%2FZclPL%2BUiGJH1dlQ%3D%0A-----END%20CERTIFICATE-----",
				lambda: func(bs *winNodeBootstrapper, in []byte) ([]byte, error) {
					return []byte(string(in) + "suffix"), nil
				},
			},
			want: []byte(`-----BEGIN CERTIFICATE-----
MIIDEDCCAfigAwIBAgIIKH9ePWRYTs9wDQYJKoZIhvcNAQELBQAwJjESMBAGA1UE
CxMJb3BlbnNoaWZ0MRAwDgYDVQQDEwdyb290LWNhMB4XDTE5MDkxMDE0MjkzMloX
DTI5MDkwNzE0MjkzMlowJjESMBAGA1UECxMJb3BlbnNoaWZ0MRAwDgYDVQQDEwdy
b290LWNhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4FcWyu6Nsdb5
+w0r1101FTPjw2W392K8mJgm8tI852WxnVdC41vpkpreNZhHpef2LYemRbX3LVv5
Ew3Ovuaz/KcsVASg5MpP0XgzFUhHT1UgAdFvh08GtUGZedXb9di66TJHnYoVrSsJ
d/uZnRIT7dsR+VdmMhB0N2vcBsLOilG3XaR24h3UmeB8cqkKxzmaG2dKf1Z1MiyM
kP/y73wzKEMtWPjNA+JaJdNf4n7Mh57fwO9IMrmMQWZP7d+8kFMnfQygXPopqFQR
DhOjG1D52hzExHWD08ShnossHJWt9ETo2eb9D1djf3E+wCZ7HQV8J5V6WlO8wR0R
C8fjKImLjQIDAQABo0IwQDAOBgNVHQ8BAf8EBAMCAqQwDwYDVR0TAQH/BAUwAwEB
/zAdBgNVHQ4EFgQUUEKZ3tCtmqwA26fFx0N+d+AxxOkwDQYJKoZIhvcNAQELBQAD
ggEBAAqAeBN7G5S1hsDiNd2lZwI5eNuGGk5T5tOEwCIuKHaSxnwkmn7qKymjsm42
+SKzN63i/SreK8CONW6Xp8kUNQW3J6iziRQD11uR8jZVoezqCW7+fWZmD4VBrUqI
FbrOEMZbc9vPxvpbN/inzKJoSLUGTtzN7CjsLmf4XdTFtEr9qBPpOFb0i3gaYn/x
K58cZ7SBbK9oyk/CF2St/9TR7unuNFDq1TPsjSKxJMC/sTyEcW6ABCOjcqu94eWt
UHfH1Be25D8kcN0/tdrJt4NgawQINUr0QIkSsY+3hh8AUHSvyCbiiCrt/n7jjF7G
rqLuyNO+hCh/ZclPL+UiGJH1dlQ=
-----END CERTIFICATE-----suffix`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := winNodeBootstrapper{installDir: filepath.Base("tmp")}
			got, err := bs.translateFile(tt.args.input, tt.args.lambda)
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, got, "got = %v, want %v", string(got), string(tt.want))
		})
	}
}

// TestPrepKubeletConfForWindows tests that we are changing the kubelet configuration in a way that allows it to run on windows
func TestPrepKubeletConfForWindows(t *testing.T) {
	type args struct {
		in []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "Base case",
			args: args{in: []byte(`{"kind":"KubeletConfiguration","apiVersion":"kubelet.config.k8s.io/v1beta1","staticPodPath":"/etc/kubernetes/manifests","syncFrequency":"0s","fileCheckFrequency":"0s","httpCheckFrequency":"0s","rotateCertificates":true,"serverTLSBootstrap":true,"authentication":{"x509":{"clientCAFile":"/etc/kubernetes/kubelet-ca.crt"},"webhook":{"cacheTTL":"0s"},"anonymous":{"enabled":false}},"authorization":{"webhook":{"cacheAuthorizedTTL":"0s","cacheUnauthorizedTTL":"0s"}},"clusterDomain":"cluster.local","clusterDNS":["172.30.0.10"],"streamingConnectionIdleTimeout":"0s","nodeStatusUpdateFrequency":"0s","nodeStatusReportFrequency":"0s","imageMinimumGCAge":"0s","volumeStatsAggPeriod":"0s","cgroupDriver":"systemd","cpuManagerReconcilePeriod":"0s","runtimeRequestTimeout":"10m0s","maxPods":250,"serializeImagePulls":false,"evictionPressureTransitionPeriod":"0s","featureGates":{"ExperimentalCriticalPodAnnotation":true,"LocalStorageCapacityIsolation":false,"RotateKubeletServerCertificate":true,"SupportPodPidsLimit":true},"containerLogMaxSize":"50Mi","systemReserved":{"cpu":"500m","memory":"500Mi"}}`)},
			want: []byte(`{"kind":"KubeletConfiguration","apiVersion":"kubelet.config.k8s.io/v1beta1","staticPodPath":"/etc/kubernetes/manifests","syncFrequency":"0s","fileCheckFrequency":"0s","httpCheckFrequency":"0s","rotateCertificates":true,"serverTLSBootstrap":true,"authentication":{"x509":{"clientCAFile":"C:\\k\\kubelet-ca.crt"},"webhook":{"cacheTTL":"0s"},"anonymous":{"enabled":false}},"authorization":{"webhook":{"cacheAuthorizedTTL":"0s","cacheUnauthorizedTTL":"0s"}},"clusterDomain":"cluster.local","clusterDNS":["172.30.0.10"],"streamingConnectionIdleTimeout":"0s","nodeStatusUpdateFrequency":"0s","nodeStatusReportFrequency":"0s","imageMinimumGCAge":"0s","volumeStatsAggPeriod":"0s","cgroupsPerQOS":false,"cgroupDriver":"cgroupfs","cpuManagerReconcilePeriod":"0s","runtimeRequestTimeout":"10m0s","maxPods":250,"serializeImagePulls":false,"evictionPressureTransitionPeriod":"0s","featureGates":{"ExperimentalCriticalPodAnnotation":true,"LocalStorageCapacityIsolation":false,"RotateKubeletServerCertificate":true,"SupportPodPidsLimit":true},"containerLogMaxSize":"50Mi","systemReserved":{"cpu":"500m","memory":"500Mi"},"enforceNodeAllocatable":[]}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := winNodeBootstrapper{installDir: `C:\k`}
			got, err := prepKubeletConfForWindows(&bs, tt.args.in)
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, got, "got = %v, want %v", string(got), string(tt.want))
		})
	}
}

// TestCloudConfExtraction tests if parseIgnitionFileContents can extract the cloud.conf present in a worker ignition
// file contents and the resulting file is in the expected format with a set of key value pairs.
// It also confirms the "--cloud-config" option constructed by WMCB is as expected. Example cloud.conf:
// {
//	"cloud": "AzurePublicCloud",
//	"tenantId": "1234a1b2-a1bc-123a-123a-ab1c2de3afgh",
//	"aadClientId": "",
//	"aadClientSecret": "",
//	"aadClientCertPath": "",
//	"aadClientCertPassword": "",
//	"useManagedIdentityExtension": true,
//	"userAssignedIdentityID": "",
//	"subscriptionId": "1a123456-12ab-123a-1234-abc1d1ab01c0",
//	"resourceGroup": "winc-test-rg",
//	"location": "centralus",
//	"vnetName": "winc-test-vnet",
//	"vnetResourceGroup": "winc-test-rg",
//	"subnetName": "winc-test-node-subnet",
//	"securityGroupName": "winc-test-node-nsg",
//	"routeTableName": "winc-test-node-routetable",
//	"primaryAvailabilitySetName": "",
//	"vmType": "",
//	"primaryScaleSetName": "",
//	"cloudProviderBackoff": true,
//	"cloudProviderBackoffRetries": 0,
//	"cloudProviderBackoffExponent": 0,
//	"cloudProviderBackoffDuration": 6,
//	"cloudProviderBackoffJitter": 0,
//	"cloudProviderRateLimit": true,
//	"cloudProviderRateLimitQPS": 6,
//	"cloudProviderRateLimitBucket": 10,
//	"cloudProviderRateLimitQPSWrite": 6,
//	"cloudProviderRateLimitBucketWrite": 10,
//	"useInstanceMetadata": true,
//	"loadBalancerSku": "standard",
//	"excludeMasterFromStandardLB": null,
//	"disableOutboundSNAT": null,
//	"maximumLoadBalancerRuleCount": 0
//}
func TestCloudConfExtraction(t *testing.T) {
	// ignitionContents is the actual worker ignition contents from an azure cluster with dummy credentials and
	// resources
	ignitionContents := `{"ignition":{"config":{},"security":{"tls":{}},"timeouts":{},"version":"2.2.0"},"networkd":{},"passwd":{"users":[{"name":"core","sshAuthorizedKeys":["ssh-rsa dummy"]}]},"storage":{"files":[{"filesystem":"root","path":"/etc/kubernetes/cloud.conf","contents":{"source":"data:,%7B%0A%09%22cloud%22%3A%20%22AzurePublicCloud%22%2C%0A%09%22tenantId%22%3A%20%221234a1b2-a1bc-123a-123a-ab1c2de3afgh%22%2C%0A%09%22aadClientId%22%3A%20%22%22%2C%0A%09%22aadClientSecret%22%3A%20%22%22%2C%0A%09%22aadClientCertPath%22%3A%20%22%22%2C%0A%09%22aadClientCertPassword%22%3A%20%22%22%2C%0A%09%22useManagedIdentityExtension%22%3A%20true%2C%0A%09%22userAssignedIdentityID%22%3A%20%22%22%2C%0A%09%22subscriptionId%22%3A%20%221a123456-12ab-123a-1234-abc1d1ab01c0%22%2C%0A%09%22resourceGroup%22%3A%20%22winc-test-vnet%22%2C%0A%09%22location%22%3A%20%22centralus%22%2C%0A%09%22vnetName%22%3A%20%22winc-test-vnet%22%2C%0A%09%22vnetResourceGroup%22%3A%20%22winc-test-rg%22%2C%0A%09%22subnetName%22%3A%20%22winc-test-node-subnet%22%2C%0A%09%22securityGroupName%22%3A%20%22winc-test-node-nsg%22%2C%0A%09%22routeTableName%22%3A%20%22winc-test-node-routetable%22%2C%0A%09%22primaryAvailabilitySetName%22%3A%20%22%22%2C%0A%09%22vmType%22%3A%20%22%22%2C%0A%09%22primaryScaleSetName%22%3A%20%22%22%2C%0A%09%22cloudProviderBackoff%22%3A%20true%2C%0A%09%22cloudProviderBackoffRetries%22%3A%200%2C%0A%09%22cloudProviderBackoffExponent%22%3A%200%2C%0A%09%22cloudProviderBackoffDuration%22%3A%206%2C%0A%09%22cloudProviderBackoffJitter%22%3A%200%2C%0A%09%22cloudProviderRateLimit%22%3A%20true%2C%0A%09%22cloudProviderRateLimitQPS%22%3A%206%2C%0A%09%22cloudProviderRateLimitBucket%22%3A%2010%2C%0A%09%22cloudProviderRateLimitQPSWrite%22%3A%206%2C%0A%09%22cloudProviderRateLimitBucketWrite%22%3A%2010%2C%0A%09%22useInstanceMetadata%22%3A%20true%2C%0A%09%22loadBalancerSku%22%3A%20%22standard%22%2C%0A%09%22excludeMasterFromStandardLB%22%3A%20null%2C%0A%09%22disableOutboundSNAT%22%3A%20null%2C%0A%09%22maximumLoadBalancerRuleCount%22%3A%200%0A%7D","verification":{}},"mode":420}]},"systemd":{"units":[{"contents":"[Unit]\nDescription=Kubernetes Kubelet\nWants=rpc-statd.service crio.service\nAfter=crio.service\n\n[Service]\nType=notify\nExecStartPre=/bin/mkdir --parents /etc/kubernetes/manifests\nExecStartPre=/bin/rm -f /var/lib/kubelet/cpu_manager_state\nEnvironmentFile=/etc/os-release\nEnvironmentFile=-/etc/kubernetes/kubelet-workaround\nEnvironmentFile=-/etc/kubernetes/kubelet-env\n\nExecStart=/usr/bin/hyperkube \\\n    kubelet \\\n      --config=/etc/kubernetes/kubelet.conf \\\n      --bootstrap-kubeconfig=/etc/kubernetes/kubeconfig \\\n      --kubeconfig=/var/lib/kubelet/kubeconfig \\\n      --container-runtime=remote \\\n      --container-runtime-endpoint=/var/run/crio/crio.sock \\\n      --node-labels=node-role.kubernetes.io/worker,node.openshift.io/os_id=${ID} \\\n      --minimum-container-ttl-duration=6m0s \\\n      --volume-plugin-dir=/etc/kubernetes/kubelet-plugins/volume/exec \\\n      --cloud-provider=azure \\\n      --cloud-config=/etc/kubernetes/cloud.conf \\\n      --v=3\n\nRestart=always\nRestartSec=10\n\n[Install]\nWantedBy=multi-user.target\n","enabled":true,"name":"kubelet.service"}]}}`

	// Create a temp directory with wmcb prefix
	dir, err := ioutil.TempDir("", "wmcb")
	require.NoError(t, err, "error creating temp directory")
	// Ignore the return error as there is not much we can do if the temporary directory is not deleted
	defer os.RemoveAll(dir)

	wnb := winNodeBootstrapper{
		installDir:  dir,
		kubeletArgs: make(map[string]string),
	}

	err = wnb.parseIgnitionFileContents([]byte(ignitionContents), map[string]fileTranslation{})
	assert.NoError(t, err, "error parsing ignition file contents")
	assert.FileExists(t, path.Join(dir, "cloud.conf"), "cloud.conf was not created")

	confContents, err := ioutil.ReadFile(path.Join(dir, "cloud.conf"))
	assert.NoError(t, err, "error reading cloud.conf")

	conf := string(confContents)
	// Check if the file beings with { and ends with }
	assert.True(t, strings.HasPrefix(conf, "{"))
	assert.True(t, strings.HasSuffix(conf, "}"))

	// Replace the beginning {\n\t, \n}, with ""
	conf = strings.Replace(conf, "{\n\t", "", -1)
	conf = strings.Replace(conf, "\n}", "", -1)

	// Split the conf items into an array. Each element will now contain "key: value"
	confItems := strings.Split(conf, ",\n\t")

	// Expected key value pairs from ignitionContents
	confExpected := map[string]string{
		"cloud":             "AzurePublicCloud",
		"tenantId":          "1234a1b2-a1bc-123a-123a-ab1c2de3afgh",
		"subscriptionId":    "1a123456-12ab-123a-1234-abc1d1ab01c0",
		"resourceGroup":     "winc-test-rg",
		"location":          "centralus",
		"vnetName":          "winc-test-vnet",
		"vnetResourceGroup": "winc-test-rg",
		"subnetName":        "winc-test-node-subnet",
		"securityGroupName": "winc-test-node-nsg",
		"routeTableName":    "winc-test-node-routetable",
	}

	for _, confItem := range confItems {
		// keyValue will have two elements, 0 being the key and 1 the value
		keyValue := strings.Split(confItem, ":")
		assert.True(t, len(keyValue) == 2)

		// Check if the key needs to be compared
		value, present := confExpected[keyValue[0]]
		if !present {
			continue
		}

		// Assert that the key value from the file matches the value in the ignition contents
		assert.Equal(t, confExpected[keyValue[0]], value)
	}

	// Check that the --cloud-conf option value is present in the kubelet args and matches tempdir + /cloud.conf
	cloudConfigOptValue, present := wnb.kubeletArgs["cloud-config"]
	assert.True(t, present, "cloud-config option is not present in kubelet args")
	assert.Equal(t, path.Join(dir, "cloud.conf"), cloudConfigOptValue,
		"unexpected --cloud-config value %s", cloudConfigOptValue)
}

// TestCloudConfNotPresent tests that parseIgnitionFileContents will only create a cloud.conf file and add the
// "--cloud-config" option to the kubelet args, if the cloud.conf file is present in the ignition file.
func TestCloudConfNotPresent(t *testing.T) {
	// ignitionContents is the actual worker ignition contents from an azure cluster with dummy credentials and
	// resources
	ignitionContents := `{"ignition":{"config":{},"security":{"tls":{}},"timeouts":{},"version":"2.2.0"},"networkd":{},"passwd":{"users":[{"name":"core","sshAuthorizedKeys":["ssh-rsa dummy"]}]},"storage":{"files":[]},"systemd":{"units":[{"contents":"[Unit]\nDescription=Kubernetes Kubelet\nWants=rpc-statd.service crio.service\nAfter=crio.service\n\n[Service]\nType=notify\nExecStartPre=/bin/mkdir --parents /etc/kubernetes/manifests\nExecStartPre=/bin/rm -f /var/lib/kubelet/cpu_manager_state\nEnvironmentFile=/etc/os-release\nEnvironmentFile=-/etc/kubernetes/kubelet-workaround\nEnvironmentFile=-/etc/kubernetes/kubelet-env\n\nExecStart=/usr/bin/hyperkube \\\n    kubelet \\\n      --config=/etc/kubernetes/kubelet.conf \\\n      --bootstrap-kubeconfig=/etc/kubernetes/kubeconfig \\\n      --kubeconfig=/var/lib/kubelet/kubeconfig \\\n      --container-runtime=remote \\\n      --container-runtime-endpoint=/var/run/crio/crio.sock \\\n      --node-labels=node-role.kubernetes.io/worker,node.openshift.io/os_id=${ID} \\\n      --minimum-container-ttl-duration=6m0s \\\n      --volume-plugin-dir=/etc/kubernetes/kubelet-plugins/volume/exec \\\n      --cloud-provider=aws \\\n      --v=3\n\nRestart=always\nRestartSec=10\n\n[Install]\nWantedBy=multi-user.target\n","enabled":true,"name":"kubelet.service"}]}}`

	// Create a temp directory with wmcb prefix
	dir, err := ioutil.TempDir("", "wmcb")
	require.NoError(t, err, "error creating temp directory")
	// Ignore the return error as there is not much we can do if the temporary directory is not deleted
	defer os.RemoveAll(dir)

	wnb := winNodeBootstrapper{
		installDir:  dir,
		kubeletArgs: make(map[string]string),
	}

	err = wnb.parseIgnitionFileContents([]byte(ignitionContents), map[string]fileTranslation{})
	assert.NoError(t, err, "error parsing ignition file contents")

	_, err = os.Stat(path.Join(dir, "cloud.conf"))
	assert.Error(t, err, "cloud.conf was created")

	// Check that the --cloud-conf option value is not present in the kubelet args
	_, present := wnb.kubeletArgs["cloud-config"]
	assert.False(t, present, "cloud-config option is not present in kubelet args")
}

// TestCloudConfInvalidNames tests that an error is thrown when an ignition file has an invalid "--cloud-config"
// kubelet argument
func TestCloudConfInvalidNames(t *testing.T) {
	// ignitionContents is the actual worker ignition contents from an azure cluster with dummy credentials and
	// resources. The "--cloud-config=/" option is incorrect here.
	ignitionContents := `{"ignition":{"config":{},"security":{"tls":{}},"timeouts":{},"version":"2.2.0"},"networkd":{},"passwd":{"users":[{"name":"core","sshAuthorizedKeys":["ssh-rsa dummy"]}]},"storage":{"files":[{"filesystem":"root","path":"/etc/kubernetes/cloud.conf","contents":{"source":"data:,not needed","verification":{}},"mode":420}]},"systemd":{"units":[{"contents":"[Unit]\nDescription=Kubernetes Kubelet\nWants=rpc-statd.service crio.service\nAfter=crio.service\n\n[Service]\nType=notify\nExecStartPre=/bin/mkdir --parents /etc/kubernetes/manifests\nExecStartPre=/bin/rm -f /var/lib/kubelet/cpu_manager_state\nEnvironmentFile=/etc/os-release\nEnvironmentFile=-/etc/kubernetes/kubelet-workaround\nEnvironmentFile=-/etc/kubernetes/kubelet-env\n\nExecStart=/usr/bin/hyperkube \\\n    kubelet \\\n      --config=/etc/kubernetes/kubelet.conf \\\n      --bootstrap-kubeconfig=/etc/kubernetes/kubeconfig \\\n      --kubeconfig=/var/lib/kubelet/kubeconfig \\\n      --container-runtime=remote \\\n      --container-runtime-endpoint=/var/run/crio/crio.sock \\\n      --node-labels=node-role.kubernetes.io/worker,node.openshift.io/os_id=${ID} \\\n      --minimum-container-ttl-duration=6m0s \\\n      --volume-plugin-dir=/etc/kubernetes/kubelet-plugins/volume/exec \\\n      --cloud-provider=azure \\\n      --cloud-config=/ \\\n      --v=3\n\nRestart=always\nRestartSec=10\n\n[Install]\nWantedBy=multi-user.target\n","enabled":true,"name":"kubelet.service"}]}}`

	wnb := winNodeBootstrapper{
		installDir:  "/",
		kubeletArgs: make(map[string]string),
	}
	err := wnb.parseIgnitionFileContents([]byte(ignitionContents), map[string]fileTranslation{})
	assert.Error(t, err, "error not thrown on encountering invalid --cloud-config option")
}

// TestEnsureCNIConfigDirIsPresent tests ensureCNIConfigDirIsPresent creates the CNI directory when a valid install path
// is passed to wmcb
func TestEnsureCNIConfigDirIsPresent(t *testing.T) {
	// Create a temp directory with wmcb prefix
	installDir, err := ioutil.TempDir("", "wmcb")
	require.NoError(t, err, "error creating temp directory")
	// Ignore the return error as there is not much we can do if the temporary directory is not deleted
	defer os.RemoveAll(installDir)

	wnb := winNodeBootstrapper{
		installDir: installDir,
	}

	err = wnb.ensureCNIDirIsPresent()
	assert.NoError(t, err, "error creating CNI config directory %s", cniDirName)
	assert.DirExists(t, filepath.Join(installDir, "cni", "config"), "CNI directory was not created")
}

// TestCheckCNIInputsInvalidInputs tests if checkCNIInputs throws the expected errors on passing invalid inputs
func TestCheckCNIInputsInvalidInputs(t *testing.T) {
	wnb := winNodeBootstrapper{
		installDir: "C:\\DoesNotExist",
	}

	err := wnb.checkCNIInputs()
	assert.Error(t, err, "no error on passing bad install dir")
	assert.Contains(t, err.Error(), "error accessing install directory", "incorrect error thrown")

	// Create a temp directory with wmcb prefix
	installDir, err := ioutil.TempDir("", "wmcb")
	require.NoError(t, err, "error creating temp directory")
	// Ignore the return error as there is not much we can do if the temporary directory is not deleted
	defer os.RemoveAll(installDir)

	// Bad CNI path
	wnb = winNodeBootstrapper{
		installDir: installDir,
		cniPath:    "C:\\DoesNotExists",
	}

	err = wnb.checkCNIInputs()
	assert.Error(t, err, "no error on passing bad CNI path")
	assert.Contains(t, err.Error(), "error accessing CNI path", "incorrect error thrown")

	// CNI path as a file
	cniPathFile, err := ioutil.TempFile(installDir, "cni")
	require.NoError(t, err, "error creating file directory")

	wnb = winNodeBootstrapper{
		installDir: installDir,
		cniPath:    cniPathFile.Name(),
	}

	err = wnb.checkCNIInputs()
	assert.Error(t, err, "no error on passing file as CNI path")
	assert.Contains(t, err.Error(), "CNI path cannot be a file", "incorrect error thrown")

	// Bad CNI config
	wnb = winNodeBootstrapper{
		installDir: installDir,
		cniPath:    installDir,
		cniConfig:  "C:\\DoesNotExist.conf",
	}

	err = wnb.checkCNIInputs()
	assert.Error(t, err, "no error on passing bad CNI config")
	assert.Contains(t, err.Error(), "error accessing CNI config", "incorrect error thrown")

	// CNI config as directory
	wnb = winNodeBootstrapper{
		installDir: installDir,
		cniPath:    installDir,
		cniConfig:  installDir,
	}

	err = wnb.checkCNIInputs()
	assert.Error(t, err, "no error on passing dir as CNI config")
	assert.Contains(t, err.Error(), "CNI config cannot be a directory", "incorrect error thrown")
}

// TestCopyCNIFilesInvalidInputs tests if copyCNIFiles() throws the expected errors on passing invalid inputs
func TestCopyCNIFilesInvalidInputs(t *testing.T) {
	// Create a temp directory with wmcb prefix
	installDir, err := ioutil.TempDir("", "wmcb")
	require.NoError(t, err, "error creating temp directory")
	// Ignore the return error as there is not much we can do if the temporary directory is not deleted
	defer os.RemoveAll(installDir)

	// Bad CNI path
	wnb := winNodeBootstrapper{
		installDir: installDir,
		cniPath:    "C:\\DoesNotExists",
	}

	err = wnb.copyCNIFiles()
	assert.Error(t, err, "no error on passing bad CNI path")
	assert.Contains(t, err.Error(), "error reading CNI path", "incorrect error thrown")

	// No files
	wnb = winNodeBootstrapper{
		installDir: installDir,
		cniPath:    installDir,
	}

	err = wnb.copyCNIFiles()
	assert.Error(t, err, "no error on passing empty CNI path")
	assert.Contains(t, err.Error(), "no files present", "incorrect error thrown")
}

// TestCopyCNIFiles tests if copyCNIFiles() copies the CNI input binaries and config to the appropriate install location
func TestCopyCNIFiles(t *testing.T) {
	// Create a temp directory with wmcb prefix
	installDir, err := ioutil.TempDir("", "wmcb")
	require.NoError(t, err, "error creating temp directory")
	// Ignore the return error as there is not much we can do if the temporary directory is not deleted
	defer os.RemoveAll(installDir)

	// Create a temp directory with cni prefix
	cniPath, err := ioutil.TempDir("", "cni")
	require.NoError(t, err, "error creating temp CNI directory")
	// Ignore the return error as there is not much we can do if the temporary directory is not deleted
	defer os.RemoveAll(cniPath)

	// Create temp CNI file
	cniFile, err := ioutil.TempFile(cniPath, "cni.exe")
	require.NoError(t, err, "error creating CNI file")

	// Create temp CNI config dir
	cniConfigPath, err := ioutil.TempDir(cniPath, "cni")
	require.NoError(t, err, "error creating temp CNI config directory")

	// Create temp CNI config file
	cniConfig, err := ioutil.TempFile(cniConfigPath, "cni.conf")
	require.NoError(t, err, "error creating CNI config")

	wnb := winNodeBootstrapper{
		installDir:           installDir,
		cniPath:              cniPath,
		cniConfig:            cniConfig.Name(),
		cniInstallDir:        filepath.Join(installDir, cniDirName),
		cniConfigInstallPath: filepath.Join(installDir, cniConfigDirName),
	}

	err = wnb.ensureCNIDirIsPresent()
	require.NoError(t, err, "error creating CNI config directory %s", cniDirName)
	require.DirExists(t, filepath.Join(installDir, "cni", "config"), "CNI directory was not created")

	err = wnb.copyCNIFiles()
	assert.NoError(t, err, "unexpected error")
	assert.FileExists(t, filepath.Join(installDir, "cni", filepath.Base(cniFile.Name())), "CNI file was not copied")
	assert.FileExists(t, filepath.Join(installDir, "cni", "config", filepath.Base(cniConfig.Name())),
		"CNI config file was not copied")
}

// TestUpdateKubeletArgsForCNI tests if updateKubeletArgsForCNI() updates the kubelet args correctly
func TestUpdateKubeletArgsForCNI(t *testing.T) {
	// kubelet command without CNI parameters
	kubeletCmd := "c:\\k\\kubelet.exe --config=c:\\k\\kubelet.conf --bootstrap-kubeconfig=c:\\k\\bootstrap-kubeconfig " +
		"--kubeconfig=c:\\k\\kubeconfig --pod-infra-container-image=mcr.microsoft.com/k8s/core/pause:1.2.0 " +
		"--cert-dir=c:/var/lib/kubelet/pki/ --windows-service --logtostderr=false --log-file=c:\\k\\kubelet.log " +
		"--register-with-taints=os=Windows:NoSchedule --cloud-provider=aws --v=3"

	installDir := "C:\\k"
	cniInstallDir := filepath.Join(installDir, "cni")
	cniConfigInstallPath := filepath.Join(cniInstallDir, "config")
	wnb := winNodeBootstrapper{
		installDir:           installDir,
		cniInstallDir:        cniInstallDir,
		cniConfigInstallPath: cniConfigInstallPath,
	}

	wnb.updateKubeletArgsForCNI(&kubeletCmd)

	// Assert that the CNI parameters were added correctly
	assert.Contains(t, kubeletCmd, "--resolv-conf=\"\"", "--resolv-conf missing in kubelet args")
	assert.Contains(t, kubeletCmd, "--network-plugin=cni", "--network-plugin missing in kubelet args")
	assert.Contains(t, kubeletCmd, "--cni-bin-dir="+cniInstallDir, "--cni-bin-dir missing in kubelet args")
	assert.Contains(t, kubeletCmd, "--cni-conf-dir="+cniConfigInstallPath, "--cni-conf-dir missing in kubelet args")
	assert.NotContains(t, kubeletCmd, "--cni-conf-dir="+cniConfigInstallPath+"cni.conf", "cni.conf present in kubelet args")

	// kubelet command with CNI parameters set to different values
	kubeletCmd = "c:\\k\\kubelet.exe --config=c:\\k\\kubelet.conf --bootstrap-kubeconfig=c:\\k\\bootstrap-kubeconfig " +
		"--kubeconfig=c:\\k\\kubeconfig --pod-infra-container-image=mcr.microsoft.com/k8s/core/pause:1.2.0 " +
		"--cert-dir=c:/var/lib/kubelet/pki/ --windows-service --logtostderr=false --log-file=c:\\k\\kubelet.log " +
		"--register-with-taints=os=Windows:NoSchedule --cloud-provider=aws --v=3 --resolv-conf=d:\\k\\etc\\resolv.conf" +
		"--network-plugin=xyz --cni-bin-dir=d:\\k\\cni --cni-conf-dir=d:\\k\\cni\\config\\cni.conf"

	wnb.updateKubeletArgsForCNI(&kubeletCmd)

	// Assert that the CNI parameters were restored correctly
	assert.Contains(t, kubeletCmd, "--resolv-conf=\"\"", "--resolv-conf missing in kubelet args")
	assert.Contains(t, kubeletCmd, "--network-plugin=cni", "--network-plugin missing in kubelet args")
	assert.Contains(t, kubeletCmd, "--cni-bin-dir="+cniInstallDir, "--cni-bin-dir missing in kubelet args")
	assert.Contains(t, kubeletCmd, "--cni-conf-dir="+cniConfigInstallPath, "--cni-conf-dir missing in kubelet args")
	assert.NotContains(t, kubeletCmd, "--cni-conf-dir="+cniConfigInstallPath+"cni.conf", "cni.conf present in kubelet args")
}
