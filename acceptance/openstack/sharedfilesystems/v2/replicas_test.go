//go:build acceptance
// +build acceptance

package v2

import (
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/acceptance/clients"
	"github.com/gophercloud/gophercloud/acceptance/tools"
	"github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/replicas"
	th "github.com/gophercloud/gophercloud/testhelper"
)

// 2.56 is required for a /v2/replicas/XXX URL support
// otherwise we need to set "X-OpenStack-Manila-API-Experimental: true"
const replicasPathMicroversion = "2.56"

func TestReplicaCreate(t *testing.T) {
	clients.RequireManilaReplicas(t)

	client, err := clients.NewSharedFileSystemV2Client()
	if err != nil {
		t.Fatalf("Unable to create a shared file system client: %v", err)
	}
	client.Microversion = replicasPathMicroversion

	share, err := CreateShare(t, client)
	if err != nil {
		t.Fatalf("Unable to create a share: %v", err)
	}

	defer DeleteShare(t, client, share)

	replica, err := CreateReplica(t, client, share)
	if err != nil {
		t.Fatalf("Unable to create a replica: %v", err)
	}

	defer DeleteReplica(t, client, replica)

	created, err := replicas.Get(client, replica.ID).Extract()
	if err != nil {
		t.Errorf("Unable to retrieve replica: %v", err)
	}
	tools.PrintResource(t, created)

	allReplicas, err := ListShareReplicas(t, client, share.ID)
	th.AssertNoErr(t, err)

	if len(allReplicas) != 2 {
		t.Errorf("Unable to list all two replicas")
	}
}

func TestReplicaPromote(t *testing.T) {
	clients.RequireManilaReplicas(t)

	client, err := clients.NewSharedFileSystemV2Client()
	if err != nil {
		t.Fatalf("Unable to create a shared file system client: %v", err)
	}
	client.Microversion = replicasPathMicroversion

	share, err := CreateShare(t, client)
	if err != nil {
		t.Fatalf("Unable to create a share: %v", err)
	}

	defer DeleteShare(t, client, share)

	replica, err := CreateReplica(t, client, share)
	if err != nil {
		t.Fatalf("Unable to create a replica: %v", err)
	}

	defer DeleteReplica(t, client, replica)

	created, err := replicas.Get(client, replica.ID).Extract()
	if err != nil {
		t.Fatalf("Unable to retrieve replica: %v", err)
	}
	tools.PrintResource(t, created)

	// sync new replica
	err = replicas.Resync(client, created.ID).ExtractErr()
	th.AssertNoErr(t, err)
	_, err = waitForReplicaState(t, client, created.ID, "in_sync")
	if err != nil {
		t.Fatalf("Replica status error: %v", err)
	}

	// promote new replica
	err = replicas.Promote(client, created.ID, &replicas.PromoteOpts{}).ExtractErr()
	th.AssertNoErr(t, err)

	_, err = waitForReplicaState(t, client, created.ID, "active")
	if err != nil {
		t.Fatalf("Replica status error: %v", err)
	}

	// promote old replica
	allReplicas, err := ListShareReplicas(t, client, share.ID)
	th.AssertNoErr(t, err)
	var oldReplicaID string
	for _, v := range allReplicas {
		if v.ID == created.ID {
			// These are not the droids you are looking for
			continue
		}
		oldReplicaID = v.ID
	}
	if oldReplicaID == "" {
		t.Errorf("Unable to get old replica")
	}
	// sync old replica
	err = replicas.Resync(client, oldReplicaID).ExtractErr()
	th.AssertNoErr(t, err)
	_, err = waitForReplicaState(t, client, oldReplicaID, "in_sync")
	if err != nil {
		t.Fatalf("Replica status error: %v", err)
	}
	err = replicas.Promote(client, oldReplicaID, &replicas.PromoteOpts{}).ExtractErr()
	th.AssertNoErr(t, err)

	_, err = waitForReplicaState(t, client, oldReplicaID, "active")
	if err != nil {
		t.Fatalf("Replica status error: %v", err)
	}
}

func TestReplicaExportLocations(t *testing.T) {
	clients.RequireManilaReplicas(t)

	client, err := clients.NewSharedFileSystemV2Client()
	if err != nil {
		t.Fatalf("Unable to create a shared file system client: %v", err)
	}
	client.Microversion = replicasPathMicroversion

	share, err := CreateShare(t, client)
	if err != nil {
		t.Fatalf("Unable to create a share: %v", err)
	}

	defer DeleteShare(t, client, share)

	replica, err := CreateReplica(t, client, share)
	if err != nil {
		t.Fatalf("Unable to create a replica: %v", err)
	}

	defer DeleteReplica(t, client, replica)

	// this call should return empty list, since replica is not yet active
	exportLocations, err := replicas.ListExportLocations(client, replica.ID).Extract()
	if err != nil {
		t.Errorf("Unable to list replica export locations: %v", err)
	}
	tools.PrintResource(t, exportLocations)

	opts := replicas.ListOpts{
		ShareID: share.ID,
	}
	pages, err := replicas.List(client, opts).AllPages()
	th.AssertNoErr(t, err)

	allReplicas, err := replicas.ExtractReplicas(pages)
	th.AssertNoErr(t, err)

	var activeReplicaID string
	for _, v := range allReplicas {
		if v.State == "active" && v.Status == "available" {
			activeReplicaID = v.ID
		}
	}

	if activeReplicaID == "" {
		t.Errorf("Unable to get active replica")
	}

	exportLocations, err = replicas.ListExportLocations(client, activeReplicaID).Extract()
	if err != nil {
		t.Errorf("Unable to list replica export locations: %v", err)
	}
	tools.PrintResource(t, exportLocations)

	exportLocation, err := replicas.GetExportLocation(client, activeReplicaID, exportLocations[0].ID).Extract()
	if err != nil {
		t.Errorf("Unable to get replica export location: %v", err)
	}
	tools.PrintResource(t, exportLocation)
	// unset CreatedAt and UpdatedAt
	exportLocation.CreatedAt = time.Time{}
	exportLocation.UpdatedAt = time.Time{}
	th.AssertEquals(t, exportLocations[0], *exportLocation)
}

func TestReplicaListDetail(t *testing.T) {
	clients.RequireManilaReplicas(t)

	client, err := clients.NewSharedFileSystemV2Client()
	if err != nil {
		t.Fatalf("Unable to create a shared file system client: %v", err)
	}
	client.Microversion = replicasPathMicroversion

	share, err := CreateShare(t, client)
	if err != nil {
		t.Fatalf("Unable to create a share: %v", err)
	}

	defer DeleteShare(t, client, share)

	replica, err := CreateReplica(t, client, share)
	if err != nil {
		t.Fatalf("Unable to create a replica: %v", err)
	}

	defer DeleteReplica(t, client, replica)

	ss, err := ListShareReplicas(t, client, share.ID)
	if err != nil {
		t.Fatalf("Unable to list replicas: %v", err)
	}

	for i := range ss {
		tools.PrintResource(t, &ss[i])
	}
}

func TestReplicaResetStatus(t *testing.T) {
	clients.RequireManilaReplicas(t)

	client, err := clients.NewSharedFileSystemV2Client()
	if err != nil {
		t.Fatalf("Unable to create a shared file system client: %v", err)
	}
	client.Microversion = replicasPathMicroversion

	share, err := CreateShare(t, client)
	if err != nil {
		t.Fatalf("Unable to create a share: %v", err)
	}

	defer DeleteShare(t, client, share)

	replica, err := CreateReplica(t, client, share)
	if err != nil {
		t.Fatalf("Unable to create a replica: %v", err)
	}

	defer DeleteReplica(t, client, replica)

	resetStatusOpts := &replicas.ResetStatusOpts{
		Status: "error",
	}
	err = replicas.ResetStatus(client, replica.ID, resetStatusOpts).ExtractErr()
	if err != nil {
		t.Fatalf("Unable to reset a replica status: %v", err)
	}

	// We need to wait till the Extend operation is done
	_, err = waitForReplicaStatus(t, client, replica.ID, "error")
	if err != nil {
		t.Fatalf("Replica status error: %v", err)
	}

	t.Logf("Replica %s status successfuly reset", replica.ID)
}

// This test available only for cloud admins
func TestReplicaForceDelete(t *testing.T) {
	clients.RequireManilaReplicas(t)
	clients.RequireAdmin(t)

	client, err := clients.NewSharedFileSystemV2Client()
	if err != nil {
		t.Fatalf("Unable to create a shared file system client: %v", err)
	}
	client.Microversion = replicasPathMicroversion

	share, err := CreateShare(t, client)
	if err != nil {
		t.Fatalf("Unable to create a share: %v", err)
	}

	defer DeleteShare(t, client, share)

	replica, err := CreateReplica(t, client, share)
	if err != nil {
		t.Fatalf("Unable to create a replica: %v", err)
	}

	defer DeleteReplica(t, client, replica)

	err = replicas.ForceDelete(client, replica.ID).ExtractErr()
	if err != nil {
		t.Fatalf("Unable to force delete a replica: %v", err)
	}

	_, err = waitForReplicaStatus(t, client, replica.ID, "deleted")
	if err != nil {
		t.Fatalf("Replica status error: %v", err)
	}

	t.Logf("Replica %s was successfuly deleted", replica.ID)
}
