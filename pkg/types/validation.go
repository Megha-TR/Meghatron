package types

import (
	"fmt"
	"regexp"
	"time"
)

var namePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

func ValidName(name string) bool {
	return len(name) > 0 && len(name) <= 63 && namePattern.MatchString(name)
}
func ValidateMeta(meta ObjectMeta) error {
	if !ValidName(meta.Name) || !ValidName(meta.Namespace) {
		return fmt.Errorf("name and namespace must be 1–63 lowercase letters, numbers or hyphens, starting and ending with a letter or number")
	}
	return nil
}
func Terminal(pod *Pod) bool {
	return pod.Status.Phase == PodFailed || pod.Status.Phase == PodSucceeded
}
func Ready(pod *Pod) bool {
	if pod.Metadata.DeletionTimestamp != nil || pod.Status.Phase != PodRunning || len(pod.Status.ContainerStatuses) != len(pod.Spec.Containers) || len(pod.Spec.Containers) == 0 {
		return false
	}
	for _, s := range pod.Status.ContainerStatuses {
		if !s.Ready {
			return false
		}
	}
	return true
}
func MarkDeleting(pod *Pod) {
	if pod.Metadata.DeletionTimestamp == nil {
		now := time.Now()
		pod.Metadata.DeletionTimestamp = &now
	}
	pod.Status.Phase = PodTerminating
}
