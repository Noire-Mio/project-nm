package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/signal"
	"project-nm/pkg/configs"
	"project-nm/pkg/contexts"
	"project-nm/pkg/endpoints"
	"project-nm/pkg/endpoints/converter"
	"project-nm/pkg/entities"
	grpcInProject "project-nm/pkg/grpc"
	clients "project-nm/pkg/grpc/client"
	"project-nm/pkg/grpc/pb"
	"project-nm/pkg/migrations"
	"project-nm/pkg/repositories"
	"project-nm/pkg/services"
	"project-nm/pkg/transports"
	"project-nm/pkg/workers"
	"reflect"
	"syscall"
	"time"

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
	
	targetSchemas := []string{"tenant_001", "tenant_002", "tenant_003"}
	a.PreheatProducts(migrateDb, a.RDB, targetSchemas)

	cfg := configs.GetConfig()
	port := cfg.ServerPort
	if port == "" {
		port = "8080"
	}

	log.Printf("[INFO] [%s] Server is starting on port :%s", cfg.ProjectID, port)

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("[CRITICAL] Failed to listen: %v", err)
	}

	m := cmux.New(listener)
	m.SetReadTimeout(time.Second * 10)

	grpcListener := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpListener := m.Match(cmux.HTTP1())

	projectNMClient, err := clients.NewProjectNMGrpcClient()
	if err != nil {
		log.Fatalf("[CRITICAL] Failed to initialize gRPC Client: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterProjectGrpcServer(grpcServer, &grpcInProject.ProjectNMServer{
		TradeEndpoint: a.Trans.TradeTrans.Endpoint,
	})

	httpServer := &http.Server{
		Handler: a.HttpHandler,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	workerManager := workers.NewWorkerManager(ctx)
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
		log.Println("[INFO] Received shutdown signal. Initiating graceful shutdown...")
		workerManager.StopAll()
		grpcServer.GracefulStop()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)

		if err := projectNMClient.Close(); err != nil {
			log.Printf("[WARNING] Failed to close gRPC Client connection cleanly: %v", err)
		}

		log.Println("[INFO] All servers, clients and workers closed cleanly. Goodbye!")

	case err := <-errChan:
		log.Printf("[CRITICAL] %v", err)
	}
}

// PreheatProducts 商品預載 
func (a *App) PreheatProducts(db *gorm.DB, rdb *redis.Client, schemas []string) {
	ctx := context.Background()
	var totalProductsWarmedUp int

	log.Println("[INFO] Cache preheating initialized.")

	checkSQL := `
        SELECT EXISTS (
            SELECT FROM information_schema.tables 
            WHERE table_schema = ? AND table_name = 'product'
        );
    `

	for _, schema := range schemas {
		var tableExists bool

		// 核心防線：重試等待資料表完全 Commit
		for retry := 1; retry <= 10; retry++ {
			err := db.Raw(checkSQL, schema).Scan(&tableExists).Error
			if err == nil && tableExists {
				break
			}
			log.Printf("[INFO] Waiting for schema %s migration to complete... (Retry %d/10)", schema, retry)
			time.Sleep(1 * time.Second)
		}

		if !tableExists {
			log.Printf("[ERROR] Skipping preheat for schema %s: Table 'product' does not exist after maximum retries.", schema)
			continue
		}

		var products []entities.Product

		tableName := fmt.Sprintf("%s.product", schema)
		if err := db.Table(tableName).Find(&products).Error; err != nil {
			log.Printf("[WARNING] Failed to fetch products for schema %s: %v", schema, err)
			continue
		}

		for _, p := range products {
			// 商品基本資料快取 (Hash 格式)
			productKey := fmt.Sprintf("cache:product:%s:%d", schema, p.ID)
			_ = rdb.HMSet(ctx, productKey, map[string]interface{}{
				"price": p.Price.String(),
			}).Err()

			// 商品獨立的原子庫存計數器 (String 格式，供 Lua 減法操作)
			stockKey := fmt.Sprintf("cache:product:stock:%s:%d", schema, p.ID)
			_ = rdb.Set(ctx, stockKey, p.Stock, 0).Err()

			totalProductsWarmedUp++
		}
	}

	log.Printf("[INFO] Cache preheating completed. Total schemas checked: %d, Total products loaded: %d", len(schemas), totalProductsWarmedUp)
}

func initTransport(db *gorm.DB, converter *converter.Converter) *transports.Trans {
	newAuthTransport := initAuthTransport(db, converter)
	newMemberTransport := initMemberTransport(db, converter)
	newTradeTransport := initTradeTransport(db, converter)
	newTrans := &transports.Trans{
		AuthTrans:   newAuthTransport,
		MemberTrans: newMemberTransport,
		TradeTrans:  newTradeTransport,
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

func initTradeTransport(db *gorm.DB, converter *converter.Converter) *transports.TradeTransport {
	return &transports.TradeTransport{
		Endpoint: &endpoints.TradeEndpoint{
			Converter: converter,
			Service:   &services.TradeService{},
			CtxFactory: &contexts.TradeFactory{
				DB:                db,
				TradeRepoFactory:  repositories.NewTradeRepo,
				MemberRepoFactory: repositories.NewMemberRepo,
			},
		},
	}
}
