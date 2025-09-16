package wasimoffv1

// Additional helpers on the generated types.

// Fill any unset (nil) task parameters from a parent task specification.
func (wt *Task_Wasip1_Params) InheritNil(parent *Task_Wasip1_Params) *Task_Wasip1_Params {
	if parent == nil {
		// nothing to do when parent is nil
		return wt
	}
	if wt.Binary == nil {
		wt.Binary = parent.Binary
	}
	if wt.Args == nil {
		wt.Args = parent.Args
	}
	if wt.Envs == nil {
		wt.Envs = parent.Envs
	}
	if wt.Stdin == nil {
		wt.Stdin = parent.Stdin
	}
	if wt.Rootfs == nil {
		wt.Rootfs = parent.Rootfs
	}
	if wt.Artifacts == nil {
		wt.Artifacts = parent.Artifacts
	}
	return wt
}

// Return a string list of needed files for a task request.
func (tr *Task_Wasip1_Request) GetRequiredFiles() (files []string) {
	files = make([]string, 0, 2) // usually max. binary + rootfs
	p := tr.Params

	if p.Binary != nil && p.Binary.GetRef() != "" {
		files = append(files, *p.Binary.Ref)
	}
	if p.Rootfs != nil && p.Rootfs.GetRef() != "" {
		files = append(files, *p.Rootfs.Ref)
	}

	return files
}
