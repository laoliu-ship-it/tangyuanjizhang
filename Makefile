REMOTE      := nana@127.0.0.1
SSH_PORT    := 3312
SSH         := ssh -p $(SSH_PORT) $(REMOTE)
RSYNC       := rsync -az -e "ssh -p $(SSH_PORT)"
REMOTE_DIR  := /home/nana/fandianjizhang
FNOS_SRC    := $(CURDIR)/fnos
APP_VERSION := $(shell grep '^version' fnos/manifest | awk '{print $$3}')
FPK_NAME    := fandianjizhang.fpk

.PHONY: all build deploy fnos-build fnos-deploy clean

# 一键：构建 + 打包 + 发布
all: build fnos-deploy

# ── 应用构建 ─────────────────────────────────────────────────
build:
	@echo "==> 1. 构建前端"
	cd web && npm run build

	@echo "==> 2. 写入 Go embed 目录"
	rm -rf server/web/static && mkdir -p server/web/static
	cp -r web/dist/. server/web/static/

	@echo "==> 3. 交叉编译 Go (linux/amd64)"
	cd server && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -mod=vendor -o $(CURDIR)/fnos/app/server ./cmd/main.go

	@echo "==> 构建完成，二进制已写入 fnos/app/server"

# ── fpk 打包（在 NAS 上执行，因为 fnpack 在那里）───────────────
fnos-build: build
	@echo "==> 4. 上传 fnos/ 到 NAS"
	$(SSH) "mkdir -p $(REMOTE_DIR)/fnos-build"
	$(RSYNC) $(FNOS_SRC)/ $(REMOTE):$(REMOTE_DIR)/fnos-build/fandianjizhang/

	@echo "==> 5. 在 NAS 上打包 fpk"
	$(SSH) "cd $(REMOTE_DIR)/fnos-build && fnpack build --directory fandianjizhang && ls -lh *.fpk"

	@echo "==> 6. 下载 fpk 到本地 dist/"
	mkdir -p $(CURDIR)/dist
	scp -P $(SSH_PORT) $(REMOTE):$(REMOTE_DIR)/fnos-build/$(FPK_NAME) $(CURDIR)/dist/ || true

	@echo "==> fpk 已保存至 dist/$(FPK_NAME)"

# ── 安装到飞牛 ───────────────────────────────────────────────
fnos-deploy: fnos-build
	@echo "==> 7. 安装 fpk 到飞牛（需要 sudo 密码）"
	ssh -t -p $(SSH_PORT) $(REMOTE) "sudo appcenter-cli install-fpk $(REMOTE_DIR)/fnos-build/$(FPK_NAME)"
	@echo "==> 安装完成，访问 http://<NAS-IP>:8090"

# ── 普通部署（systemd 模式，原有流程）────────────────────────
deploy: build
	@echo "==> 普通部署（systemd）"
	bash deploy.sh

clean:
	rm -f fnos/app/server
	rm -rf server/web/static dist/
