//go:build acceptance || blockstorage
// +build acceptance blockstorage

package v3

import (
	"fmt"
	"testing"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/acceptance/clients"
	"github.com/gophercloud/gophercloud/acceptance/tools"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/pagination"
	th "github.com/gophercloud/gophercloud/testhelper"
)

func TestSnapshots(t *testing.T) {
	clients.RequireLong(t)

	client, err := clients.NewBlockStorageV3Client()
	th.AssertNoErr(t, err)

	volume1, err := CreateVolume(t, client)
	th.AssertNoErr(t, err)
	defer DeleteVolume(t, client, volume1)

	snapshot1, err := CreateSnapshot(t, client, volume1)
	th.AssertNoErr(t, err)
	defer DeleteSnapshot(t, client, snapshot1)

	// Update snapshot
	updatedSnapshotName := tools.RandomString("ACPTTEST", 16)
	updatedSnapshotDescription := tools.RandomString("ACPTTEST", 16)
	updateOpts := snapshots.UpdateOpts{
		Name:        &updatedSnapshotName,
		Description: &updatedSnapshotDescription,
	}
	t.Logf("Attempting to update snapshot: %s", updatedSnapshotName)
	updatedSnapshot, err := snapshots.Update(client, snapshot1.ID, updateOpts).Extract()
	th.AssertNoErr(t, err)

	tools.PrintResource(t, updatedSnapshot)
	th.AssertEquals(t, updatedSnapshot.Name, updatedSnapshotName)
	th.AssertEquals(t, updatedSnapshot.Description, updatedSnapshotDescription)

	volume2, err := CreateVolume(t, client)
	th.AssertNoErr(t, err)
	defer DeleteVolume(t, client, volume2)

	snapshot2, err := CreateSnapshot(t, client, volume2)
	th.AssertNoErr(t, err)
	defer DeleteSnapshot(t, client, snapshot2)

	listOpts := snapshots.ListOpts{
		Limit: 1,
	}

	err = snapshots.List(client, listOpts).EachPage(func(page pagination.Page) (bool, error) {
		actual, err := snapshots.ExtractSnapshots(page)
		th.AssertNoErr(t, err)
		th.AssertEquals(t, 1, len(actual))

		var found bool
		for _, v := range actual {
			if v.ID == snapshot1.ID || v.ID == snapshot2.ID {
				found = true
			}
		}

		th.AssertEquals(t, found, true)

		return true, nil
	})

	th.AssertNoErr(t, err)
}

func TestSnapshotsResetStatus(t *testing.T) {
	clients.RequireLong(t)

	client, err := clients.NewBlockStorageV3Client()
	th.AssertNoErr(t, err)

	volume1, err := CreateVolume(t, client)
	th.AssertNoErr(t, err)
	defer DeleteVolume(t, client, volume1)

	snapshot1, err := CreateSnapshot(t, client, volume1)
	th.AssertNoErr(t, err)
	defer DeleteSnapshot(t, client, snapshot1)

	// Reset snapshot status to error
	resetOpts := snapshots.ResetStatusOpts{
		Status: "error",
	}
	t.Logf("Attempting to reset snapshot status to %s", resetOpts.Status)
	err = snapshots.ResetStatus(client, snapshot1.ID, resetOpts).ExtractErr()
	th.AssertNoErr(t, err)

	snapshot, err := snapshots.Get(client, snapshot1.ID).Extract()
	th.AssertNoErr(t, err)

	if snapshot.Status != resetOpts.Status {
		th.AssertNoErr(t, fmt.Errorf("unexpected %q snapshot status", snapshot.Status))
	}

	// Reset snapshot status to available
	resetOpts = snapshots.ResetStatusOpts{
		Status: "available",
	}
	t.Logf("Attempting to reset snapshot status to %s", resetOpts.Status)
	err = snapshots.ResetStatus(client, snapshot1.ID, resetOpts).ExtractErr()
	th.AssertNoErr(t, err)

	snapshot, err = snapshots.Get(client, snapshot1.ID).Extract()
	th.AssertNoErr(t, err)

	if snapshot.Status != resetOpts.Status {
		th.AssertNoErr(t, fmt.Errorf("unexpected %q snapshot status", snapshot.Status))
	}
}

func TestSnapshotsUpdateStatus(t *testing.T) {
	clients.RequireLong(t)

	client, err := clients.NewBlockStorageV3Client()
	th.AssertNoErr(t, err)

	volume1, err := CreateVolume(t, client)
	th.AssertNoErr(t, err)
	defer DeleteVolume(t, client, volume1)

	snapshot1, err := CreateSnapshot(t, client, volume1)
	th.AssertNoErr(t, err)
	defer DeleteSnapshot(t, client, snapshot1)

	// Update snapshot status to error
	resetOpts := snapshots.ResetStatusOpts{
		Status: "creating",
	}
	t.Logf("Attempting to update snapshot status to %s", resetOpts.Status)
	err = snapshots.ResetStatus(client, snapshot1.ID, resetOpts).ExtractErr()
	th.AssertNoErr(t, err)

	snapshot, err := snapshots.Get(client, snapshot1.ID).Extract()
	th.AssertNoErr(t, err)

	if snapshot.Status != resetOpts.Status {
		th.AssertNoErr(t, fmt.Errorf("unexpected %q snapshot status", snapshot.Status))
	}

	// Update snapshot status to available
	updateOpts := snapshots.UpdateStatusOpts{
		Status: "available",
	}
	t.Logf("Attempting to update snapshot status to %s", updateOpts.Status)
	err = snapshots.UpdateStatus(client, snapshot1.ID, updateOpts).ExtractErr()
	th.AssertNoErr(t, err)

	snapshot, err = snapshots.Get(client, snapshot1.ID).Extract()
	th.AssertNoErr(t, err)

	if snapshot.Status != updateOpts.Status {
		th.AssertNoErr(t, fmt.Errorf("unexpected %q snapshot status", snapshot.Status))
	}
}

func TestSnapshotsForceDelete(t *testing.T) {
	clients.RequireLong(t)

	client, err := clients.NewBlockStorageV3Client()
	th.AssertNoErr(t, err)

	volume, err := CreateVolume(t, client)
	th.AssertNoErr(t, err)
	defer DeleteVolume(t, client, volume)

	snapshot, err := CreateSnapshot(t, client, volume)
	th.AssertNoErr(t, err)
	defer DeleteSnapshot(t, client, snapshot)

	// Force delete snapshot
	t.Logf("Attempting to force delete %s snapshot", snapshot.ID)
	err = snapshots.ForceDelete(client, snapshot.ID).ExtractErr()
	th.AssertNoErr(t, err)

	err = tools.WaitFor(func() (bool, error) {
		_, err := snapshots.Get(client, snapshot.ID).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				return true, nil
			}
		}

		return false, nil
	})
	th.AssertNoErr(t, err)
}
