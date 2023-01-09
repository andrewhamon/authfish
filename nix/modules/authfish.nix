{ lib, pkgs, config, ... }:
with lib;
let
  cfg = config.services.authfish;
in
{
  options = {
    services.authfish = {
      enable = mkEnableOption "Authfish";

      dataDir = mkOption {
        type = types.str;
        default = "/var/lib/authfish/";
        description = "The directory where Authfish stores its data files.";
      };

      port = mkOption {
        type = types.int;
        default = 8478;
      };

      user = mkOption {
        type = types.str;
        default = "authfish";
      };

      group = mkOption {
        type = types.str;
        default = "authfish";
      };

      domains = mkOption {
        type = types.listOf types.str;
      };

      virtualHostName = mkOption {
        type = types.str;
      };

      enableNginx = mkOption {
        type = types.bool;
      };

      enableACME = mkOption {
        type = types.bool;
      };

      forceSSL = mkOption {
        type = types.bool;
      };
    };
  };

  config = mkIf cfg.enable {
    environment.systemPackages = with pkgs; [
      authfish
    ];

    systemd.services.authfish = {
      description = "Authfish";
      after = [ "network.target" ];
      wantedBy = [ "multi-user.target" ];

      serviceConfig = {
        Type = "simple";
        User = cfg.user;
        Group = cfg.group;
        ExecStart = "${pkgs.authfish}/bin/authfish server --port ${toString cfg.port} --domain ${strings.concatStringsSep "," cfg.domains}";
        Restart = "on-failure";
      };
    };

    users.users = mkIf (cfg.user == "authfish") {
      authfish = {
        isNormalUser = true;
        home = cfg.dataDir;
        group = cfg.group;
      };
    };

    users.groups = mkIf (cfg.group == "authfish") {
      authfish = { };
    };

    services.nginx.virtualHosts = {
      "${cfg.virtualHostName}" = {
        enableACME = cfg.enableACME;
        forceSSL = cfg.forceSSL;
        locations."/" = {
          proxyPass = "http://localhost:${toString cfg.port}";
          extraConfig = ''
            proxy_set_header X-Original-URL $scheme://$http_host$request_uri;
          '';
        };
      };
    };
  };
}
