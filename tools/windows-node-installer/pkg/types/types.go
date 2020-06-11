package types

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// This package should have the types that will be used by component. For example, aws should have it's own
// sub-package
// TODO: Move every cloud provider types here

const (
	// ContainerLogsPort number will be opened on the windows node via firewall
	ContainerLogsPort = "10250"
	// FirewallRuleName is the firewall rule name to open the Container Logs Port
	FirewallRuleName = "ContainerLogsPort"
	// remotePowerShellCmdPrefix holds the PowerShell prefix that needs to be prefixed  for every remote PowerShell
	// command executed on the remote Windows VM
	remotePowerShellCmdPrefix = "powershell.exe -NonInteractive -ExecutionPolicy Bypass "
	// SshPort is the SSH port
	SshPort = 22
)

// Windows represents a Windows host.
// TODO: Add a struct called Connectivity which has information related to Winrm, SSH and have
//		getters and setters for it so that it can be exposed as a public method
//		jira: https://issues.redhat.com/browse/WINC-239. Remove the fields related to SSHClient and WinrmClient and put
// 		them in connectivity struct
type Windows struct {
	// Credentials to access the Windows VM created
	Credentials *Credentials
	// SSHClient contains the ssh client information to access the Windows VM via ssh
	SSHClient *ssh.Client
}

// WindowsVM is the interface for interacting with a Windows object created by the cloud provider
type WindowsVM interface {
	// CopyFile copies the given file to the remote directory in the Windows VM. The remote directory is created if it
	// does not exist
	CopyFile(string, string) error
	// Run executes the given command remotely on the Windows VM over a ssh connection and returns the combined output
	// of stdout and stderr. If the bool is set, it implies that the cmd is to be execute in PowerShell. This function
	// should be used in scenarios where you want to execute a command that runs in the background. In these cases we
	// have observed that Run() returns before the command completes and as a result killing the process.
	RunOverSSH(string, bool) (string, error)
	// GetCredentials returns the interface for accessing the VM credentials. It is up to the caller to check if non-nil
	// Credentials are returned before usage.
	GetCredentials() *Credentials
	// Reinitialize re-initializes the Windows VM. Presently only the ssh client is reinitialized.
	Reinitialize() error
}

func (w *Windows) CopyFile(filePath, remoteDir string) error {
	if w.SSHClient == nil {
		return fmt.Errorf("CopyFile cannot be called without a SSH client")
	}

	ftp, err := sftp.NewClient(w.SSHClient)
	if err != nil {
		return fmt.Errorf("sftp client initialization failed: %v", err)
	}
	defer ftp.Close()

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening %s file to be transferred: %v", filePath, err)
	}
	defer f.Close()

	if err = ftp.MkdirAll(remoteDir); err != nil {
		return fmt.Errorf("error creating remote directory %s: %v", remoteDir, err)
	}

	remoteFile := remoteDir + "\\" + filepath.Base(filePath)
	dstFile, err := ftp.Create(remoteFile)
	if err != nil {
		return fmt.Errorf("error initializing %s file on Windows VMs: %v", remoteFile, err)
	}

	_, err = io.Copy(dstFile, f)
	if err != nil {
		return fmt.Errorf("error copying %s to the Windows VM: %v", filePath, err)
	}

	// Forcefully close it so that we can execute the binary later
	dstFile.Close()
	return nil
}

func (w *Windows) RunOverSSH(cmd string, psCmd bool) (string, error) {
	if w.SSHClient == nil {
		return "", fmt.Errorf("RunOverSSH cannot be called without a ssh client")
	}

	session, err := w.SSHClient.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	if psCmd {
		cmd = remotePowerShellCmdPrefix + cmd
	}

	out, err := session.CombinedOutput(cmd)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (w *Windows) GetCredentials() *Credentials {
	return w.Credentials
}

func (w *Windows) Reinitialize() error {
	if err := w.GetSSHClient(); err != nil {
		return fmt.Errorf("failed to reinitialize ssh client: %v", err)
	}
	return nil
}

// GetSSHClient gets the ssh client associated with Windows VM created
func (w *Windows) GetSSHClient() error {
	if w.SSHClient != nil {
		// Close the existing client to be on the safe side
		if err := w.SSHClient.Close(); err != nil {
			log.Printf("warning - error closing ssh client connection: %v", err)
		}
	}

	config := &ssh.ClientConfig{
		User:            w.Credentials.GetUserName(), //TODO: Change this to make sure that this works for Azure.
		Auth:            []ssh.AuthMethod{ssh.Password(w.Credentials.GetPassword())},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	var sshClient *ssh.Client
	var err error
	retries := 10
	// Allow the VM to boot up and enable the SSH service
	for retry := 0; retry < retries; retry++ {
		time.Sleep(30 * time.Second)
		sshClient, err = ssh.Dial("tcp", w.Credentials.GetIPAddress()+":22", config)
		if err == nil {
			break
		}
		log.Printf("ssh dial error: %v. retrying %d/%d", err, retry, retries)
	}

	if err != nil {
		return fmt.Errorf("failed to setup ssh client: %s", err)
	}
	w.SSHClient = sshClient
	return nil
}

// Credentials holds the information to access the Windows instance created.
type Credentials struct {
	// instanceID uniquely identifies the instanceID
	instanceID string
	// ipAddress contains the public ip address of the instance created
	ipAddress string
	// password to access the instance created
	password string
	// user used for accessing the  instance created
	user string
}

// NewCredentials takes the instanceID, ip address, password and user of the Windows instance created and returns the
// Credentials structure
func NewCredentials(instanceID, ipAddress, password, user string) *Credentials {
	return &Credentials{instanceID: instanceID, ipAddress: ipAddress, password: password, user: user}
}

// GetIPAddress returns the ip address of the given node
func (cred *Credentials) GetIPAddress() string {
	return cred.ipAddress
}

// GetPassword returns the password associated with the given node
func (cred *Credentials) GetPassword() string {
	return cred.password
}

// GetInstanceID returns the instanceId associated with the given node
func (cred *Credentials) GetInstanceId() string {
	return cred.instanceID
}

// GetUserName returns the user name associated with the given node
func (cred *Credentials) GetUserName() string {
	return cred.user
}
