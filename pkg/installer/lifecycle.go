package installer

func (inst *Installer) closeInputQueue() {
	inst.inputQueueCloseOnce.Do(func() {
		close(inst.InputQueue)
	})
}
