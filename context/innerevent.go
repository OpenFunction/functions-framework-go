package context

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"k8s.io/klog/v2"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
)

type InnerEvent interface {

	// SetMetadata sets the metadata in innerEventData.
	SetMetadata(key string, value string)

	// GetMetadata returns the metadata in innerEventData.
	GetMetadata() map[string]string

	// SetUserData sets the userData in innerEventData.
	SetUserData(data interface{})

	// GetUserData returns the userData in innerEventData.
	GetUserData() interface{}

	// GetCloudEvent returns the cloudevent object in innerEvent.
	GetCloudEvent() cloudevents.Event

	// MergeMetadata merges the metadata of the incoming event into the new event.
	MergeMetadata(event InnerEvent)

	// Clone clones a new innerEvent.
	Clone(event *cloudevents.Event)

	// GetCloudEventJSON returns the cloudevent in json format.
	GetCloudEventJSON() []byte

	// SetSubject sets the subject of the cloudevent in the innerEvent.
	SetSubject(s string)
}

type innerEvent struct {
	mu         sync.Mutex
	cloudevent *cloudevents.Event
	data       *innerEventData
}

type innerEventData struct {
	Metadata map[string]string `json:"metadata,omitempty"`
	UserData interface{}       `json:"userData,omitempty"`
}

func NewInnerEvent(ctx RuntimeContext) InnerEvent {
	ie := &innerEvent{}
	ce := cloudevents.NewEvent()
	ie.cloudevent = &ce
	ie.data = &innerEventData{}
	ie.data.Metadata = map[string]string{}
	ie.initCloudEventHeaders(ctx)
	return ie
}

func (inner *innerEvent) SetMetadata(key string, value string) {
	inner.mu.Lock()
	defer func() {
		inner.save()
		inner.mu.Unlock()
	}()
	inner.data.Metadata[key] = value
}

func (inner *innerEvent) GetMetadata() map[string]string {
	inner.mu.Lock()
	defer inner.mu.Unlock()
	return inner.data.Metadata
}

func (inner *innerEvent) SetUserData(data interface{}) {
	inner.mu.Lock()
	defer func() {
		inner.save()
		inner.mu.Unlock()
	}()
	inner.data.UserData = data
}

func (inner *innerEvent) SetSubject(s string) {
	inner.mu.Lock()
	defer inner.mu.Unlock()
	inner.cloudevent.SetSubject(s)
}

func (inner *innerEvent) GetUserData() interface{} {
	return inner.data.UserData
}

func (inner *innerEvent) initCloudEventHeaders(ctx RuntimeContext) {
	var source string
	var t string

	if ctx.GetMode() == KubernetesMode {
		source = fmt.Sprintf("%s/%s", ctx.GetPodNamespace(), ctx.GetName())
		t = fmt.Sprintf("%s.%s.%s", innerEventTypePrefix, ctx.GetPodNamespace(), ctx.GetName())

	} else {
		source = ctx.GetName()
		t = fmt.Sprintf("%s.%s", innerEventTypePrefix, ctx.GetName())
	}

	inner.cloudevent.SetID(uuid.New().String())
	inner.cloudevent.SetTime(time.Now())
	inner.cloudevent.SetSource(source)
	inner.cloudevent.SetType(t)
	inner.cloudevent.SetDataContentType(cloudevents.ApplicationJSON)
}

func (inner *innerEvent) GetCloudEvent() cloudevents.Event {
	return *inner.cloudevent
}

func (inner *innerEvent) GetCloudEventJSON() []byte {
	ceBytes, err := json.Marshal(*inner.cloudevent)
	if err != nil {
		return nil
	}
	return ceBytes
}

func (inner *innerEvent) MergeMetadata(event InnerEvent) {
	if event == nil || event.GetMetadata() == nil {
		return
	}

	inner.mu.Lock()
	defer func() {
		inner.save()
		inner.mu.Unlock()
	}()

	for k, v := range event.GetMetadata() {
		inner.data.Metadata[k] = v
	}
}

func (inner *innerEvent) Clone(event *cloudevents.Event) {
	inner.mu.Lock()
	defer func() {
		inner.save()
		inner.mu.Unlock()
	}()

	inner.cloudevent = event

	d := &innerEventData{}
	if event.Data() != nil {
		if err := event.DataAs(d); err == nil {
			inner.data.Metadata = d.Metadata
			inner.data.UserData = d.UserData
		} else {
			inner.data.UserData = event.Data()
		}
	}
}

func (inner *innerEvent) save() {
	if inner.cloudevent == nil || (inner.data != nil && reflect.DeepEqual(inner.data.Metadata, map[string]string{}) && inner.data.UserData == nil) {
		fmt.Println(inner.data.UserData)
		return
	}

	if err := inner.cloudevent.SetData(cloudevents.ApplicationJSON, *inner.data); err != nil {
		klog.Errorf("failed to set cloudevent data: %v\n", err)
	}
}

func convertEvent(ctx RuntimeContext, inputName string, data interface{}) InnerEvent {
	inner := NewInnerEvent(ctx)
	ce := &cloudevents.Event{}
	if data != nil {
		switch data := data.(type) {
		case []byte:
			if err := json.Unmarshal(data, ce); err != nil {
				inner.SetSubject(inputName)
				inner.SetUserData(data)
				return inner
			} else {
				inner.Clone(ce)
				return inner
			}
		case cloudevents.Event:
			inner.Clone(&data)
			return inner
		default:
			inner.SetSubject(inputName)
			inner.SetUserData(data)
			return inner
		}
	}
	inner.SetSubject(inputName)
	return inner
}
