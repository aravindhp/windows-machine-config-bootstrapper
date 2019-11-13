package e2e

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/masterzen/winrm"
	"github.com/openshift/windows-machine-config-operator/tools/windows-node-installer/pkg/cloudprovider"
	"github.com/openshift/windows-machine-config-operator/tools/windows-node-installer/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// Get kubeconfig, AWS credentials, and artifact dir from environment variable set by the OpenShift CI operator.
	kubeconfig     = os.Getenv("KUBECONFIG")
	awsCredentials = os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
	dir            = os.Getenv("ARTIFACT_DIR")
	privateKeyPath = os.Getenv("KUBE_SSH_KEY_PATH")

	// Path of the WSU playbook
	playbookPath = os.Getenv("WSU_PATH")
	// clusterAddress is the address of the OpenShift cluster e.g. "foo.fah.com".
	// This should not include "https://api-" or a port
	clusterAddress = os.Getenv("CLUSTER_ADDR")

	// The CI-operator uses AWS region `us-east-1` which has the corresponding image ID: ami-0b8d82dea356226d3 for
	// Microsoft Windows Server 2019 Base with Containers.
	imageID      = "ami-0b8d82dea356226d3"
	instanceType = "m4.large"
	sshKey       = "libra"

	// Cloud provider factory that we will use in these tests
	cloud cloudprovider.Cloud
	// Credentials for a spun up instance
	createdInstanceCreds *types.Credentials
	// Temp directory ansible created on the windows host
	ansibleTempDir = ""
	// k8sclientset is the kubernetes clientset we will use to query the cluster's status
	k8sclientset *kubernetes.Clientset
)

// createAWSWindowsInstance creates a windows instance and populates the "cloud" and "createdInstanceCreds" global
// variables
func createAWSWindowsInstance() error {
	var err error
	cloud, err = cloudprovider.CloudProviderFactory(kubeconfig, awsCredentials, "default", dir,
		imageID, instanceType, sshKey, privateKeyPath)
	if err != nil {
		return fmt.Errorf("could not setup cloud provider: %s", err)
	}
	createdInstanceCreds, err = cloud.CreateWindowsVM()
	if err != nil {
		return fmt.Errorf("could not create windows VM: %s", err)
	}
	return nil
}

// createhostFile creates an ansible host file and returns the path of it
func createHostFile(ip, password string) (string, error) {
	hostFile, err := ioutil.TempFile("", "testWSU")
	if err != nil {
		return "", fmt.Errorf("coud not make temporary file: %s", err)
	}
	defer hostFile.Close()

	_, err = hostFile.WriteString(fmt.Sprintf(`[win]
%s ansible_password='%s'

[win:vars]
ansible_user=Administrator
cluster_address=%s
ansible_port=5986
ansible_connection=winrm
ansible_winrm_server_cert_validation=ignore`, ip, password, clusterAddress))
	return hostFile.Name(), err
}

// TestWSU creates a Windows instance, runs the WSU, and then runs a series of tests to ensure all expected
// behavior was achieved. The following environment variables must be set for this test to run: KUBECONFIG,
// AWS_SHARED_CREDENTIALS_FILE, ARTIFACT_DIR, KUBE_SSH_KEY_PATH, WSU_PATH, CLUSTER_ADDR
func TestWSU(t *testing.T) {
	require.NotEmptyf(t, kubeconfig, "KUBECONFIG environment variable not set")
	require.NotEmptyf(t, awsCredentials, "AWS_SHARED_CREDENTIALS_FILE environment variable not set")
	require.NotEmptyf(t, dir, "ARTIFACT_DIR environment variable not set")
	require.NotEmptyf(t, privateKeyPath, "KUBE_SSH_KEY_PATH environment variable not set")
	require.NotEmptyf(t, playbookPath, "WSU_PATH environment variable not set")
	require.NotEmptyf(t, clusterAddress, "CLUSTER_ADDR environment variable not set")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	require.NoError(t, err, "Could not build kubeconfig")
	k8sclientset, err = kubernetes.NewForConfig(config)
	require.NoError(t, err, "Could create k8s clientset")

	// TODO: Check if other cloud provider credentials are available
	if awsCredentials == "" {
		t.Fatal("No cloud provider credentials available")
	}
	err = createAWSWindowsInstance()
	require.NoErrorf(t, err, "Error spinning up Windows VM: %s", err)
	require.NotNil(t, createdInstanceCreds, "Instance credentials are not set")
	defer cloud.DestroyWindowsVMs()
	// In order to run the ansible playbook we create an inventory file:
	// https://docs.ansible.com/ansible/latest/user_guide/intro_inventory.html
	hostFilePath, err := createHostFile(createdInstanceCreds.GetIPAddress(), createdInstanceCreds.GetPassword())
	require.NoErrorf(t, err, "Could not write to host file: %s", err)
	cmd := exec.Command("ansible-playbook", "-vvv", "-i", hostFilePath, playbookPath)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "WSU playbook returned error: %s, with output: %s", err, string(out))

	// Ansible will copy files to a temporary directory with a path such as:
	// C:\\Users\\Administrator\\AppData\\Local\\Temp\\ansible.z5wa1pc5.vhn\\
	initialSplit := strings.Split(string(out), "C:\\\\Users\\\\Administrator\\\\AppData\\\\Local\\\\Temp\\\\ansible.")
	require.True(t, len(initialSplit) > 1, "Could not find Windows temp dir: %s", out)
	ansibleTempDir = "C:\\Users\\Administrator\\AppData\\Local\\Temp\\ansible." + strings.Split(initialSplit[1], "\"")[0]

	t.Run("Files copied to Windows node", testFilesCopied)
	t.Run("Node is in ready state", testNodeReady)
}

// testFilesCopied tests that the files we attempted to copy to the Windows host, exist on the Windows host
func testFilesCopied(t *testing.T) {
	expectedFileList := []string{"kubelet.exe", "worker.ign", "wmcb.exe"}
	endpoint := winrm.NewEndpoint(createdInstanceCreds.GetIPAddress(), 5986, true, true,
		nil, nil, nil, 0)
	client, err := winrm.NewClient(endpoint, "Administrator", createdInstanceCreds.GetPassword())
	require.NoErrorf(t, err, "Could not create winrm client: %s", err)

	// Check if each of the files we expect on the Windows host are there
	for _, filename := range expectedFileList {
		fullPath := ansibleTempDir + "\\" + filename
		// This command will write to stdout, only if the file we are looking for does not exist
		command := fmt.Sprintf("if not exist %s echo fail", fullPath)
		stdout := new(bytes.Buffer)
		_, err := client.Run(command, stdout, os.Stderr)
		assert.NoError(t, err, "Error looking for %s: %s", fullPath, err)
		assert.Emptyf(t, stdout.String(), "Missing file: %s", fullPath)
	}
}

// testNodeReady tests that the bootstrapped node was added to the cluster and is in the ready state
func testNodeReady(t *testing.T) {
	var createdNode *v1.Node
	nodes, err := k8sclientset.CoreV1().Nodes().List(metav1.ListOptions{})
	require.NoError(t, err, "Could not get list of nodes")
	require.NotZero(t, len(nodes.Items), "No nodes found")

	// Find the node that we spun up
	for _, node := range nodes.Items {
		for _, address := range node.Status.Addresses {
			if address.Type == "ExternalIP" && address.Address == createdInstanceCreds.GetIPAddress() {
				createdNode = &node
				break
			}
		}
		if createdNode != nil {
			break
		}
	}
	require.NotNil(t, createdNode, "Created node not found through Kubernetes API")

	// Make sure the node is in a ready state
	foundReady := false
	for _, condition := range createdNode.Status.Conditions {
		if condition.Type != v1.NodeReady {
			continue
		}
		foundReady = true
		assert.Equalf(t, v1.ConditionTrue, condition.Status, "Node not in ready state: %s", condition.Reason)
		break
	}
	// Just in case node is missing the ready condition, for whatever reason
	assert.True(t, foundReady, "Node did not have ready condition")
}
