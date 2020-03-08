package converter

import (
	corev1 "k8s.io/api/core/v1"
	"WarpCloud/walm/pkg/models/k8s"
	"sort"
	"WarpCloud/walm/pkg/k8s/utils"
)

func ConvertEventListFromK8s(oriEvents []corev1.Event) ([]k8s.Event, error) {
	if oriEvents == nil {
		return nil, nil
	}
	sort.Sort(utils.SortableEvents(oriEvents))

	walmEvents := []k8s.Event{}
	for _, oriEvent := range oriEvents {
		event := oriEvent.DeepCopy()
		walmEvent := k8s.Event{
			Type:           event.Type,
			Reason:         event.Reason,
			Message:        event.Message,
			Count:          event.Count,
			FirstTimestamp: event.FirstTimestamp.String(),
			LastTimestamp:  event.LastTimestamp.String(),
			From:           utils.FormatEventSource(event.Source),
		}
		walmEvents = append(walmEvents, walmEvent)
	}

	return walmEvents, nil
}