// package qoscontroller guarantees the resource availability for the primary application. It constantly watches the node
// and application status. In case of performance drop, correction actions like freeze or kill are triggered
// to ensure primary applications' stability.
package qoscontroller
