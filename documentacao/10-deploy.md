# Deploy e Producao

## Servidor

O sistema roda em Oracle Cloud Always Free (ARM/aarch64). Dados de acesso (IP, SSH key, credenciais) estao em `PRODUCAO.md` na raiz do projeto (ignorado pelo git).

A arquitetura ARM significa que o build Docker compila binarios para `linux/arm64`. Isso e feito automaticamente no servidor (build local), mas e significativamente mais lento que em x86.

## Containers

O `docker-compose.prod.yml` sobe 3 containers:

| Container | Imagem | Porta | Descricao |
|-----------|--------|-------|-----------|
| app | Build local (Dockerfile) | 80:8080 | Aplicacao principal Go |
| postgres | postgres:16-alpine | — | Banco de dados (volume persistente `pgdata`) |
| pluggy-service | Build local (pluggy-service/Dockerfile) | 8081 (interno) | Microservico Pluggy |

## Variaveis de producao (.env.prod)

Crie a partir de `.env.prod.example`:

```
DB_USER=fincontrol
DB_PASS=<senha-segura>
DB_NAME=fincontrol
SESSION_SECRET=<secret-aleatorio>
PLUGGY_CLIENT_ID=<client-id-pluggy>       # opcional, pode configurar pela UI
PLUGGY_CLIENT_SECRET=<client-secret-pluggy> # opcional, pode configurar pela UI
```

> `PLUGGY_CLIENT_ID` e `PLUGGY_CLIENT_SECRET` podem ficar vazios. As credenciais Pluggy sao configuradas pela tela de admin (Cadastros > Integracao Bancaria) e salvas na tabela `integracoes_config`.

---

## Processo de deploy completo

### Passo 1 — Backup do banco (OBRIGATORIO antes de deploy)

Sempre faca backup antes de qualquer deploy. O dump e salvo no home do servidor.

```bash
ssh -i <SSH_KEY_PATH> <USER>@<SERVER_IP> \
  "docker exec finbertoldi-postgres-1 pg_dump -U fincontrol fincontrol > /home/<USER>/backup_fincontrol_\$(date +%Y%m%d_%H%M%S).sql"
```

Para baixar o backup para a maquina local:

```bash
# Listar backups no servidor
ssh -i <SSH_KEY_PATH> <USER>@<SERVER_IP> "ls -lh /home/<USER>/backup_fincontrol_*.sql"

# Baixar o mais recente
scp -i <SSH_KEY_PATH> <USER>@<SERVER_IP>:/home/<USER>/backup_fincontrol_XXXXXXXX_XXXXXX.sql ./backups/
```

### Passo 2 — Enviar arquivos para o servidor

```bash
scp -i <SSH_KEY_PATH> -r . <USER>@<SERVER_IP>:/home/<USER>/finBertoldi/
```

#### Problema conhecido: permissao da pasta .git

O `scp -r` copia a pasta `.git/` junto, e os objetos Git tem permissao 444 (somente leitura). Em deploys subsequentes, o scp falha com `Permission denied` ao tentar sobrescrever esses arquivos.

**Solucao:** Remover a pasta `.git` no servidor (ela nao e necessaria em producao):

```bash
ssh -i <SSH_KEY_PATH> <USER>@<SERVER_IP> \
  "sudo chmod -R u+w /home/<USER>/finBertoldi/.git/ && sudo rm -rf /home/<USER>/finBertoldi/.git/"
```

**Prevencao:** Usar `rsync` com `--exclude .git` em vez de `scp`:

```bash
rsync -avz --exclude '.git' --exclude 'backups' --exclude 'Oracle' \
  -e "ssh -i <SSH_KEY_PATH>" \
  . <USER>@<SERVER_IP>:/home/<USER>/finBertoldi/
```

> O `rsync` tambem e mais rapido pois envia apenas arquivos alterados.

### Passo 3 — Build e restart dos containers

```bash
ssh -i <SSH_KEY_PATH> <USER>@<SERVER_IP> \
  "cd /home/<USER>/finBertoldi && docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build"
```

#### Tempo de build

O servidor ARM (Oracle Always Free) e lento para compilar Go. Tempos tipicos:

| Etapa | Tempo estimado |
|-------|---------------|
| Download de dependencias Go | 2-5 min |
| Build do app (fincontrol) | 3-8 min |
| Build do pluggy-service | 2-5 min |
| **Total (primeiro build)** | **8-15 min** |
| **Total (rebuild com cache)** | **3-8 min** |

> O build roda em background no servidor. Nao feche o terminal SSH — se precisar, use `nohup` ou `screen`.

#### Warnings esperados

Os seguintes warnings sao normais e nao indicam erro:

```
level=warning msg="The \"PLUGGY_CLIENT_ID\" variable is not set. Defaulting to a blank string."
level=warning msg="The \"PLUGGY_CLIENT_SECRET\" variable is not set. Defaulting to a blank string."
```

> Aparecem quando as credenciais Pluggy nao estao no `.env.prod`. Nao afetam o funcionamento — as credenciais sao lidas do banco via UI de admin.

### Passo 4 — Verificar o deploy

```bash
# Status dos containers (todos devem estar "Up")
ssh -i <SSH_KEY_PATH> <USER>@<SERVER_IP> \
  "cd /home/<USER>/finBertoldi && docker compose -f docker-compose.prod.yml ps"

# Logs do app (verificar se migracoes rodaram e servidor iniciou)
ssh -i <SSH_KEY_PATH> <USER>@<SERVER_IP> \
  "docker logs finbertoldi-app-1 --tail 20"

# Logs do pluggy-service
ssh -i <SSH_KEY_PATH> <USER>@<SERVER_IP> \
  "docker logs finbertoldi-pluggy-service-1 --tail 20"
```

**Saida esperada do app:**
```
[migrate] v11 (transacoes_banco_e_pluggy_items) aplicada
[migrate] v12 (integracoes_config) aplicada
Servidor iniciado em http://localhost:8080
```

**Saida esperada do pluggy-service:**
```
Pluggy service iniciado em :8081
[scheduler] sync a cada 6h0m0s
```

---

## Problemas comuns e solucoes

### 1. Build falha com erro de versao do Go

```
go: go.mod requires go >= X.Y.Z (running go A.B.C; GOTOOLCHAIN=local)
```

**Causa:** O `go.mod` de algum modulo (app ou pluggy-service) tem versao de Go mais alta que a imagem Docker (`golang:1.24-alpine`).

**Solucao:** Alinhar a versao no `go.mod` com a imagem Docker:

```bash
# Verificar versao nos Dockerfiles
grep "FROM golang" Dockerfile pluggy-service/Dockerfile

# Ajustar go.mod para a mesma versao
# Em go.mod: go 1.24.0
# Em pluggy-service/go.mod: go 1.24.0
```

### 2. Container reiniciando em loop

```bash
# Ver o motivo
docker logs finbertoldi-app-1 --tail 50

# Causa comum: banco nao esta pronto
# O healthcheck do postgres resolve isso, mas em primeiro deploy pode demorar
docker compose -f docker-compose.prod.yml --env-file .env.prod restart app
```

### 3. Porta 80 ja em uso

```bash
# Verificar o que esta na porta 80
sudo lsof -i :80
# ou
sudo netstat -tlnp | grep :80

# Se for outro container antigo
docker stop <container_id> && docker rm <container_id>
```

### 4. Espaco em disco

O servidor Oracle Always Free tem disco limitado. Limpar imagens Docker antigas:

```bash
# Ver uso de disco do Docker
docker system df

# Limpar imagens e containers orfaos
docker system prune -f

# Limpar tudo (CUIDADO: remove volumes nao usados tambem)
# docker system prune -a --volumes
```

### 5. Permissao negada no scp

Ver secao "Problema conhecido: permissao da pasta .git" acima.

---

## Deploy parcial (sem rebuild)

Para mudancas que nao exigem recompilacao (CSS, templates, arquivos estaticos):

```bash
# Apenas CSS
scp -i <SSH_KEY_PATH> static/app.css <USER>@<SERVER_IP>:/home/<USER>/finBertoldi/static/app.css

# Templates (precisa restart do container para recarregar)
scp -i <SSH_KEY_PATH> -r templates/ <USER>@<SERVER_IP>:/home/<USER>/finBertoldi/templates/
ssh -i <SSH_KEY_PATH> <USER>@<SERVER_IP> \
  "cd /home/<USER>/finBertoldi && docker compose -f docker-compose.prod.yml --env-file .env.prod restart app"
```

> **Nota:** Templates sao embarcados no binario Go em build. Mudancas em templates SIM exigem rebuild. O comando acima so funciona se os templates forem lidos do disco em runtime. Verifique o Dockerfile — se ele copia `templates/` separadamente, funciona sem rebuild.

---

## Dockerfiles

### App principal (Dockerfile)

Multi-stage build: compila em `golang:1.24-alpine`, roda em `alpine:3.20`.

```dockerfile
FROM golang:1.24-alpine AS builder    # Stage de compilacao
WORKDIR /app
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o fincontrol .

FROM alpine:3.20                      # Stage de runtime (imagem minima)
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/fincontrol . # Binario compilado
COPY templates/ ./templates/          # Templates HTML
COPY static/    ./static/             # CSS, JS, imagens
ENV TZ=America/Sao_Paulo
EXPOSE 8080
CMD ["./fincontrol"]
```

### Pluggy Service (pluggy-service/Dockerfile)

Mesmo padrao multi-stage:

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download                   # Cache de dependencias
COPY . .
RUN CGO_ENABLED=0 go build -o pluggy-service .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/pluggy-service .
EXPOSE 8081
CMD ["./pluggy-service"]
```

---

## Comandos uteis

```bash
# Logs em tempo real
docker logs finbertoldi-app-1 --tail 50 -f
docker logs finbertoldi-pluggy-service-1 --tail 50 -f

# Acessar banco
docker exec -it finbertoldi-postgres-1 psql -U fincontrol -d fincontrol

# Restart sem rebuild
docker compose -f docker-compose.prod.yml --env-file .env.prod restart

# Restart so um container
docker compose -f docker-compose.prod.yml --env-file .env.prod restart app

# Status
docker compose -f docker-compose.prod.yml ps

# Ver uso de recursos
docker stats --no-stream
```
