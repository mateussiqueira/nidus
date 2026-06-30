# Docker Compose

Deploy stacks multi-container no StackRun.

## Como usar

1. Crie um `docker-compose.yml` no seu projeto
2. Vá em **Dashboard → Projeto → Deploy → Compose**
3. Cole o YAML e clique **Deploy**

## Exemplo: WordPress + MySQL

```yaml
version: "3.8"

services:
  db:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: rootpass
      MYSQL_DATABASE: wordpress
      MYSQL_USER: wpuser
      MYSQL_PASSWORD: wppass
    volumes:
      - db_data:/var/lib/mysql

  wordpress:
    image: wordpress:latest
    ports:
      - "8080:80"
    environment:
      WORDPRESS_DB_HOST: db
      WORDPRESS_DB_USER: wpuser
      WORDPRESS_DB_PASSWORD: wppass
      WORDPRESS_DB_NAME: wordpress
    volumes:
      - wp_data:/var/www/html
    networks:
      - nidus

volumes:
  db_data:
  wp_data:

networks:
  nidus:
    external: true
```

## Volumes persistentes

Volumes garantem que dados sobrevivam a redeploys e reinicializações.

Declare volumes nomeados no compose e o StackRun gerencia o ciclo de vida automaticamente.
Bancos de dados, uploads e arquivos de configuração devem sempre usar volumes.

## Rede interna

A rede `nidus` permite comunicação entre containers do mesmo projeto e também
entre projetos diferentes que estejam na mesma rede.

Sempre declare `network.*stackrun: external: true` para que o StackRun conecte
os containers à rede interna automaticamente.
