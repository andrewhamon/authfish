{
  protectWithAuthfish = config: originalVhost:
    let
      originalExtraConfig = originalVhost.extraConfig or "";
      newExtraConfig = ''
        auth_request /auth_request;
        error_page 401 /authfish_login;
      '';
      combinedExtraConfig = newExtraConfig + originalExtraConfig;

      proxyUrl = "http://localhost:${toString config.services.authfish.port}";

      originalLocations = originalVhost.locations or { };
      combinedLocations = originalLocations // {
        "/auth_request" = {
          proxyPass = "${proxyUrl}/check";
          extraConfig = ''
            internal;
            proxy_set_header X-Original-URL $scheme://$http_host$request_uri;
          '';
        };

        "/authfish_login" = {
          proxyPass = "${proxyUrl}/login";
          extraConfig = ''
            auth_request off;
            proxy_set_header X-Authfish-Login-Path /authfish_login;
            proxy_set_header X-Original-URL $scheme://$http_host$request_uri;
          '';
        };
      };
    in
    originalVhost // {
      extraConfig = combinedExtraConfig;
      locations = combinedLocations;
    };
}
