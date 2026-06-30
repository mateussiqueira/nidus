module.exports = {
  apps: [{
    name: "nidus-proxy",
    cwd: "/root/nidus/apps/proxy",
    script: "./nidus-proxy",
    env: {
      DATABASE_URL: "postgres://nidus:nidus_dev_2026@localhost:5432/nidus?sslmode=disable",
      PROXY_PORT: "8080",
      DASHBOARD_URL: "http://localhost:3000",
    },
  }]
}
