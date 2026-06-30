module.exports = {
  apps: [{
    name: "stackrun-proxy",
    cwd: "/root/stackrun/apps/proxy",
    script: "./stackrun-proxy",
    env: {
      DATABASE_URL: "postgres://stackrun:stackrun_dev_2026@localhost:5432/stackrun?sslmode=disable",
      PROXY_PORT: "8080",
      DASHBOARD_URL: "http://localhost:3000",
    },
  }]
}
