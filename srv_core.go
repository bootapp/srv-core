package main

import (
	"context"
	"flag"
	"github.com/bootapp/rest-grpc-oauth2/auth"
	"github.com/bootapp/srv-core/oauth"
	pb "github.com/bootapp/srv-core/proto/core"
	"github.com/bootapp/srv-core/server"
	"github.com/bootapp/srv-core/utils"
	"github.com/golang/glog"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var (
	grpcEndpoint = flag.String("grpc_endpoint", ":9092", "The endpoint of the core grpc service")
	httpEndpoint = flag.String("http_endpoint", ":8092", "The endpoint of the core restful service")
	oauthEndpoint = flag.String("oauth_endpoint", ":9081", "The endpoint of the oauth server")
)

func main() {
	_ = flag.Set("alsologtostderr", "true")
	flag.Parse()
	defer glog.Flush()
	ctx := context.Background()
	ctx, cancel := context.WithCancel(context.Background())
	//====== initialize auth client
	authenticator :=auth.GetInstance()
	//====== initialize oauth server
	oauthServer := oauth.NewPassOAuthServer()
	//====== read configs and listen changes from apollo
	rpcSrv := server.GRpcServiceAddr{}
	server.ApolloConfig(ctx, false, &rpcSrv, &oauthServer, authenticator)

	utils.InitRedis("localhost:6379", "")

	go func() {
		glog.Info("oauth server listening...")
		glog.Fatal(http.ListenAndServe(*oauthEndpoint, nil))
		glog.Info("oauth server stopped.")
	}()
	go func() {
		defer cancel()
		_ = gwRun(ctx, *httpEndpoint, *grpcEndpoint)
	}()
	_ = grpcRun(ctx, *grpcEndpoint, rpcSrv)
}

func grpcRun(ctx context.Context, grpcEndpoint string, addr server.GRpcServiceAddr) error {
	l, err := net.Listen("tcp", grpcEndpoint)
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	srvCoreUserSrv := server.NewSrvCoreUserServiceServer(addr.DALCoreUserSrv)
	srvCoreSecuritySrv := server.NewSecurityServer(addr.DALCoreUserSrv)
	pb.RegisterSrvCoreUserServiceServer(grpcServer, srvCoreUserSrv)
	pb.RegisterSrvSecurityServiceServer(grpcServer, srvCoreSecuritySrv)
	go func() {
		defer grpcServer.GracefulStop()
		<-ctx.Done()
		glog.Info("grpc server shutting down...")
	}()
	glog.Info("grpc server running...")
	return grpcServer.Serve(l)
}

func gwRun(ctx context.Context, httpEndpoint string, grpcEndpoint string) error {
	//ctx, cancel := context.WithCancel(context.Background())
	mux := runtime.NewServeMux(
		runtime.WithForwardResponseOption(auth.GatewayResponseCookieAnnotator),
		runtime.WithMetadata(auth.GatewayRequestCookieParser))
	opts := []grpc.DialOption{grpc.WithInsecure()}
	if err := pb.RegisterSrvCoreUserServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, opts); err != nil {
		glog.Fatal("failed to start rest gateway: %v", err)
		return err
	}
	if err := pb.RegisterSrvSecurityServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, opts); err != nil {
		glog.Fatal("failed to start rest gateway: %v", err)
		return err
	}
	srv := &http.Server {
		Addr:    httpEndpoint,
		Handler: mux,
	}
	// graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<- c
		glog.Info("rest gateway shutting down...")
		_, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err:= srv.Shutdown(ctx); err != nil {
			glog.Fatalf("failed to shutdown rest gateway %v", err)
		}
	}()
	glog.Info("restful gateway running...")
	return srv.ListenAndServe()
}