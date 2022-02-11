/*
This file is derived from: https://github.com/dapr/go-sdk/tree/main/service/grpc
and is referenced here for testing the dapr service.
*/

package async

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net"
	"os"
	"strings"

	cpb "github.com/dapr/dapr/pkg/proto/common/v1"
	pb "github.com/dapr/dapr/pkg/proto/runtime/v1"
	"github.com/dapr/go-sdk/actor"
	"github.com/dapr/go-sdk/actor/config"
	"github.com/dapr/go-sdk/service/common"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	"github.com/OpenFunction/functions-framework-go/runtime/async/internal"
)

// FakeServer is the gRPC service implementation for Dapr.
type FakeServer struct {
	pb.UnimplementedAppCallbackServer
	listener        net.Listener
	invokeHandlers  map[string]common.ServiceInvocationHandler
	topicRegistrar  internal.TopicRegistrar
	bindingHandlers map[string]common.BindingInvocationHandler
	authToken       string
}

func (s *FakeServer) RegisterActorImplFactory(f actor.Factory, opts ...config.Option) {
	panic("Actor is not supported by gRPC API")
}

// Start registers the server and starts it.
func (s *FakeServer) Start() error {
	gs := grpc.NewServer()
	pb.RegisterAppCallbackServer(gs, s)
	return gs.Serve(s.listener)
}

// Stop stops the previously started service.
func (s *FakeServer) Stop() error {
	return s.listener.Close()
}

// AddBindingInvocationHandler appends provided binding invocation handler with its name to the service.
func (s *FakeServer) AddBindingInvocationHandler(name string, fn common.BindingInvocationHandler) error {
	if name == "" {
		return fmt.Errorf("binding name required")
	}
	if fn == nil {
		return fmt.Errorf("binding handler required")
	}
	s.bindingHandlers[name] = fn
	return nil
}

// ListInputBindings is called by Dapr to get the list of bindings the app will get invoked by. In this example, we are telling Dapr
// To invoke our app with a binding named storage.
func (s *FakeServer) ListInputBindings(ctx context.Context, in *empty.Empty) (*pb.ListInputBindingsResponse, error) {
	list := make([]string, 0)
	for k := range s.bindingHandlers {
		list = append(list, k)
	}

	return &pb.ListInputBindingsResponse{
		Bindings: list,
	}, nil
}

// OnBindingEvent gets invoked every time a new event is fired from a registered binding. The message carries the binding name, a payload and optional metadata.
func (s *FakeServer) OnBindingEvent(ctx context.Context, in *pb.BindingEventRequest) (*pb.BindingEventResponse, error) {
	if in == nil {
		return nil, errors.New("nil binding event request")
	}
	if fn, ok := s.bindingHandlers[in.Name]; ok {
		e := &common.BindingEvent{
			Data:     in.Data,
			Metadata: in.Metadata,
		}
		data, err := fn(ctx, e)
		if err != nil {
			return nil, errors.Wrapf(err, "error executing %s binding", in.Name)
		}
		return &pb.BindingEventResponse{
			Data: data,
		}, nil
	}

	return nil, fmt.Errorf("binding not implemented: %s", in.Name)
}

// AddServiceInvocationHandler appends provided service invocation handler with its method to the service.
func (s *FakeServer) AddServiceInvocationHandler(method string, fn common.ServiceInvocationHandler) error {
	if method == "" {
		return fmt.Errorf("servie name required")
	}
	if fn == nil {
		return fmt.Errorf("invocation handler required")
	}
	s.invokeHandlers[method] = fn
	return nil
}

// OnInvoke gets invoked when a remote service has called the app through Dapr.
func (s *FakeServer) OnInvoke(ctx context.Context, in *cpb.InvokeRequest) (*cpb.InvokeResponse, error) {
	if in == nil {
		return nil, errors.New("nil invoke request")
	}
	if s.authToken != "" {
		if md, ok := metadata.FromIncomingContext(ctx); !ok {
			return nil, errors.New("authentication failed")
		} else if vals := md.Get(common.APITokenKey); len(vals) > 0 {
			if vals[0] != s.authToken {
				return nil, errors.New("authentication failed: app token mismatch")
			}
		} else {
			return nil, errors.New("authentication failed. app token key not exist")
		}
	}
	if fn, ok := s.invokeHandlers[in.Method]; ok {
		e := &common.InvocationEvent{}
		e.ContentType = in.ContentType

		if in.Data != nil {
			e.Data = in.Data.Value
			e.DataTypeURL = in.Data.TypeUrl
		}

		if in.HttpExtension != nil {
			e.Verb = in.HttpExtension.Verb.String()
			e.QueryString = in.HttpExtension.Querystring
		}

		ct, er := fn(ctx, e)
		if er != nil {
			return nil, er
		}

		if ct == nil {
			return &cpb.InvokeResponse{}, nil
		}

		return &cpb.InvokeResponse{
			ContentType: ct.ContentType,
			Data: &any.Any{
				Value:   ct.Data,
				TypeUrl: ct.DataTypeURL,
			},
		}, nil
	}
	return nil, fmt.Errorf("method not implemented: %s", in.Method)
}

// AddTopicEventHandler appends provided event handler with topic name to the service.
func (s *FakeServer) AddTopicEventHandler(sub *common.Subscription, fn common.TopicEventHandler) error {
	if sub == nil {
		return errors.New("subscription required")
	}
	if err := s.topicRegistrar.AddSubscription(sub, fn); err != nil {
		return err
	}

	return nil
}

// OnTopicEvent fired whenever a message has been published to a topic that has been subscribed.
// Dapr sends published messages in a CloudEvents v1.0 envelope.
func (s *FakeServer) OnTopicEvent(ctx context.Context, in *pb.TopicEventRequest) (*pb.TopicEventResponse, error) {
	if in == nil || in.Topic == "" || in.PubsubName == "" {
		// this is really Dapr issue more than the event request format.
		// since Dapr will not get updated until long after this event expires, just drop it
		return &pb.TopicEventResponse{Status: pb.TopicEventResponse_DROP}, errors.New("pub/sub and topic names required")
	}
	key := in.PubsubName + "-" + in.Topic
	if sub, ok := s.topicRegistrar[key]; ok {
		data := interface{}(in.Data)
		if len(in.Data) > 0 {
			mediaType, _, err := mime.ParseMediaType(in.DataContentType)
			if err == nil {
				var v interface{}
				switch mediaType {
				case "application/json":
					if err := json.Unmarshal(in.Data, &v); err == nil {
						data = v
					}
				case "text/plain":
					// Assume UTF-8 encoded string.
					data = string(in.Data)
				default:
					if strings.HasPrefix(mediaType, "application/") &&
						strings.HasSuffix(mediaType, "+json") {
						if err := json.Unmarshal(in.Data, &v); err == nil {
							data = v
						}
					}
				}
			}
		}

		e := &common.TopicEvent{
			ID:              in.Id,
			Source:          in.Source,
			Type:            in.Type,
			SpecVersion:     in.SpecVersion,
			DataContentType: in.DataContentType,
			Data:            data,
			RawData:         in.Data,
			Topic:           in.Topic,
			PubsubName:      in.PubsubName,
		}
		h := sub.DefaultHandler
		if in.Path != "" {
			if pathHandler, ok := sub.RouteHandlers[in.Path]; ok {
				h = pathHandler
			}
		}
		if h == nil {
			return &pb.TopicEventResponse{Status: pb.TopicEventResponse_RETRY}, fmt.Errorf(
				"route %s for pub/sub and topic combination not configured: %s/%s",
				in.Path, in.PubsubName, in.Topic,
			)
		}
		retry, err := h(ctx, e)
		if err == nil {
			return &pb.TopicEventResponse{Status: pb.TopicEventResponse_SUCCESS}, nil
		}
		if retry {
			return &pb.TopicEventResponse{Status: pb.TopicEventResponse_RETRY}, err
		}
		return &pb.TopicEventResponse{Status: pb.TopicEventResponse_DROP}, err
	}
	return &pb.TopicEventResponse{Status: pb.TopicEventResponse_RETRY}, fmt.Errorf(
		"pub/sub and topic combination not configured: %s/%s",
		in.PubsubName, in.Topic,
	)
}

// NewFakeService creates new Service.
func NewFakeService(address string) (common.Service, *FakeServer, error) {
	if address == "" {
		return nil, nil, errors.New("nil address")
	}
	s := newFakeService(bufconn.Listen(1024 * 1024))
	return s, s, nil
}

func newFakeService(lis net.Listener) *FakeServer {
	return &FakeServer{
		listener:        lis,
		invokeHandlers:  make(map[string]common.ServiceInvocationHandler),
		topicRegistrar:  make(internal.TopicRegistrar),
		bindingHandlers: make(map[string]common.BindingInvocationHandler),
		authToken:       os.Getenv(common.AppAPITokenEnvVar),
	}
}
