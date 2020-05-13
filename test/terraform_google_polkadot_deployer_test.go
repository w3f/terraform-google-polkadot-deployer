package test

import (
    "io/ioutil"
    "os"
    "testing"
    "time"
    "fmt"
    "strings"

    "github.com/gruntwork-io/terratest/modules/k8s"
    "github.com/gruntwork-io/terratest/modules/random"
    "github.com/gruntwork-io/terratest/modules/terraform"
    "github.com/gruntwork-io/terratest/modules/test-structure"
    "github.com/stretchr/testify/assert"
)

func TestTerraformGooglePolkadotDeployer(t *testing.T) {
    t.Parallel()

    terraformDir := "../"

    // At the end of the test, run `terraform destroy` to clean up any resources that were created
    defer test_structure.RunTestStage(t, "teardown", func() {
        terraformOptions := test_structure.LoadTerraformOptions(t, terraformDir)
        terraform.Destroy(t, terraformOptions)
    })

    // Deploy infrastructure
    test_structure.RunTestStage(t, "setup", func() {
        terraformOptions := createTerraformOptions(t, terraformDir)
        test_structure.SaveTerraformOptions(t, terraformDir, terraformOptions)
        terraform.InitAndApply(t, terraformOptions)
    })

    // Validate Cluster Size
    test_structure.RunTestStage(t, "validate_node_count", func() {
        terraformOptions := test_structure.LoadTerraformOptions(t, terraformDir)
        testNodeCount(t, terraformOptions)
    })
}

func createTerraformOptions(t *testing.T, terraformDir string) (*terraform.Options) {

    // A unique ID we can use to namespace resources so we don't clash with anything already in the AWS account or
    // tests running in parallel
    uniqueID := strings.ToLower(random.UniqueId())

    // Set up expected values to be checked later
    nodeCount := 2
    clusterName := fmt.Sprintf("terratest-polkadot-deployer-%s", uniqueID)
    gcpProjectId := getEnv("GCP_PROJECT_ID", "development-252112")

    terraformOptions := &terraform.Options{
        TerraformDir: terraformDir,
        Vars: map[string]interface{}{
            "cluster_name":   clusterName,
            "gcp_project_id": gcpProjectId,
            "location":       "europe-west4-b",
            "machine_type":   "n1-standard-1",
            "node_count":     nodeCount,
        },
        NoColor: true,
    }

    return terraformOptions
}

func createTempFile(t *testing.T, content []byte) (f *os.File){
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

func getEnv(key, fallback string) string {
    if value, ok := os.LookupEnv(key); ok {
        return value
    }
    return fallback
}

func testNodeCount(t *testing.T, terraformOptions *terraform.Options) {
    // Setup the kubectl config and context
    kubeconfig := terraform.Output(t, terraformOptions, "kubeconfig")
    kubeconfigFile := createTempFile(t, []byte(kubeconfig))
    defer os.Remove(kubeconfigFile.Name())
    options := k8s.NewKubectlOptions("", kubeconfigFile.Name(), "default")

    // Test that the Node count matches the Terraform specification
    k8s.WaitUntilAllNodesReady(t, options, 40, 10*time.Second)
    nodes := k8s.GetNodes(t, options)
    assert.Equal(t, len(nodes), int(terraformOptions.Vars["node_count"].(float64)))
}
