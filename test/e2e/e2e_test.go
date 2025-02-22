package e2e_test

import (
	"cni-benchmark/test/utils"
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E")
}

var (
	binaryPath string
	testenv    *envtest.Environment
	clientSet  *kubernetes.Clientset
)

var _ = BeforeSuite(func() {
	// Set up logging
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	// Get the current working directory
	cwd, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())

	// Determine the main.go location relative to the test file
	mainPath := filepath.Join(cwd, "..", "..", "cmd", "main.go")
	_, err = os.Stat(mainPath)
	Expect(err).NotTo(HaveOccurred(), "main.go not found at %s", mainPath)

	// Create temporary directory for the binary
	tmpDir, err := os.MkdirTemp("", "e2e-test-*")
	Expect(err).NotTo(HaveOccurred())

	// Set binary name based on OS
	binaryName := "cni-benchmark"
	if os.Getenv("GOOS") == "windows" {
		binaryName += ".exe"
	}
	binaryPath = filepath.Join(tmpDir, binaryName)

	// Build the binary
	cmd := exec.Command("go", "build", "-o", binaryPath, mainPath)
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	Expect(cmd.Run()).To(Succeed())

	// Create and start envtest
	testenv = &envtest.Environment{AttachControlPlaneOutput: false}
	restConfig, err := testenv.Start()
	Expect(err).NotTo(HaveOccurred())

	// Create clientSet for in-test access
	clientSet, err = kubernetes.NewForConfig(restConfig)
	Expect(err).NotTo(HaveOccurred())

	// Write kubeconfig to a temporary file
	kubeconfigFile, err := os.CreateTemp(tmpDir, "kubeconfig-*.yaml")
	Expect(err).NotTo(HaveOccurred())
	defer kubeconfigFile.Close()

	kubeconfigBytes, err := utils.CreateKubeconfigFileForRestConfig(restConfig)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	err = os.WriteFile(kubeconfigFile.Name(), kubeconfigBytes, 0o600)
	Expect(err).NotTo(HaveOccurred())

	// Set KUBECONFIG environment variable
	err = os.Setenv("KUBECONFIG", kubeconfigFile.Name())
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	// Clean up binary and its directory
	if binaryPath != "" {
		Expect(os.RemoveAll(filepath.Dir(binaryPath))).To(Succeed())
	}

	// Stop test environment
	if testenv != nil {
		Expect(testenv.Stop()).To(Succeed())
	}
})

// getFreePort finds first free port for server to listen ensuring no conflicts
func getFreePort() (port uint16, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return uint16(l.Addr().(*net.TCPAddr).Port), nil
		}
	}
	return
}

// createConfigMaps creates info configmaps
func createConfigMaps() (err error) {
	os := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "os-info",
			Namespace: "default",
		},
		Data: map[string]string{
			"OS_NAME":    "e2e-tests",
			"OS_VERSION": runtime.Version(),
		},
	}
	_, err = clientSet.CoreV1().ConfigMaps("default").Create(context.Background(), os, metav1.CreateOptions{})
	if err != nil {
		return
	}

	k8s := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8s-info",
			Namespace: "default",
		},
		Data: map[string]string{
			"K8S_PROVIDER":         "e2e-tests",
			"K8S_PROVIDER_VERSION": runtime.Version(),
			"K8S_VERSION":          runtime.Version(),
			"K8S_SERVICE_HOSTNAME": "kubernetes.default",
			"K8S_SERVICE_PORT":     "443",
			"K8S_POD_CIDR_IPV4":    "10.244.0.0/16",
		},
	}
	_, err = clientSet.CoreV1().ConfigMaps("default").Create(context.Background(), k8s, metav1.CreateOptions{})
	if err != nil {
		return
	}
	cni := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cni-info",
			Namespace: "default",
		},
		Data: map[string]string{
			"CNI_NAME":        "e2e-tests",
			"CNI_VERSION":     runtime.Version(),
			"CNI_DESCRIPTION": "e2e-tests",
		},
	}
	_, err = clientSet.CoreV1().ConfigMaps("default").Create(context.Background(), cni, metav1.CreateOptions{})
	return
}

// deleteConfigMaps deletes previously created configmaps
func deleteConfigMaps() (err error) {
	for _, name := range []string{"os-info", "k8s-info", "cni-info"} {
		err = clientSet.CoreV1().ConfigMaps("default").Delete(context.Background(), name, metav1.DeleteOptions{})
		if err != nil {
			return
		}
	}
	return
}

var _ = Describe("Server-Client", func() {
	var port uint16

	BeforeEach(func() {
		var err error
		port, err = getFreePort()
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		Expect(createConfigMaps()).To(Succeed())
	})

	AfterEach(func() {
		Expect(deleteConfigMaps()).To(Succeed())
	})

	It("should run server and client successfully", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start server process
		serverCmd := exec.CommandContext(ctx, binaryPath)
		serverCmd.Stdout = GinkgoWriter
		serverCmd.Stderr = GinkgoWriter
		serverCmd.Env = append(os.Environ(),
			"MODE=server",
			"PORT="+strconv.Itoa(int(port)),
		)
		Expect(serverCmd.Start()).To(Succeed())

		// Ensure server cleanup
		defer func() {
			if serverCmd.Process != nil {
				serverCmd.Process.Kill()
			}
		}()

		// Run client process
		clientCmd := exec.Command(binaryPath)
		clientCmd.Stdout = GinkgoWriter
		clientCmd.Stderr = GinkgoWriter
		clientCmd.Env = append(os.Environ(),
			"MODE=client",
			"SERVER=localhost",
			"PORT="+strconv.Itoa(int(port)),
			"DURATION=5",
			"DATABASE_URL=sqlite://:memory:?cache=shared",
			"TEST_CASE=e2e-tests",
		)

		// Run client and wait for completion
		Expect(clientCmd.Run()).To(Succeed())
		// Run client and handle error
		Expect(clientCmd.ProcessState.ExitCode()).To(Equal(0))
	})
})
