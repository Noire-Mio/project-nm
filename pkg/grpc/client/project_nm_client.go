package clients

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"sync" // 🎯 引入原子操作同步鎖
	"time"

	"project-nm/pkg/configs"
	"project-nm/pkg/grpc/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// 全局靜態變數：將真實的長連線生命週期提升到套件最高級別，超脫於任何外部實體
var (
	globalConn *grpc.ClientConn // 🎯 全網唯一的實體 TCP 長連線
	connOnce   sync.Once        // 🎯 鋼鐵防線：確保高併發下連線初始化只會執行一次
	initErr    error            // 紀錄初始化時的錯誤
)

type IProjectNMClient interface {
	ExecuteOrder(info *pb.GRPCUserInfo, items []*pb.TradeGrpcItem) (*pb.TradeGrpcResponse, error)
	Close() error
}

type ProjectNMClient struct {
	// 🎯 這裡保持空白！因為我們不再依賴這個結構體實體內部的指標
}

// NewProjectNMGrpcClient 外部完全不改，所以這個工廠依然會被空的呼叫
func NewProjectNMGrpcClient() (IProjectNMClient, error) {
	// 這裡直接返回一個空外殼實體即可，因為精髓全部轉移到內部的單例模式了
	return &ProjectNMClient{}, nil
}

// 內部方法：專門負責單例連線的建立
func (s *ProjectNMClient) ensureConnection() error {
	// 🎯 利用 sync.Once 機制，4000 個人同時搶，也只有一個人能進來建立連線
	connOnce.Do(func() {
		target := configs.GetConfig().RelationalGRPC.ProjectNMUrl
		useTLSConnection := configs.GetConfig().RelationalGRPC.UseTLSConnection

		if strings.HasPrefix(target, "192.168") {
			useTLSConnection = false
		}

		var opts []grpc.DialOption

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

		// 正式建立全局唯一的長連線
		globalConn, initErr = grpc.NewClient(target, opts...)
	})

	return initErr
}

func (s *ProjectNMClient) ExecuteOrder(info *pb.GRPCUserInfo, items []*pb.TradeGrpcItem) (*pb.TradeGrpcResponse, error) {
	// 🎯 核心補強（延遲加載）：在請求要發射的當下，強行校驗並確保連線池已經開通
	if err := s.ensureConnection(); err != nil {
		return nil, fmt.Errorf("GRPC_LAZY_INIT_FAILED: 全局長連線延遲初始化失敗: %w", err)
	}

	// 🎯 修正：使用 globalConn 全局靜態長連線，徹底擺脫 s.conn 造成的 nil 指標崩潰！
	c := pb.NewProjectGrpcClient(globalConn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.ExecuteOrder(ctx, &pb.TradeGrpcRequest{
		GrpcUserInfo: info,
		Items:        items,
	})
}

func (s *ProjectNMClient) Close() error {
	// 優雅關機時，改為關閉全局靜態連線
	if globalConn != nil {
		return globalConn.Close()
	}
	return nil
}
