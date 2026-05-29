package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"project-nm/pkg/configs"
	"project-nm/pkg/contexts"
	"project-nm/pkg/endpoints"
	"project-nm/pkg/repositories"
	"project-nm/pkg/services"
	"project-nm/pkg/workers"

	"project-nm/pkg/endpoints/converter"
	"project-nm/pkg/migrations"
	"project-nm/pkg/transports"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

// App 應用程式核心結構
type App struct {
	WebEngine   *gin.Engine
	HttpHandler http.Handler
	DB          *gorm.DB
	RDB         *redis.Client
	Trans       *transports.Trans
}

// Mapper 負責將不同領域的轉換器註冊進 Converter
type Mapper struct {
	ConvertToViewModel converter.ConvertToViewModel
}

func (m Mapper) Register(c *converter.Converter) {
	v := reflect.ValueOf(m)
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		for j := 0; j < f.NumMethod(); j++ {
			if err := c.Register(f.Method(j).Interface()); err != nil {
				panic(err)
			}
		}
	}
}

// InitApplication 初始化應用程式環境
func InitApplication(db *gorm.DB, rdb *redis.Client) *App {
	convertToWebViewModel := converter.ConvertToViewModel{}
	mapper := Mapper{
		ConvertToViewModel: convertToWebViewModel,
	}
	structConverter := converter.New()
	mapper.Register(structConverter)

	// WebEngine
	webEngine := gin.Default()
	webEngine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"PUT", "PATCH", "GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"X-Total-Count"},
		AllowCredentials: true,
	}))

	trans := initTransport(db, structConverter)
	httpHandler := trans.MakeHttpHandler(webEngine)

	return &App{
		WebEngine:   webEngine,
		HttpHandler: httpHandler,
		DB:          db,
		RDB:         rdb,
		Trans:       trans,
	}
}

// Migrate 執行資料庫遷移
func (a *App) Migrate(db *gorm.DB) {
	if err := migrations.RunMigration(db); err != nil {
		panic(err)
	}
}

func (a *App) Serve(migrateDb *gorm.DB) {
	a.Migrate(migrateDb)

	cfg := configs.GetConfig()
	port := cfg.ServerPort
	if port == "" {
		port = "8080"
	}

	log.Printf("[project-nm] %s is starting on port :%s", cfg.ProjectID, port)

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	m := cmux.New(listener)
	m.SetReadTimeout(time.Second * 10)

	grpcListener := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpListener := m.Match(cmux.HTTP1())

	grpcServer := grpc.NewServer()
	// pb.RegisterPosBackendServer(grpcServer, &grpcInProject.PosBackendServer{
	//  MemberWallet:  a.Trans.MemberWalletTrans.Endpoint,
	// })

	httpServer := &http.Server{
		Handler: a.HttpHandler,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 將 ctx 傳入 WorkerManager，讓所有工人共享同一個關機燈號
	workerManager := workers.NewWorkerManager(ctx)

	// 註冊你的會員初始
	workerManager.Register(workers.NewMemberInitWorker(repositories.NewMemberRepo))
	workerManager.Register(workers.NewTradeWorker(repositories.NewTradeRepo, repositories.NewMemberRepo))

	workerManager.StartAll()

	errChan := make(chan error)

	go func() {
		if err := grpcServer.Serve(grpcListener); err != nil && err != cmux.ErrListenerClosed {
			errChan <- err
		}
	}()

	go func() {
		if err := httpServer.Serve(httpListener); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	go func() {
		if err := m.Serve(); err != nil {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("[project-nm] Received shutdown signal. Initiating graceful shutdown...")

		//  優先通知背景工人停止從 Redis Stream 搬新任務
		workerManager.StopAll()

		// 關閉網路服務層 (gRPC & HTTP)
		grpcServer.GracefulStop()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)

		log.Println("[project-nm] All servers and workers closed cleanly. Goodbye!")

	case err := <-errChan:
		log.Printf("[Critical Error] %v", err)
	}
}

// Serve 啟動單一埠號多協議服務
// func (a *App) Serve(migrateDb *gorm.DB) {
// 	a.Migrate(migrateDb)

// 	cfg := configs.GetConfig()
// 	port := cfg.ServerPort
// 	if port == "" {
// 		port = "8080"
// 	}

// 	log.Printf("[project-nm] %s is starting on port :%s", cfg.ProjectID, port)

// 	listener, err := net.Listen("tcp", ":"+port)
// 	if err != nil {
// 		log.Fatalf("Failed to listen: %v", err)
// 	}

// 	m := cmux.New(listener)
// 	m.SetReadTimeout(time.Second * 10)

// 	grpcListener := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
// 	httpListener := m.Match(cmux.HTTP1())

// 	grpcServer := grpc.NewServer()
// 	// pb.RegisterPosBackendServer(grpcServer, &grpcInProject.PosBackendServer{
// 	// 	MemberWallet:  a.Trans.MemberWalletTrans.Endpoint,
// 	// })

// 	httpServer := &http.Server{
// 		Handler: a.HttpHandler,
// 	}

// 	// workers
// 	workerManager := workers.NewWorkerManager()
// 	workerManager.Register(workers.NewMemberInitWorker(repositories.NewMemberRepo))
// 	workerManager.StartAll()

// 	errChan := make(chan error)
// 	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
// 	defer stop()

// 	go func() {
// 		if err := grpcServer.Serve(grpcListener); err != nil && err != cmux.ErrListenerClosed {
// 			errChan <- err
// 		}
// 	}()

// 	go func() {
// 		if err := httpServer.Serve(httpListener); err != nil && err != http.ErrServerClosed {
// 			errChan <- err
// 		}
// 	}()

// 	go func() {
// 		if err := m.Serve(); err != nil {
// 			errChan <- err
// 		}
// 	}()

// 	select {
// 	case <-ctx.Done():
// 		log.Println("[project-nm] Received shutdown signal...")
// 		grpcServer.GracefulStop()
// 		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 		defer cancel()
// 		_ = httpServer.Shutdown(shutdownCtx)
// 	case err := <-errChan:
// 		log.Printf("[Critical Error] %v", err)
// 	}
// }

func initTransport(db *gorm.DB, converter *converter.Converter) *transports.Trans {
	newAuthTransport := initAuthTransport(db, converter)
	newMemberTransport := initMemberTransport(db, converter)
	newTrans := &transports.Trans{
		AuthTrans:   newAuthTransport,
		MemberTrans: newMemberTransport,
	}
	return newTrans
}

func initAuthTransport(db *gorm.DB, converter *converter.Converter) *transports.AuthTransport {
	return &transports.AuthTransport{
		Endpoint: &endpoints.AuthEndpoint{
			Converter: converter,
			Service:   &services.AuthService{},
			CtxFactory: &contexts.UserFactory{
				DB:              db,
				UserRepoFactory: repositories.NewUserRepo,
			},
		},
	}
}

func initMemberTransport(db *gorm.DB, converter *converter.Converter) *transports.MemberTransport {
	return &transports.MemberTransport{
		Endpoint: &endpoints.MemberEndpoint{
			Converter: converter,
			Service:   &services.MemberService{},
			CtxFactory: &contexts.MemberFactory{
				DB:                db,
				MemberRepoFactory: repositories.NewMemberRepo,
			},
		},
	}
}

// func initTradeTransport(db *gorm.DB, converter *converter.Converter) *transports.MemberTransport {
// 	return &transports.MemberTransport{
// 		Endpoint: &endpoints.MemberEndpoint{
// 			Converter: converter,
// 			Service:   &services.TradeService{},
// 			CtxFactory: &contexts.MemberFactory{
// 				DB:                db,
// 				MemberRepoFactory: repositories.NewMemberRepo,
// 			},
// 		},
// 	}
// }
