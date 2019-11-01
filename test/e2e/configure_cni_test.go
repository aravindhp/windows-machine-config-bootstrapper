package e2e

import (
	"testing"
	"time"

	"github.com/openshift/windows-machine-config-operator/pkg/bootstrapper"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var cniPath string
var cniConfig string

func init() {
	pflag.StringVar(&cniPath, "cni-path", "C:\\Windows\\Temp\\cni", "CNI binary location")
	pflag.StringVar(&cniConfig, "cni-config", "C:\\Windows\\Temp\\cni\\config\\cni.conf", "CNI config location")
}

func TestConfigureCNI(t *testing.T) {
	t.Run("Configure CNI without kubelet service", testConfigureCNIWithoutKubeletSvc)
}

// testConfigureCNIWithoutKubeletSvc tests if WMCB returns an error if CNI configuration is attempted without a kubelet
// service
func testConfigureCNIWithoutKubeletSvc(t *testing.T) {
	require.True(t, !svcExists(t, "kubelet"))

	// Instantiate the bootstrapper
	wmcb, err := bootstrapper.NewWinNodeBootstrapper("C:\\k", "", "", "", "")
	require.NoError(t, err, "could not instantiate wmcb")

	err = wmcb.ConfigureCNI()
	assert.Error(t, err, "no error when attempting to configure CNI without kubelet svc")
	assert.Contains(t, err.Error(), "kubelet service is not present", "incorrect error thrown")
}

// testConfigureCNI tests if ConfigureCNI() runs successfully by checking if the kubelet service comes up after
// configuring CNI
func testConfigureCNI(t *testing.T) {
	wmcb, err := bootstrapper.NewWinNodeBootstrapper(installDir, "", "", cniPath, cniConfig)
	require.NoError(t, err, "could not create wmcb")

	err = wmcb.ConfigureCNI()
	assert.NoError(t, err, "error running wmcb.ConfigureCNI")

	err = wmcb.Disconnect()
	assert.NoError(t, err, "could not disconnect from windows svc API")

	// Wait for the service to start
	time.Sleep(2 * time.Second)
	assert.Truef(t, svcRunning(t, bootstrapper.KubeletServiceName),
		"kubelet service is not running after configuring CNI")

	// Wait for kubelet log to be populated
	time.Sleep(5 * time.Second)
	assert.True(t, isKubeletRunning(t, kubeletLogPath))

}
