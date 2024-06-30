package customresources

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	PhasePending = "PENDING"
	PhaseRunning = "RUNNING"
	PhaseDone    = "DONE"
)

type PodDeleter struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   PodDeleterSpec
	Status PodDeleterStatus
}

type PodDeleterSpec struct {
	NumDeleted int
}

type PodDeleterStatus struct {
	Phase string
}

type PodDeleterList struct {
	metav1.TypeMeta
	metav1.ListMeta

	List []PodDeleter
}
