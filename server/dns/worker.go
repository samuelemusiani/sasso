package dns

import (
	"context"
	"time"
)

var (
	workerContext    context.Context    = nil
	workerCancelFunc context.CancelFunc = nil
	workerReturnChan chan error         = make(chan error, 1)
)

func StartWorker() {
	workerContext, workerCancelFunc = context.WithCancel(context.Background())
	go func() {
		workerReturnChan <- worker(workerContext)
		close(workerReturnChan)
	}()
}

func ShutdownWorker() error {
	if workerCancelFunc != nil {
		workerCancelFunc()
	}
	var err error = nil
	if workerReturnChan != nil {
		err = <-workerReturnChan
	}
	if err != nil && err != context.Canceled {
		return err
	} else {
		return nil
	}
}

func worker(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Second):
		// Just a small delay to let other components start
	}

	logger.Info("Proxmox worker started")

	timeToWait := 10 * time.Second

	for {
		// Handle graceful shutdown at the start of each cycle
		select {
		case <-ctx.Done():
			logger.Info("Proxmox worker shutting down")
			return ctx.Err()
		case <-time.After(timeToWait):
		}

		now := time.Now()

		// DTODO: This is the main loop that it's executed periodically
		//
		// This loop should check that records and views are present in the DNS.
		// Then delete stale records and views and add missing ones.
		//
		// Records are always of type A (we don't support IPv6 for now).
		//
		// The IPs are on the interface, but the name of the record is the VM
		// name. To retrieve the list of VMs and their IPs, we need a special DB
		// query with a JOIN.
		//
		// A VM could have multiple interfaces, so we need to take the primary one
		// only (the one with the gateway).
		//
		// A view is per VNet. So for the ACLs we must check the VNet subnet and
		// to add a record in the correct view we must check the VNet of the interface.
		//
		// A view is also per User. In the user view all the records of all his VMs
		// must be present. (It's like a sum of all the other views for that user).
		//
		// GROUPS:
		// A view per VNet is still created, and all the Group VMs are added there.
		//
		// For all the members of the group, their user view must also contain the Group VMs.
		// To distinguish Group VMs in the user view, we can add them to a subdomain.
		// For example if a normal VM is "vm1.sasso", a Group VM in the group "devs"
		// should be "vm2.devs.sasso". This is sufficient for the users only, but
		// because it could create some confusion we can also add these records in the
		// views of the Group. So in the Groups view we have "vm2.sasso" and
		// "vm2.devs.sasso". In the user views we have only "vm2.devs.sasso".

		elapsed := time.Since(now)
		if elapsed < 10*time.Second {
			timeToWait = 10*time.Second - elapsed
		} else {
			timeToWait = 0
		}
	}
}
