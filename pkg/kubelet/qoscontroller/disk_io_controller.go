
package qoscontroller

// Disk I/O Controller
type DiskIOController struct {
}

// initialize of QosController is implemented by DiskIOController and does all the initialization works
func (dc *DiskIOController) initialize(qosResourceStatus *QosResourceStatus) error {
	return nil
}

//process of QosController is implemented by DiskIOController and does all what a disk I/O controller has to do
func (dc *DiskIOController) process(qosResourceStatus *QosResourceStatus) error {
	return nil
}
