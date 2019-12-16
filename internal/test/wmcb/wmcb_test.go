package wmcb

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pkg/sftp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// remotePowerShellCmdPrefix holds the powershell prefix that needs to be prefixed to every command run on the
	// remote powershell session opened
	remotePowerShellCmdPrefix = "powershell.exe -NonInteractive -ExecutionPolicy Bypass "
	// nodeLabels represents the node label that need to be applied to the Windows node created
	nodeLabel          = "node.openshift.io/os_id=Windows"
	wmcbUnitTestBinary = "wmcb_unit_test.exe"
	wmcbE2ETestBinary  = "wmcb_e2e_test.exe"
)

var (
	// windowsTaint is the taint that needs to be applied to the Windows node
	windowsTaint = v1.Taint{
		Key:    "os",
		Value:  "Windows",
		Effect: v1.TaintEffectNoSchedule,
	}
	// binaryToBeTransferred holds the binary that needs to be transferred to the Windows VM
	// TODO: Make this an array later with a comma separated values for more binaries to be transferred
	binaryToBeTransferred = flag.String("binaryToBeTransferred", "",
		"Absolute path of the binary to be transferred")
)

//copyTestBinaryToWindowsVM copies the test binary to the Windows VM created as part of the test framework
func copyTestBinaryToWindowsVM(filename string) error {
	sftp, err := sftp.NewClient(framework.SSHClient)
	if err != nil {
		return fmt.Errorf("sftp client initialization failed: %v", err)
	}
	defer sftp.Close()

	f, err := os.Open(*binaryToBeTransferred)
	if err != nil {
		return fmt.Errorf("unable to open binary file to be transferred: %v", err)
	}

	dstFile, err := sftp.Create(filepath.Join(framework.RemoteDir, filename))
	if err != nil {
		return fmt.Errorf("unable to create remote file: %v", err)
	}

	_, err = io.Copy(dstFile, f)
	if err != nil {
		return fmt.Errorf("unable to copy binary to the Windows VM: %v", err)
	}

	// Forcefully close it so that we can execute the binary later
	err = dstFile.Close()
	if err != nil {
		log.Printf("error closing %s: %v", dstFile.Name(), err)
	}

	return nil
}

// remoteExecuteTestBinary executes the test binary remotely on the Windows VM created as part of the test framework
func remoteExecuteTestBinary(filename string) error {
	stdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("unable to open pipe to read stdout: %v", err)
	}
	os.Stdout = w

	// Remotely execute the test binary.
	_, err = framework.WinrmClient.Run(remotePowerShellCmdPrefix+filepath.Join(framework.RemoteDir,
		wmcbUnitTestBinary)+" --test.v", os.Stdout, os.Stderr)
	if err != nil {
		return fmt.Errorf("unable to execute the test binary remotely: %v", err)
	}
	w.Close()

	out, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("unable to read stdout from the remote Windows VM: %v", err)
	}

	os.Stdout = stdout

	// Log the test output
	log.Printf("%s", out)

	if strings.Contains(string(out), "FAIL") {
		return fmt.Errorf("%s remote test failure", filename)
	}

	if !strings.Contains(string(out), "PASS") {
		return fmt.Errorf("%s remote test failure", filename)
	}
	return nil
}

// TestWMCBUnit runs the unit tests for WMCB
func TestWMCBUnit(t *testing.T) {
	// Transfer the binary to the windows using scp
	err := copyTestBinaryToWindowsVM(wmcbUnitTestBinary)
	require.NoErrorf(t, err, "error copying %s to Windows VM", wmcbUnitTestBinary)

	err = remoteExecuteTestBinary(wmcbUnitTestBinary)
	assert.NoError(t, err, "unit test failure")
}

// hasWindowsTaint returns true if the given Windows node has the Windows taint
func hasWindowsTaint(winNodes []v1.Node) bool {
	// We've just created one Windows node as part of our CI suite. So, it's ok to return instead of checking for all
	// the items in the node
	for _, node := range winNodes {
		for _, taint := range node.Spec.Taints {
			if taint.Key == windowsTaint.Key && taint.Value == windowsTaint.Value && taint.Effect == windowsTaint.Effect {
				return true
			}
		}
	}
	return false
}

// TestWMCBCluster runs the cluster tests for the nodes
func TestWMCBCluster(t *testing.T) {
	//TODO: Transfer the WMCB binary to the Windows node and approve CSR for the Windows node.
	// I want this to be moved to another test. We've another card for this, so let's come back
	// to that later(WINC-82). As of now, this test is limited to check if the taint has been
	// applied to the Windows node and skipped for now.
	client := framework.K8sclientset
	winNodes, err := client.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: "kubernetes.io/os=windows"})
	require.NoErrorf(t, err, "error while getting Windows node: %v", err)
	assert.Equal(t, hasWindowsTaint(winNodes.Items), true, "expected Windows Taint to be present on the Windows Node")
	winNodes, err = client.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: nodeLabel})
	require.NoErrorf(t, err, "error while getting Windows node: %v", err)
	assert.Lenf(t, winNodes.Items, 1, "expected one node to have node label but found: %v", len(winNodes.Items))
}
