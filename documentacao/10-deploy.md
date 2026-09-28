# Deploy e Producao

## Servidor

O sistema roda em Oracle Cloud Always Free (ARM). Dados de acesso (IP, SSH key, credenciais) estao em `PRODUCAO.md` na raiz do projeto (ignorado pelo git).

## Containers

O `docker-compose.prod.yml` sobe 3 containers:

| Container | Imagem | Porta | Descricao |
|-----------|--------|-------|-----------|
| app | Build local (Dockerfile) | 80:8080 | Aplicacao principal |
| postgres | postgres:16-alpine | — | Banco de dados |
| pluggy-service | Build local (pluggy-service/Dockerfile) | — | Microservico Pluggy |

## Variaveis de producao (.env.prod)

Crie a partir de `.env.prod.example`:

```
DB_USER=fincontrol
DB_PASS=<senha-segura>
DB_NAME=fincontrol
SESSION_SECRET=<secret-aleatorio>
PLUGGY_CLIENT_ID=<client-id-pluggy>
PLUGGY_CLIENT_SECRET=<client-secret-pluggy>
```

## Deploy

```bash
# 1. Copiar arquivos para servidor
scp -i <SSH_KEY_PATH> -r . <USER>@<SERVER_IP>:/home/<USER>/finBertoldi/

# 2. Rebuildar containers
ssh -i <SSH_KEY_PATH> <USER>@<SERVER_IP> "cd /home/<USER>/finBertoldi && docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build"
```

## Backup do banco

```bash
ssh -i <SSH_KEY_PATH> <USER>@<SERVER_IP> \
  "docker exec finbertoldi-postgres-1 pg_dump -U fincontrol fincontrol > /home/<USER>/backup_fincontrol_\$(date +%Y%m%d_%H%M%S).sql"
```

## Comandos uteis

```bash
# Logs
docker logs finbertoldi-app-1 --tail 50 -f
docker logs finbertoldi-pluggy-service-1 --tail 50 -f

# Banco
docker exec -it finbertoldi-postgres-1 psql -U fincontrol -d fincontrol

# Restart
docker compose -f docker-compose.prod.yml --env-file .env.prod restart

# Status
docker compose -f docker-compose.prod.yml ps
```
