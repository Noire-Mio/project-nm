package clients

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"project-nm/pkg/configs"
	"project-nm/pkg/grpc/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type IProjectNMClient interface {
	ExecuteOrder(info *pb.GRPCUserInfo, items []*pb.TradeGrpcItem) (*pb.TradeGrpcResponse, error)
	Close() error 
}

type ProjectNMClient struct {
	conn *grpc.ClientConn 
}

// 初始化時直接建立好永久 TCP 連線（長連線池機制）
func NewProjectNMGrpcClient() (IProjectNMClient, error) {
	client := &ProjectNMClient{}
	conn, err := client.initGRPCConnection()
	if err != nil {
		return nil, fmt.Errorf("GRPC_CLIENT_INIT_FAILED: 建立全局連線失敗: %w", err)
	}
	client.conn = conn
	return client, nil
}


func (s *ProjectNMClient) initGRPCConnection() (*grpc.ClientConn, error) {
	target := configs.GetConfig().RelationalGRPC.ProjectNMUrl
	useTLSConnection := configs.GetConfig().RelationalGRPC.UseTLSConnection

	if strings.HasPrefix(target, "192.168") {
		useTLSConnection = false
	}

	var opts []grpc.DialOption

	// 優化連線逾時
	opts = append(opts, grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		dialer := &net.Dialer{Timeout: 3 * time.Second}
		return dialer.DialContext(ctx, "tcp", addr)
	}))

	if useTLSConnection {
		credential := credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})
		opts = append(opts, grpc.WithTransportCredentials(credential))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (s *ProjectNMClient) ExecuteOrder(info *pb.GRPCUserInfo, items []*pb.TradeGrpcItem) (*pb.TradeGrpcResponse, error) {
	c := pb.NewProjectGrpcClient(s.conn)

	// 將 context.Background() 改為能控制逾時的 Context，防止遠端掛掉時整個執行緒卡死
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.ExecuteOrder(ctx, &pb.TradeGrpcRequest{
		GrpcUserInfo: info,
		Items:        items,
	})
}

func (s *ProjectNMClient) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}
