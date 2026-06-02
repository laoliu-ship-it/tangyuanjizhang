package main

import (
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"fandianjizhang/server/config"
	"fandianjizhang/server/internal/handler"
	"fandianjizhang/server/internal/middleware"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
	"fandianjizhang/server/internal/service"
	"fandianjizhang/server/pkg/excel"
	pkgocr "fandianjizhang/server/pkg/ocr"
	pkgllm "fandianjizhang/server/pkg/llm"
	"fandianjizhang/server/pkg/storage"
	webstatic "fandianjizhang/server/web"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 连接数据库
	db, err := gorm.Open(mysql.Open(cfg.DB.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 自动迁移新增表
	if err := db.AutoMigrate(&model.TenantLLMConfig{}); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 初始化 Casbin enforcer（内存策略，启动时加载固定权限规则）
	enforcer, err := middleware.NewCasbinEnforcer()
	if err != nil {
		log.Fatalf("初始化 Casbin 失败: %v", err)
	}

	// 初始化 OCR 引擎
	ocrEngine, err := pkgocr.NewEngine(cfg)
	if err != nil {
		log.Fatalf("初始化 OCR 引擎失败: %v", err)
	}

	// 初始化本地存储
	store := storage.NewLocalStorage(cfg.Upload.Path)

	// 初始化 Repo 层
	userRepo := repo.NewUserRepo(db)
	tenantRepo := repo.NewTenantRepo(db)
	tenantMemberRepo := repo.NewTenantMemberRepo(db)
	categoryRepo := repo.NewCategoryRepo(db)
	merchantRepo := repo.NewMerchantRepo(db)
	transactionRepo := repo.NewTransactionRepo(db)
	llmConfigRepo := repo.NewTenantLLMConfigRepo(db)

	// 初始化 Service 层
	authSvc := service.NewAuthService(userRepo, tenantRepo, tenantMemberRepo, categoryRepo, cfg)
	tenantSvc := service.NewTenantService(tenantRepo, tenantMemberRepo, userRepo, categoryRepo)
	categorySvc := service.NewCategoryService(categoryRepo)
	merchantSvc := service.NewMerchantService(merchantRepo)
	transactionSvc := service.NewTransactionService(transactionRepo, categoryRepo, merchantRepo)
	ocrSvc := service.NewOCRService(ocrEngine, cfg.OCR.AIMode)
	statisticsSvc := service.NewStatisticsService(transactionRepo)
	llmSvc := service.NewLLMService(cfg.LLM, llmConfigRepo)
	// LLM 结果缓存：同文件内容 SHA256 相同则直接返回，默认 1 小时
	llmCacheTTL := time.Duration(cfg.OCR.LLMCacheTTLSeconds) * time.Second
	if llmCacheTTL <= 0 {
		llmCacheTTL = time.Hour
	}
	var llmSvcI service.LLMService = llmSvc
	llmSvcI = pkgllm.NewCachedLLMService(llmSvcI, llmCacheTTL)
	log.Printf("[LLM cache] enabled, TTL=%v", llmCacheTTL)

	// 初始化 Handler 层
	authHandler := handler.NewAuthHandler(authSvc)
	tenantHandler := handler.NewTenantHandler(tenantSvc)
	categoryHandler := handler.NewCategoryHandler(categorySvc)
	merchantHandler := handler.NewMerchantHandler(merchantSvc)
	transactionHandler := handler.NewTransactionHandler(transactionSvc)
	ocrHandler := handler.NewOCRHandler(ocrSvc, merchantSvc, categorySvc, llmSvcI, store)
	llmHandler := handler.NewLLMHandler(llmSvcI)
	statisticsHandler := handler.NewStatisticsHandler(statisticsSvc)
	exporter := excel.NewExporter()
	exportHandler := handler.NewExportHandler(transactionSvc, exporter, categoryRepo, transactionRepo)

	// 初始化路由
	r := gin.Default()
	r.Use(cors.Default())

	// 复用中间件
	jwtMW := middleware.JWT(cfg)
	tenantMW := middleware.Tenant(tenantMemberRepo)

	api := r.Group("/api")

	// ========== 公开路由（无需认证） ==========
	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login", authHandler.Login)

	// ========== 需要 JWT 的路由 ==========
	auth := api.Group("/")
	auth.Use(jwtMW)
	{
		auth.GET("/tenants", tenantHandler.List)
		auth.POST("/tenants", tenantHandler.Create)
		// 更新租户：只有租户 owner 可以（在 service 层校验）
		auth.PUT("/tenants/:id", tenantHandler.Update)
	}

	// ========== 需要 JWT + 租户成员校验的路由 ==========
	tenant := api.Group("/")
	tenant.Use(jwtMW, tenantMW)
	{
		// 租户成员管理（admin 权限）
		tenant.POST("/tenants/:id/members",
			middleware.Casbin(enforcer, "tenant", "write"),
			tenantHandler.InviteMember)
		tenant.DELETE("/tenants/:id/members/:userId",
			middleware.Casbin(enforcer, "tenant", "write"),
			tenantHandler.RemoveMember)
		tenant.PUT("/tenants/:id/members/:userId",
			middleware.Casbin(enforcer, "tenant", "write"),
			tenantHandler.UpdateMemberRole)
		tenant.GET("/tenants/:id/members",
			tenantHandler.ListMembers)

		// OCR 上传（editor 及以上）
		tenant.POST("/upload/ocr",
			middleware.Casbin(enforcer, "ocr", "write"),
			ocrHandler.Upload)
		// OCR + LLM 分析（editor 及以上）
		tenant.POST("/upload/ocr/analyze",
			middleware.Casbin(enforcer, "ocr", "write"),
			ocrHandler.Analyze)

		// LLM 配置（admin 查看和修改）
		tenant.GET("/llm/config",
			middleware.Casbin(enforcer, "tenant", "read"),
			llmHandler.GetConfig)
		tenant.PUT("/llm/config",
			middleware.Casbin(enforcer, "tenant", "write"),
			llmHandler.SaveConfig)

		// 交易记录批量创建（editor 及以上）
		tenant.POST("/transactions/batch",
			middleware.Casbin(enforcer, "transaction", "write"),
			transactionHandler.BatchCreate)
		// 交易记录列表（所有成员可读）
		tenant.GET("/transactions",
			middleware.Casbin(enforcer, "transaction", "read"),
			transactionHandler.List)
		// 交易记录创建（editor 及以上）
		tenant.POST("/transactions",
			middleware.Casbin(enforcer, "transaction", "write"),
			transactionHandler.Create)
		// 交易记录更新（editor 及以上）
		tenant.PUT("/transactions/:id",
			middleware.Casbin(enforcer, "transaction", "write"),
			transactionHandler.Update)
		// 交易记录删除（editor 及以上）
		tenant.DELETE("/transactions/:id",
			middleware.Casbin(enforcer, "transaction", "write"),
			transactionHandler.Delete)

		// 统计（所有成员可读）
		tenant.GET("/statistics/daily",
			middleware.Casbin(enforcer, "statistics", "read"),
			statisticsHandler.Daily)
		tenant.GET("/statistics/monthly",
			middleware.Casbin(enforcer, "statistics", "read"),
			statisticsHandler.Monthly)
		tenant.GET("/statistics/yearly",
			middleware.Casbin(enforcer, "statistics", "read"),
			statisticsHandler.Yearly)
		tenant.GET("/statistics/range",
			middleware.Casbin(enforcer, "statistics", "read"),
			statisticsHandler.Range)

		// 导出（所有成员可读）
		tenant.GET("/export/excel",
			middleware.Casbin(enforcer, "export", "read"),
			exportHandler.Excel)

		// 导入模板下载（所有成员可读）
		tenant.GET("/import/template",
			exportHandler.DownloadTemplate)

		// 解析 Excel 表头（editor 及以上）
		tenant.POST("/import/parse-headers",
			middleware.Casbin(enforcer, "transaction", "write"),
			exportHandler.ParseHeaders)

		// 导入 Excel（editor 及以上）
		tenant.POST("/import/excel",
			middleware.Casbin(enforcer, "transaction", "write"),
			exportHandler.ImportExcel)

		// 分类读取（所有成员）
		tenant.GET("/categories",
			middleware.Casbin(enforcer, "category", "read"),
			categoryHandler.List)
		// 分类写操作（仅 admin）
		tenant.POST("/categories",
			middleware.Casbin(enforcer, "category", "write"),
			categoryHandler.Create)
		tenant.PUT("/categories/:id",
			middleware.Casbin(enforcer, "category", "write"),
			categoryHandler.Update)
		tenant.DELETE("/categories/:id",
			middleware.Casbin(enforcer, "category", "write"),
			categoryHandler.Delete)

		// 商户读取（所有成员）
		tenant.GET("/merchants",
			middleware.Casbin(enforcer, "merchant", "read"),
			merchantHandler.List)
		// 商户写操作（仅 admin）
		tenant.POST("/merchants",
			middleware.Casbin(enforcer, "merchant", "write"),
			merchantHandler.Create)
		tenant.PUT("/merchants/:id",
			middleware.Casbin(enforcer, "merchant", "write"),
			merchantHandler.Update)
		tenant.DELETE("/merchants/:id",
			middleware.Casbin(enforcer, "merchant", "write"),
			merchantHandler.Delete)
	}

	// 静态文件服务（上传的图片）
	r.Static("/uploads", cfg.Upload.Path)

	// 前端静态文件（内嵌于二进制）
	subFS, err := fs.Sub(webstatic.FS, "static")
	if err != nil {
		log.Fatalf("加载前端资源失败: %v", err)
	}
	fileServer := http.FileServer(http.FS(subFS))
	r.NoRoute(func(c *gin.Context) {
		urlPath := c.Request.URL.Path
		// /api/* 未匹配路由直接 404
		if strings.HasPrefix(urlPath, "/api/") {
			c.Status(http.StatusNotFound)
			return
		}
		// 有内容哈希的 assets 文件：强缓存一年
		if strings.HasPrefix(urlPath, "/assets/") {
			if filePath := strings.TrimPrefix(urlPath, "/"); fileExists(subFS, filePath) {
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		// 其余所有路径（SPA 路由）：返回 index.html，禁止缓存
		data, _ := fs.ReadFile(subFS, "index.html")
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	log.Printf("服务启动，监听端口 :%s", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}

func fileExists(fsys fs.FS, path string) bool {
	f, err := fsys.Open(path)
	if err != nil {
		return false
	}
	st, err := f.Stat()
	f.Close()
	return err == nil && !st.IsDir()
}
