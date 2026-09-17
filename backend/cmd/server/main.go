package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"groundclearance/internal/config"
	"groundclearance/internal/handler"
	"groundclearance/internal/model"
	"groundclearance/internal/repository"
	"groundclearance/internal/router"
	"groundclearance/internal/service"
	"groundclearance/internal/util"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()
	logger := util.NewLogger(slog.LevelInfo)
	if err := cfg.Validate(); err != nil {
		logger.Error("invalid configuration", "error", err.Error())
		os.Exit(1)
	}
	db, err := gorm.Open(postgres.Open(cfg.DBDSN()), &gorm.Config{})
	if err != nil {
		logger.Error("connect PostgreSQL failed", "error", err.Error())
		os.Exit(1)
	}
	if err := db.AutoMigrate(&model.User{}, &model.GroundUnit{}, &model.Turnaround{},
		&model.SafetyCheck{}, &model.ClearanceDecision{}, &model.AuditLog{}); err != nil {
		logger.Error("database migration failed", "error", err.Error())
		os.Exit(1)
	}
	if cfg.SeedDemoData {
		if err := service.NewSeedService(db, logger).Seed(); err != nil {
			logger.Error("seed data failed", "error", err.Error())
			os.Exit(1)
		}
	}
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword, DB: cfg.RedisDB})
	redisCtx, redisCancel := context.WithTimeout(context.Background(), 3*time.Second)
	if err := redisClient.Ping(redisCtx).Err(); err != nil {
		redisCancel()
		logger.Error("connect Redis failed", "error", err.Error())
		os.Exit(1)
	}
	redisCancel()
	defer redisClient.Close()

	userRepo := repository.NewUserRepository(db)
	unitRepo := repository.NewGroundUnitRepository(db)
	turnaroundRepo := repository.NewTurnaroundRepository(db)
	checkRepo := repository.NewSafetyCheckRepository(db)
	clearanceRepo := repository.NewClearanceDecisionRepository(db)

	userSvc := service.NewUserService(userRepo, logger)
	unitSvc := service.NewGroundUnitService(db, unitRepo, turnaroundRepo, clearanceRepo, logger)
	turnaroundSvc := service.NewTurnaroundService(db, turnaroundRepo, checkRepo, clearanceRepo, unitRepo, userRepo, logger)
	checkSvc := service.NewSafetyCheckService(db, checkRepo, turnaroundRepo, clearanceRepo, unitRepo, logger)
	clearanceSvc := service.NewClearanceDecisionService(db, clearanceRepo, turnaroundRepo, checkRepo, unitRepo, logger)

	engine := router.New(cfg, db, redisClient, logger,
		handler.NewUserHandler(userSvc, logger),
		handler.NewTurnaroundHandler(turnaroundSvc, logger),
		handler.NewGroundUnitHandler(unitSvc, logger),
		handler.NewSafetyCheckHandler(checkSvc, logger),
		handler.NewClearanceDecisionHandler(clearanceSvc, logger),
		handler.NewAuditLogHandler(db, logger),
	).Setup()

	server := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		logger.Info("server started", "port", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err.Error())
			stop()
		}
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err.Error())
	}
}
