package azureassignedidentity

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/test/common/util"
	"github.com/pkg/errors"
)

// GetAll will return a list of AzureAssignedIdentity deployed on a Kubernetes cluster
func GetAll() (*aadpodid.AzureAssignedIdentityList, error) {
	cmd := exec.Command("kubectl", "get", "AzureAssignedIdentity", "-ojson")
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get AzureAssignedIdentity from the Kubernetes cluster")
	}

	list := aadpodid.AzureAssignedIdentityList{}
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshall json")
	}

	return &list, nil
}

// GetByPrefix will return an AzureAssignedIdentity with matched prefix
func GetByPrefix(prefix string) (aadpodid.AzureAssignedIdentity, error) {
	list, err := GetAll()
	if err != nil {
		return aadpodid.AzureAssignedIdentity{}, err
	}

	for _, azureAssignedIdentity := range list.Items {
		if strings.HasPrefix(azureAssignedIdentity.ObjectMeta.Name, prefix) {
			return azureAssignedIdentity, nil
		}
	}

	return aadpodid.AzureAssignedIdentity{}, errors.Errorf("No AzureAssignedIdentity has a name prefix with '%s'", prefix)
}

// WaitOnLengthMatched will block until the number of Azure Assigned Identity matches the target
func WaitOnLengthMatched(target int) (bool, error) {
	successChannel, errorChannel := make(chan bool, 1), make(chan error)
	duration, sleep := 70*time.Second, 10*time.Second
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	fmt.Println("# Tight-poll to check if the Azure Assigned Identity is deleted...")
	go func() {
		for {
			select {
			case <-ctx.Done():
				errorChannel <- errors.Errorf("Timeout exceeded (%s) while waiting for the number of AzureAssignedIdentity to be equal to %d", duration.String(), target)
			default:
				list, err := GetAll()
				if err != nil {
					errorChannel <- err
				}
				if len(list.Items) == target {
					successChannel <- true
					return
				}
				fmt.Printf("# The number of Azure Assigned Identity is not equal to %d yet. Retrying in %s...\n", target, sleep.String())
				time.Sleep(sleep)
			}
		}
	}()

	for {
		select {
		case err := <-errorChannel:
			return false, err
		case success := <-successChannel:
			return success, nil
		}
	}
}
