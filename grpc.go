package main

import (
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	pb "github.com/uberswe/beubo_grpc/grpc"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
)

// Grpc

// protoc -I grpc --go_out=./grpc grpc/beubo.proto

const (
	grpcPort = ":50051"
)

var (
	responseChannel chan pb.PluginMessage
	requestChannel  chan pb.PluginMessage
)

type server struct{}

// Register is called by Beubo to fetch information about the plugin
func Register() map[string]string {
	return map[string]string{
		"name": "Beubo gRPC",
		// identifier should be a unique identifier used to differentiate this plugin from other plugins
		"identifier": "beubo_grpc",
	}
}

func (s *server) Connect(stream pb.BeuboGRPC_ConnectServer) error {
	for {
		go func() {
			for {
				response := <-responseChannel
				serialized, err := proto.Marshal(&response)
				if err != nil {
					log.Println("Could not serialize plugin message")
					return
				}
				err = stream.Send(&pb.Event{
					Key: "response",
					Values: []*any.Any{
						{
							TypeUrl: proto.MessageName(&response),
							Value:   serialized,
						},
					},
				})
				if err != nil {
					log.Print(err)
					return
				}
			}
		}()

		go func() {
			for {
				request := <-requestChannel
				serialized, err := proto.Marshal(&request)
				if err != nil {
					log.Println("Could not serialize plugin message")
					return
				}
				err = stream.Send(&pb.Event{
					Key: "request",
					Values: []*any.Any{
						{
							TypeUrl: proto.MessageName(&request),
							Value:   serialized,
						},
					},
				})
				if err != nil {
					log.Print(err)
					return
				}
			}
		}()

		event, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		log.Printf("Event received: %s (%s)\n", event.Key, event.Data)
		for _, anyVar := range event.Values {
			log.Println(anyVar.TypeUrl)
			if anyVar.TypeUrl == "beubo.PluginMessage" {
				var m pb.PluginMessage
				err := proto.Unmarshal(anyVar.Value, &m)
				if err != nil {
					return err
				}
				log.Printf("Plugin message unmarshalled %s\n", m.Name)
			}
		}
	}
}

func (s *server) Requests(pluginMessage *pb.PluginMessage, stream pb.BeuboGRPC_RequestsServer) error {
	log.Printf("Plugin registered to receive requests: %s (%s)\n", pluginMessage.Name, pluginMessage.Identifier)
	return nil
}

func grpcInit() {
	log.Printf("Starting gRPC server")
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("gRPC Listening on %s", grpcPort)
	s := grpc.NewServer()
	pb.RegisterBeuboGRPCServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
