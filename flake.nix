{
  description = "Symbiont — Neptune Apex local dashboard dev environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        # Shared build config for Go binaries (go-duckdb needs CGO + Arrow C headers)
        goBuildBase = {
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-N61ynzRCwzIexwsKyXvGSPF5B5HsDitaQ9jWMWSoAxM=";
          proxyVendor = true; # go-duckdb bundles libduckdb.a in deps/ — must preserve non-Go files
          nativeBuildInputs = [ pkgs.pkg-config ];
          buildInputs = [ pkgs.arrow-cpp ];
        };
      in
      {
        packages.poller = pkgs.buildGoModule (goBuildBase // {
          pname = "symbiont-poller";
          subPackages = [ "cmd/poller" ];
          meta.description = "Symbiont Apex poller binary";
        });

        packages.api = pkgs.buildGoModule (goBuildBase // {
          pname = "symbiont-api";
          subPackages = [ "cmd/api" ];
          meta.description = "Symbiont API server binary";
        });

        devShells.default = pkgs.mkShell {
          name = "symbiont";

          buildInputs = with pkgs; [
            # Go toolchain (1.23+)
            go_1_24

            # Database CLIs
            duckdb
            sqlite

            # Go IDE support
            gopls
            delve
            golangci-lint

            # Frontend
            nodejs_22
            nodePackages.typescript

            # Dev utilities
            gotools        # goimports, etc.
            go-task        # Taskfile runner (optional)
            jq             # JSON pretty-print for log inspection
            curl           # API testing

            # Load .env files for running binaries
            # Use: godotenv -f .env ./poller
            # (godotenv binary from the go package)
          ];

          shellHook = ''
            echo "Symbiont dev shell ready."
            echo "Go: $(go version)"
            echo "DuckDB: $(duckdb --version 2>/dev/null || echo 'not found')"
            echo "SQLite: $(sqlite3 --version 2>/dev/null || echo 'not found')"
            echo ""
            echo "Quick commands:"
            echo "  go build ./...         Build all binaries"
            echo "  go test ./...          Run all tests"
            echo "  go run ./cmd/poller    Run poller (requires .env)"
            echo "  go run ./cmd/api       Run API server (requires .env)"
          '';
        };

        # NixOS module for systemd services (used in configuration.nix)
        nixosModules.default = { config, lib, pkgs, ... }:
          let
            cfg = config.services.symbiont;
          in
          {
            options.services.symbiont = {
              enable = lib.mkEnableOption "Symbiont Neptune Apex dashboard";

              pollerPackage = lib.mkOption {
                type = lib.types.package;
                description = "The symbiont-poller binary package.";
              };

              apiPackage = lib.mkOption {
                type = lib.types.package;
                description = "The symbiont-api binary package.";
              };

              envFile = lib.mkOption {
                type = lib.types.str;
                default = "/etc/symbiont/env";
                description = "Path to environment file with secrets.";
              };

              dataDir = lib.mkOption {
                type = lib.types.str;
                default = "/var/lib/symbiont";
                description = "Directory for database files and backups.";
              };
            };

            config = lib.mkIf cfg.enable {
              users.users.symbiont = {
                isSystemUser = true;
                group = "symbiont";
                home = cfg.dataDir;
                createHome = true;
                description = "Symbiont service user";
              };

              users.groups.symbiont = {};

              systemd.services.symbiont-poller = {
                description = "Symbiont Apex Poller";
                after = [ "network.target" ];
                wantedBy = [ "multi-user.target" ];

                serviceConfig = {
                  ExecStart = "${cfg.pollerPackage}/bin/poller";
                  EnvironmentFile = cfg.envFile;
                  User = "symbiont";
                  Group = "symbiont";
                  StateDirectory = "symbiont";
                  Restart = "always";
                  RestartSec = "5s";

                  # Hardening
                  PrivateTmp = true;
                  NoNewPrivileges = true;
                  ProtectSystem = "strict";
                  ReadWritePaths = [ cfg.dataDir ];
                };
              };

              systemd.services.symbiont-api = {
                description = "Symbiont API Server";
                after = [ "network.target" "symbiont-poller.service" ];
                wantedBy = [ "multi-user.target" ];

                serviceConfig = {
                  ExecStart = "${cfg.apiPackage}/bin/api";
                  EnvironmentFile = cfg.envFile;
                  User = "symbiont";
                  Group = "symbiont";
                  StateDirectory = "symbiont";
                  Restart = "always";
                  RestartSec = "5s";

                  # Hardening
                  PrivateTmp = true;
                  NoNewPrivileges = true;
                  ProtectSystem = "strict";
                  ReadWritePaths = [ cfg.dataDir ];
                };
              };
            };
          };
      }
    );
}
