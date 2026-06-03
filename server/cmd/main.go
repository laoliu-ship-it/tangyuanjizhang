package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"fandianjizhang/server/config"
	"fandianjizhang/server/internal/casbin"
	"fandianjizhang/server/internal/dto"
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
	"golang.org/x/crypto/bcrypt"
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
	if err := db.AutoMigrate(
		&model.Tenant{},
		&model.TenantMember{},
		&model.TenantLLMConfig{},
		&model.TenantRole{},
		&model.RolePermission{},
		&model.MediaFile{},
		&model.PlatformAdmin{},
	); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 初始化 Casbin EnforcerPool（按租户从数据库加载权限）
	enforcerPool, err := casbin.NewEnforcerPool(db)
	if err != nil {
		log.Fatalf("初始化 Casbin EnforcerPool 失败: %v", err)
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
	roleRepo := repo.NewTenantRoleRepo(db)
	permRepo := repo.NewRolePermissionRepo(db)
	mediaRepo := repo.NewMediaFileRepo(db)
	platformAdminRepo := repo.NewPlatformAdminRepo(db)
	platformStatsRepo := repo.NewPlatformStatsRepo(db)

	// 初始化 RBAC Service
	rbacSvc := service.NewRBACService(roleRepo, permRepo, enforcerPool)

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
	ocrHandler := handler.NewOCRHandler(ocrSvc, merchantSvc, categorySvc, llmSvcI, store, mediaRepo, cfg.Upload)
	llmHandler := handler.NewLLMHandler(llmSvcI)
	statisticsHandler := handler.NewStatisticsHandler(statisticsSvc)
	exporter := excel.NewExporter()
	exportHandler := handler.NewExportHandler(transactionSvc, exporter, categoryRepo, transactionRepo)
	rbacHandler := handler.NewRBACHandler(rbacSvc)
	platformAdminSvc := service.NewPlatformAdminService(platformAdminRepo, platformStatsRepo, userRepo, cfg)
	platformAdminHandler := handler.NewPlatformAdminHandler(platformAdminSvc)

	// 初始化路由
	r := gin.Default()
	r.Use(cors.Default())

	// 复用中间件
	jwtMW := middleware.JWT(cfg)
	tenantMW := middleware.Tenant(tenantMemberRepo)
	platformJWTMW := middleware.PlatformJWT(cfg)

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
			middleware.Casbin(enforcerPool, "tenant", "write"),
			tenantHandler.InviteMember)
		tenant.DELETE("/tenants/:id/members/:userId",
			middleware.Casbin(enforcerPool, "tenant", "write"),
			tenantHandler.RemoveMember)
		tenant.PUT("/tenants/:id/members/:userId",
			middleware.Casbin(enforcerPool, "tenant", "write"),
			tenantHandler.UpdateMemberRole)
		tenant.GET("/tenants/:id/members",
			tenantHandler.ListMembers)

			// RBAC 角色管理
			tenant.GET("/permissions",
				rbacHandler.ListPermissions)
			tenant.GET("/tenants/:id/roles",
				middleware.Casbin(enforcerPool, "tenant", "read"),
				rbacHandler.ListRoles)
			tenant.POST("/tenants/:id/roles",
				middleware.Casbin(enforcerPool, "tenant", "write"),
				rbacHandler.CreateRole)
			tenant.PUT("/tenants/:id/roles/:roleId",
				middleware.Casbin(enforcerPool, "tenant", "write"),
				rbacHandler.UpdateRole)
			tenant.DELETE("/tenants/:id/roles/:roleId",
				middleware.Casbin(enforcerPool, "tenant", "write"),
				rbacHandler.DeleteRole)

		// OCR 上传（需增改账目权限）
		tenant.POST("/upload/ocr",
			middleware.Casbin(enforcerPool, "transaction", "write"),
			ocrHandler.Upload)
		// OCR + LLM 分析（需增改账目权限）
		tenant.POST("/upload/ocr/analyze",
			middleware.Casbin(enforcerPool, "transaction", "write"),
			ocrHandler.Analyze)

		// LLM 配置（admin 查看和修改）
		tenant.GET("/llm/config",
			middleware.Casbin(enforcerPool, "tenant", "read"),
			llmHandler.GetConfig)
		tenant.PUT("/llm/config",
			middleware.Casbin(enforcerPool, "tenant", "write"),
			llmHandler.SaveConfig)

		// 交易记录批量创建（editor 及以上）
		tenant.POST("/transactions/batch",
			middleware.Casbin(enforcerPool, "transaction", "write"),
			transactionHandler.BatchCreate)
		// 交易记录列表（所有成员可读）
		tenant.GET("/transactions",
			middleware.Casbin(enforcerPool, "transaction", "read"),
			transactionHandler.List)
		// 交易记录创建（editor 及以上）
		tenant.POST("/transactions",
			middleware.Casbin(enforcerPool, "transaction", "write"),
			transactionHandler.Create)
		// 交易记录更新（editor 及以上）
		tenant.PUT("/transactions/:id",
			middleware.Casbin(enforcerPool, "transaction", "write"),
			transactionHandler.Update)
		// 交易记录删除（editor 及以上）
		tenant.DELETE("/transactions/:id",
			middleware.Casbin(enforcerPool, "transaction", "write"),
			transactionHandler.Delete)

		// 统计（所有成员可读）
		tenant.GET("/statistics/daily",
			middleware.Casbin(enforcerPool, "statistics", "read"),
			statisticsHandler.Daily)
		tenant.GET("/statistics/monthly",
			middleware.Casbin(enforcerPool, "statistics", "read"),
			statisticsHandler.Monthly)
		tenant.GET("/statistics/yearly",
			middleware.Casbin(enforcerPool, "statistics", "read"),
			statisticsHandler.Yearly)
		tenant.GET("/statistics/range",
			middleware.Casbin(enforcerPool, "statistics", "read"),
			statisticsHandler.Range)

		// 导出（所有成员可读）
		tenant.GET("/export/excel",
			middleware.Casbin(enforcerPool, "export", "read"),
			exportHandler.Excel)

		// 测试导出 API - 直接返回查询结果用于调试
		tenant.GET("/export/debug", func(c *gin.Context) {
			tenantID := middleware.GetTenantID(c)
			start := c.Query("start_date")
			end := c.Query("end_date")
			txType := c.Query("type")

			if start == "" {
				now := time.Now()
				start = fmt.Sprintf("%d-%02d-01", now.Year(), now.Month())
			}
			if end == "" {
				end = time.Now().Format("2006-01-02")
			}

			filter := dto.TransactionFilter{
				StartDate: start,
				EndDate:   end,
				Type:      txType,
				Page:      1,
				PageSize:  100000,
			}

			// 直接查询数据库
			txs, total, err := transactionRepo.List(c.Request.Context(), tenantID, filter)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			// 返回详细信息
			c.JSON(200, gin.H{
				"tenant_id":    tenantID,
				"filter":       filter,
				"total":        total,
				"count":        len(txs),
				"transactions": txs,
			})
		})

		// 导入模板下载（所有成员可读）
		tenant.GET("/import/template",
			exportHandler.DownloadTemplate)

		// 解析 Excel 表头（editor 及以上）
		tenant.POST("/import/parse-headers",
			middleware.Casbin(enforcerPool, "transaction", "write"),
			exportHandler.ParseHeaders)

		// 导入 Excel（editor 及以上）
		tenant.POST("/import/excel",
			middleware.Casbin(enforcerPool, "transaction", "write"),
			exportHandler.ImportExcel)

		// 分类读取（所有成员）
		tenant.GET("/categories",
			middleware.Casbin(enforcerPool, "category", "read"),
			categoryHandler.List)
		// 分类写操作（仅 admin）
		tenant.POST("/categories",
			middleware.Casbin(enforcerPool, "category", "write"),
			categoryHandler.Create)
		tenant.PUT("/categories/:id",
			middleware.Casbin(enforcerPool, "category", "write"),
			categoryHandler.Update)
		tenant.DELETE("/categories/:id",
			middleware.Casbin(enforcerPool, "category", "write"),
			categoryHandler.Delete)

		// 商户读取（所有成员）
		tenant.GET("/merchants",
			middleware.Casbin(enforcerPool, "merchant", "read"),
			merchantHandler.List)
		// 商户写操作（仅 admin）
		tenant.POST("/merchants",
			middleware.Casbin(enforcerPool, "merchant", "write"),
			merchantHandler.Create)
		tenant.PUT("/merchants/:id",
			middleware.Casbin(enforcerPool, "merchant", "write"),
			merchantHandler.Update)
		tenant.DELETE("/merchants/:id",
			middleware.Casbin(enforcerPool, "merchant", "write"),
			merchantHandler.Delete)
	}

	// ========== 平台管理员路由（独立认证，不受租户中间件影响） ==========
	platform := api.Group("/platform")
	{
		platform.POST("/auth/login", platformAdminHandler.Login)

		platformAuth := platform.Group("/")
		platformAuth.Use(platformJWTMW)
		{
			platformAuth.GET("/dashboard", platformAdminHandler.Dashboard)
			platformAuth.GET("/users", platformAdminHandler.ListUsers)
			platformAuth.GET("/users/:id", platformAdminHandler.GetUserDetail)
		}
	}

	// Seed platform admin from env (must succeed before starting HTTP server)
	ensurePlatformAdmin(db)

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

func ensurePlatformAdmin(db *gorm.DB) {
	email := getEnv("PLATFORM_ADMIN_EMAIL", "admin@fandianjizhang.com")
	password := getEnv("PLATFORM_ADMIN_PASSWORD", "admin123456")
	name := getEnv("PLATFORM_ADMIN_NAME", "Admin")

	var admin model.PlatformAdmin
	result := db.Where("email = ? AND deleted_at IS NULL", email).First(&admin)

	if result.Error == nil {
		log.Printf("[seed] platform admin already exists: %s (skipping)", email)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[seed] ERROR: failed to hash platform admin password: %v", err)
		return
	}

	admin = model.PlatformAdmin{
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
	}
	if err := db.Create(&admin).Error; err != nil {
		log.Printf("[seed] ERROR: failed to create platform admin: %v", err)
	} else {
		log.Printf("[seed] created platform admin: %s (name: %s)", email, name)
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
