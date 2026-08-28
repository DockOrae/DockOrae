package appstore

// 预置应用清单(compose 模板用 Go text/template;password 参数留空自动生成随机值)

var apps = []*App{
	{
		Key: "mysql", Name: "MySQL", Icon: "🐬", Category: "database",
		Description: "MySQL 8.4 — 流行的开源关系型数据库",
		Ports:       []string{"3306"},
		Params: []Param{
			{Key: "port", Label: "port", Type: "number", Default: "3306", Required: true},
			{Key: "root_password", Label: "root_password", Type: "password", Required: true},
		},
		tpl: `services:
  mysql:
    image: mysql:8.4
    container_name: {{.key}}-mysql
    restart: always
    environment:
      MYSQL_ROOT_PASSWORD: "{{.root_password}}"
      TZ: Asia/Shanghai
    ports:
      - "{{.port}}:3306"
    volumes:
      - {{.key}}_data:/var/lib/mysql
volumes:
  {{.key}}_data:
`,
	},
	{
		Key: "postgres", Name: "PostgreSQL", Icon: "🐘", Category: "database",
		Description: "PostgreSQL 16 — 强大的开源对象关系数据库",
		Ports:       []string{"5432"},
		Params: []Param{
			{Key: "port", Label: "port", Type: "number", Default: "5432", Required: true},
			{Key: "postgres_password", Label: "postgres_password", Type: "password", Required: true},
		},
		tpl: `services:
  postgres:
    image: postgres:16-alpine
    container_name: {{.key}}-postgres
    restart: always
    environment:
      POSTGRES_PASSWORD: "{{.postgres_password}}"
      TZ: Asia/Shanghai
    ports:
      - "{{.port}}:5432"
    volumes:
      - {{.key}}_data:/var/lib/postgresql/data
volumes:
  {{.key}}_data:
`,
	},
	{
		Key: "redis", Name: "Redis", Icon: "🔴", Category: "database",
		Description: "Redis 7 — 高性能内存键值数据库",
		Ports:       []string{"6379"},
		Params: []Param{
			{Key: "port", Label: "port", Type: "number", Default: "6379", Required: true},
			{Key: "requirepass", Label: "requirepass", Type: "password"},
		},
		tpl: `services:
  redis:
    image: redis:7-alpine
    container_name: {{.key}}-redis
    restart: always
    command: ["redis-server", "--appendonly", "yes"{{if .requirepass}}, "--requirepass", "{{.requirepass}}"{{end}}]
    ports:
      - "{{.port}}:6379"
    volumes:
      - {{.key}}_data:/data
volumes:
  {{.key}}_data:
`,
	},
	{
		Key: "nginx", Name: "Nginx", Icon: "🌐", Category: "service",
		Description: "Nginx — 高性能 Web 服务器与反向代理",
		Ports:       []string{"80", "443"},
		Params: []Param{
			{Key: "http_port", Label: "http_port", Type: "number", Default: "80", Required: true},
			{Key: "https_port", Label: "https_port", Type: "number", Default: "443"},
		},
		tpl: `services:
  nginx:
    image: nginx:alpine
    container_name: {{.key}}-nginx
    restart: always
    ports:
      - "{{.http_port}}:80"
{{if .https_port}}      - "{{.https_port}}:443"
{{end}}    volumes:
      - {{.key}}_html:/usr/share/nginx/html
      - {{.key}}_conf:/etc/nginx/conf.d
volumes:
  {{.key}}_html:
  {{.key}}_conf:
`,
	},
	{
		Key: "gitea", Name: "Gitea", Icon: "🍵", Category: "service",
		Description: "Gitea 1.27 — 轻量自托管 Git 服务",
		Ports:       []string{"3000", "222"},
		Params: []Param{
			{Key: "http_port", Label: "http_port", Type: "number", Default: "3000", Required: true},
			{Key: "ssh_port", Label: "ssh_port", Type: "number", Default: "222"},
		},
		tpl: `services:
  server:
    image: docker.gitea.com/gitea:1.27.2
    container_name: {{.key}}
    restart: always
    environment:
      USER_UID: "1000"
      USER_GID: "1000"
    volumes:
      - {{.key}}_data:/data
    ports:
      - "{{.http_port}}:3000"
{{if .ssh_port}}      - "{{.ssh_port}}:22"
{{end}}
volumes:
  {{.key}}_data:
`,
	},
	{
		Key: "portainer", Name: "Portainer", Icon: "🐳", Category: "tools",
		Description: "Portainer CE — 轻量 Docker 管理界面",
		Ports:       []string{"9000"},
		Params: []Param{
			{Key: "port", Label: "port", Type: "number", Default: "9000", Required: true},
		},
		tpl: `services:
  portainer:
    image: portainer/portainer-ce:latest
    container_name: {{.key}}
    restart: always
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - {{.key}}_data:/data
    ports:
      - "{{.port}}:9000"
volumes:
  {{.key}}_data:
`,
	},
	{
		Key: "npm", Name: "Nginx Proxy Manager", Icon: "🛡️", Category: "tools",
		Description: "Nginx Proxy Manager — 可视化反向代理与 SSL 管理",
		Ports:       []string{"80", "443", "8181"},
		Params: []Param{
			{Key: "http_port", Label: "http_port", Type: "number", Default: "80", Required: true},
			{Key: "https_port", Label: "https_port", Type: "number", Default: "443"},
			{Key: "admin_port", Label: "admin_port", Type: "number", Default: "8181"},
		},
		tpl: `services:
  npm:
    image: jc21/nginx-proxy-manager:latest
    container_name: {{.key}}
    restart: always
    environment:
      TZ: Asia/Shanghai
    ports:
      - "{{.http_port}}:80"
{{if .https_port}}      - "{{.https_port}}:443"
{{end}}{{if .admin_port}}      - "{{.admin_port}}:81"
{{end}}    volumes:
      - {{.key}}_data:/data
      - {{.key}}_letsencrypt:/etc/letsencrypt
volumes:
  {{.key}}_data:
  {{.key}}_letsencrypt:
`,
	},
	{
		Key: "nextcloud", Name: "Nextcloud", Icon: "☁️", Category: "service",
		Description: "Nextcloud — 自托管云盘与协作平台",
		Ports:       []string{"8080"},
		Params: []Param{
			{Key: "port", Label: "port", Type: "number", Default: "8080", Required: true},
			{Key: "admin_password", Label: "admin_password", Type: "password", Required: true},
		},
		tpl: `services:
  app:
    image: nextcloud:29-apache
    container_name: {{.key}}
    restart: always
    environment:
      NEXTCLOUD_ADMIN_USER: admin
      NEXTCLOUD_ADMIN_PASSWORD: "{{.admin_password}}"
      TZ: Asia/Shanghai
    ports:
      - "{{.port}}:80"
    volumes:
      - {{.key}}_data:/var/www/html
volumes:
  {{.key}}_data:
`,
	},
}
