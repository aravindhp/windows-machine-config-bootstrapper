package e2e

import (
	"os"
	"testing"
	"time"

	"github.com/openshift/windows-machine-config-bootstrapper/tools/windows-node-installer/pkg/types"
	"github.com/stretchr/testify/assert"
)

const (
	// sshPort is the SSH port
	sshPort = "22"
)

var (
	// windowsVM provides the Windows VM object with appropriate methods to interact with it
	windowsVM types.WindowsVM
	// retryCount is the number of times we hit a request.
	retryCount = 30
	// retryInterval is interval of time until we retry after a failure.
	retryInterval = 5 * time.Second

	kubeconfig     = os.Getenv("KUBECONFIG")
	artifactDir    = os.Getenv("ARTIFACT_DIR")
	privateKeyPath = os.Getenv("KUBE_SSH_KEY_PATH")

	// imageID is the image that will be fed to the WNI for the tests. This is being set to empty, as we wish for it
	// to use the latest Windows image
	imageID = ""
	sshKey  = "libra"
)

// testInstanceFirewallRule checks if the created instance has opened container logs port via firewall
func testInstanceFirewallRule(t *testing.T) {
	// winFirewallCmd will verify if firewall rule is present on the windows node
	winFirewallCmd := "Get-NetFirewallRule -DisplayName ContainerLogsPort"
	var err error
	for i := 0; i < retryCount; i++ {
		_, err = windowsVM.RunOverSSH(winFirewallCmd, true)
		time.Sleep(retryInterval)
		if err == nil {
			break
		}
	}
	assert.NoError(t, err, "ContainerLogsPort firewall rule not present on the windows node")
}
