<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-col gap-2">
      <h2 class="text-info font-bold">Introduction</h2>
      <p>
        A VM can be backed up at any time, and the backup can be restored later. This is useful for
        saving the state of a VM before making changes that might break it.
      </p>
      <p>
        You can make backups manually or you can rely on the automatic backup system, which will
        create backups of your VM at regular intervals. Note that automatic backups intervals are
        usually large (12h or more) and they are not guaranteed. If you are about to make a risky
        change, take a manual backup!
      </p>
      <p>
        Backups works even when the VM is running, but they are more consistent when the VM is
        stopped. If you want to make sure that the backup is as consistent as possible, stop the VM
        before making the backup and start it again after the backup is done.
      </p>
    </div>

    <div class="flex flex-col gap-2">
      <h2 class="text-info font-bold">Name, Time, Notes</h2>
      <p>
        <b class="text-info">Name</b> is the name of the backup. You can choose it when creating the
        backup. If the name is '<i>unkown</i>', it means that the backup was created automatically
        by the system.
      </p>
      <p><b class="text-info">Time</b> is the time when the backup was created.</p>
      <p>
        <b class="text-info">Notes</b> are optional notes that you can add to the backup. They can
        be useful for remembering why you created the backup or what changes you made after creating
        it. Automatic backups don't have notes.
      </p>

      <h2 class="text-info font-bold">Protection</h2>
      <p>
        Backups are also managed by an external pruning system, which will automatically delete old
        backups to free up space. Please not that this deletion is based on a complex set of rules
        and could delete any backup, not just the oldest. If you want to protect a backup from being
        deleted, you can mark it as protected. Protected backups won't be deleted by the pruning
        system, but they can still be deleted manually by you.
      </p>
    </div>

    <div class="flex flex-col gap-2">
      <h2 class="text-info font-bold">Restore</h2>
      <p>
        Restoring is the process of deleting the current state of the VM and replacing it with the
        state saved in the backup. You can restore a backup at any time, but the VM must be stopped
        before restoring.
      </p>
    </div>

    <div class="flex flex-col gap-2">
      <h2 class="text-info font-bold">Deletion</h2>
      <p>You can delete a manual backup at any time, but you cannot delete an automatic backup.</p>
    </div>
  </div>
</template>
