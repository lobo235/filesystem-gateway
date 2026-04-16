job "filesystem-gateway" {
  node_pool   = "default"
  datacenters = ["dc1"]
  type        = "service"

  update {
    max_parallel     = 1
    health_check     = "checks"
    min_healthy_time = "15s"
    healthy_deadline = "5m"
    auto_revert      = true
  }

  group "filesystem-gateway" {
    count = 1

    network {
      port "http" {
        to = 8080
      }
    }

    service {
      name     = "filesystem-gateway"
      port     = "http"
      provider = "consul"
      tags = [
        "traefik.enable=true",
        "traefik.http.routers.filesystem-gateway.rule=Host(`filesystem-gateway.example.com`)",
        "traefik.http.routers.filesystem-gateway.entrypoints=websecure",
        "traefik.http.routers.filesystem-gateway.tls=true",
      ]

      check {
        type     = "http"
        path     = "/health"
        port     = "http"
        interval = "30s"
        timeout  = "5s"

        check_restart {
          limit = 3
          grace = "30s"
        }
      }
    }

    restart {
      attempts = 3
      interval = "2m"
      delay    = "15s"
      mode     = "fail"
    }

    vault {
      cluster     = "default"
      change_mode = "restart"
    }

    task "filesystem-gateway" {
      driver = "docker"

      config {
        image      = "gitea.example.com/netlobo/filesystem-gateway:latest"
        force_pull = true
        ports      = ["http"]
        volumes = [
          "/path/to/data:/mnt/data",
          "/path/to/status:/data",
        ]
      }

      template {
        data = <<EOF
{{ with secret "kv/data/nomad/default/filesystem-gateway" }}
GATEWAY_API_KEY={{ .Data.data.gateway_api_key }}
{{ end }}
EOF
        destination = "secrets/filesystem-gateway.env"
        env         = true
      }

      env {
        PORT          = "8080"
        LOG_LEVEL     = "info"
        NFS_BASE_PATH = "/mnt/data"
        DATA_DIR      = "/data"
      }

      template {
        data        = "{{ with nomadVar \"nomad/jobs/filesystem-gateway\" }}{{ .image_digest }}{{ end }}"
        destination = "local/deploy-trigger"
        change_mode = "restart"
      }

      resources {
        cpu    = 200
        memory = 128
      }

      kill_timeout = "35s"
    }
  }
}
