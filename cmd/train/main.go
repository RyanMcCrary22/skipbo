// Command skipbo-train starts the gRPC training server for RL agents.
//
// Usage:
//
//	skipbo-train [flags]
//
// Flags:
//
//	--port N    gRPC port (default: 50051)
package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"

	pb "github.com/RyanMcCrary22/skipbo/proto/skipbopb"
	"github.com/RyanMcCrary22/skipbo/training"
)

func main() {
	port := flag.Int("port", 50051, "gRPC server port")
	onnxLib := flag.String("onnx-lib", "", "Path to libonnxruntime.dylib or .so (required for ONNX opponents)")
	flag.Parse()

	if *onnxLib != "" {
		log.Printf("Initializing ONNX runtime from %s...", *onnxLib)
		if err := training.InitOnnxRuntime(*onnxLib); err != nil {
			log.Fatalf("Failed to initialize ONNX: %v", err)
		}
		defer training.DestroyOnnxRuntime()
	} else {
		log.Printf("Warning: --onnx-lib not provided. ONNX opponents will fail to load.")
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", *port, err)
	}

	grpcServer := grpc.NewServer()
	srv := training.NewServer()
	pb.RegisterSkipBoEnvServer(grpcServer, srv)

	log.Printf("🎮 Skip-Bo Training Server listening on :%d", *port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
