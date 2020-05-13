package test

import (
    "crypto/tls"
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "testing"
    "time"
    "strings"

    "github.com/gruntwork-io/terratest/modules/helm"
    "github.com/gruntwork-io/terratest/modules/k8s"
    "github.com/gruntwork-io/terratest/modules/random"
    "github.com/gruntwork-io/terratest/modules/terraform"
    "github.com/gruntwork-io/terratest/modules/test-structure"
    "github.com/gruntwork-io/terratest/modules/http-helper"

    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/assert"
)

func TestTerraformClusterCreation(t *testing.T) {
    t.Parallel()

    terraformDir := "../"
    nodePort := 30100
    // uniqueID := strings.ToLower(random.UniqueId())
    uniqueID := "ext1234"

    clusterName := fmt.Sprintf("terratest-polkadot-deployer-%s", uniqueID)
    gcpProjectId := getEnv("GCP_PROJECT_ID", "development-252112")
    terraformVars := map[string]interface{} {
        "cluster_name":   clusterName,
        "gcp_project_id": gcpProjectId,
        "location":       "europe-west4-b",
        "machine_type":   "n1-standard-1",
        "node_count":     2,
    }

    // At the end of the test, run `terraform destroy` to clean up any resources that were created
    defer test_structure.RunTestStage(t, "teardown", func() {
        destroyTerraformStack(t, terraformDir)
    })

    // Deploy infrastructure
    test_structure.RunTestStage(t, "setup", func() {
        createTerraformStack(t, terraformDir, terraformVars)
    })

    // Configure kubectl
    kubeconfigFile, kubectlOptions := setupKubeconfig(t, terraformDir)
    defer os.Remove(kubeconfigFile.Name())

    // Validate cluster size
    test_structure.RunTestStage(t, "validate_node_count", func() {
        testNodeCount(t, terraformDir, kubectlOptions)
    })

    // Validate external connectivity to the service
    test_structure.RunTestStage(t, "validate_service", func() {
        testServiceAvailability(t, kubectlOptions, nodePort)
    })
}

// Deploy terraform module to cloud provider
func createTerraformStack(t *testing.T, terraformDir string, terraformVars map[string]interface{}, ) {
    terraformOptions := &terraform.Options{
        TerraformDir: terraformDir,
        Vars: terraformVars,
        NoColor: true,
    }

    test_structure.SaveTerraformOptions(t, terraformDir, terraformOptions)
    terraform.InitAndApply(t, terraformOptions)
}

// Destroy previously created terraform stack
func destroyTerraformStack(t *testing.T, terraformDir string) {
    terraformOptions := test_structure.LoadTerraformOptions(t, terraformDir)
    terraform.Destroy(t, terraformOptions)
}

// Create temporary file
func createTempFile(t *testing.T, content []byte) *os.File {
    tempFile, err := ioutil.TempFile(os.TempDir(), random.UniqueId())
    if err != nil {
        t.Fatal("Cannot create temporary file", err)
    }

    if _, err = tempFile.Write(content); err != nil {
        t.Fatal("Failed to write to temporary file", err)
    }
    if err := tempFile.Close(); err != nil {
        t.Fatal(err)
    }

    return tempFile
}

// Write kubeconfig file from terraform output and configure to use it kubectl
func setupKubeconfig(t *testing.T, terraformDir string) (*os.File, *k8s.KubectlOptions) {
    terraformOptions := test_structure.LoadTerraformOptions(t, terraformDir)
    kubeconfig := terraform.Output(t, terraformOptions, "kubeconfig")
    kubeconfigFile := createTempFile(t, []byte(kubeconfig))
    kubectlOptions := k8s.NewKubectlOptions("", kubeconfigFile.Name(), "default")

    return kubeconfigFile, kubectlOptions
}

// Get environment variable with fallback value
func getEnv(key, fallback string) string {
    if value, ok := os.LookupEnv(key); ok {
        return value
    }
    return fallback
}

// Get k8s node's IP address of a giving type
func getNodeAddress(t *testing.T, kubectlOptions *k8s.KubectlOptions, addrType string) string {
    nodes := k8s.GetNodes(t, kubectlOptions)
    for _, addr := range nodes[0].Status.Addresses {
        if string(addr.Type) == addrType {
            return addr.Address
        }
    }
    t.Fatalf("%s address is not available", addrType)
    return ""
}

// Test that the Node count matches the Terraform specification
func testNodeCount(t *testing.T, terraformDir string, kubectlOptions *k8s.KubectlOptions) {
    terraformOptions := test_structure.LoadTerraformOptions(t, terraformDir)

    k8s.WaitUntilAllNodesReady(t, kubectlOptions, 40, 10*time.Second)
    nodes := k8s.GetNodes(t, kubectlOptions)
    assert.Equal(t, len(nodes), int(terraformOptions.Vars["node_count"].(float64)))
}

// Test service deployment and verify it's availability on configured nodePort
func testServiceAvailability(t *testing.T, kubectlOptions *k8s.KubectlOptions, nodePort int) {
    helmOptions := &helm.Options{
        KubectlOptions: kubectlOptions,
        SetValues: map[string]string{
            "image.repo": "nginx",
            "image.tag":  "1.8",
            "nodePort":   fmt.Sprintf("%d", nodePort),
        },
    }

    helmChartPath, err := filepath.Abs("fixtures/nginx-chart")
    require.NoError(t, err)

    helmReleaseName := fmt.Sprintf("polkadot-nginx-%s", strings.ToLower(random.UniqueId()))
    defer helm.Delete(t, helmOptions, helmReleaseName, true)
    helm.Install(t, helmOptions, helmChartPath, helmReleaseName)

    // Validate service availability
    k8s.WaitUntilServiceAvailable(t, kubectlOptions, helmReleaseName, 20, 5*time.Second)
    service := k8s.GetService(t, kubectlOptions, helmReleaseName)
    require.Equal(t, service.Name, helmReleaseName)

    // Check external connectivity
    tlsConfig := tls.Config{}
    url := fmt.Sprintf("http://%s:%d", getNodeAddress(t, kubectlOptions, "ExternalIP"), nodePort)
    http_helper.HttpGetWithRetryWithCustomValidation(
        t,
        url,
        &tlsConfig,
        30,
        5*time.Second,
        func(statusCode int, body string) bool {
            return statusCode == 200
        },
    )
}
