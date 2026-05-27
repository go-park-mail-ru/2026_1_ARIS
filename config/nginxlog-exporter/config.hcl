listen {
  port = 4040
  address = "0.0.0.0"
  metrics_endpoint = "/metrics"
}

namespace "gateway" {
  format = "$remote_addr - $remote_user [$time_local] \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\" \"$http_x_forwarded_for\" $request_length $request_time $upstream_response_time \"$upstream_status\" \"$upstream_addr\""

  source {
    files = ["/var/log/nginx/access.log"]
  }

  labels {
    app = "aris"
  }

  histogram_buckets = [.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10]

  relabel "gateway_route" {
    from = "request"
    split = 2
    separator = " "

    match "^/$" {
      replacement = "/"
    }

    match "^/health$" {
      replacement = "/health"
    }

    match "^/api/auth(?:/.*)?$" {
      replacement = "/api/auth"
    }

    match "^/api/(?:users|profile|friends|settings)(?:/.*)?$" {
      replacement = "/api/user"
    }

    match "^/api/(?:public/feed|public/popular-posts|feed|posts|post)(?:/.*)?$" {
      replacement = "/api/post"
    }

    match "^/api/media(?:/.*)?$" {
      replacement = "/api/media"
    }

    match "^/api/(?:chats|presence|sticker-packs)(?:/.*)?$" {
      replacement = "/api/chat"
    }

    match "^/api/support(?:/.*)?$" {
      replacement = "/api/support"
    }

    match "^/api/communities(?:/.*)?$" {
      replacement = "/api/community"
    }

    match "^/api/search(?:\\?.*)?$" {
      replacement = "/api/search"
    }

    match "^/api/games(?:/.*)?$" {
      replacement = "/api/game"
    }

    match "^/ws/support(?:/.*)?$" {
      replacement = "/ws/support"
    }

    match "^/ws/games(?:/.*)?$" {
      replacement = "/ws/game"
    }

    match "^/ws(?:/.*)?$" {
      replacement = "/ws/chat"
    }

    match "^/media(?:/.*)?$" {
      replacement = "/media"
    }

    match "^/metrics(?:/.*)?$" {
      replacement = "/metrics"
    }

    match "^/prometheus(?:/.*)?$" {
      replacement = "/prometheus"
    }

    match "^/(?:assets/.*|.*\\.(?:js|css|png|jpg|jpeg|gif|svg|webp|ico|woff2?))$" {
      replacement = "/static"
    }

    match "^.*$" {
      replacement = "other"
    }
  }

  relabel "upstream_status" {
    from = "upstream_status"
  }

  relabel "upstream_addr" {
    from = "upstream_addr"
  }
}
