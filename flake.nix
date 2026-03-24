{
  description = "Symbiont — Neptune Apex local dashboard";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        # ── Frontend (React/Vite) ────────────────────────────────────────────
        # Builds frontend/dist/. Used as an input to the source build below.
        frontend = pkgs.buildNpmPackage {
          pname = "symbiont-frontend";
          version = "0.1.0";
          src = ./frontend;
          npmDepsHash = "sha256-xD/jLzxc5ALkq3y/Eila4uADtqBqN3/VX58PPBXJcx8=";
          installPhase = ''
            runHook preInstall
            cp -r dist $out
            runHook postInstall
          '';
        };

        # ── Go build base (shared between source packages) ───────────────────
        # go-duckdb needs CGO + Apache Arrow C headers.
        goBuildBase = {
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-SLaRU9M6NwnoYowyojtOYml0i+XNVhkt22uvqBrYFi4=";
          proxyVendor = true; # go-duckdb bundles libduckdb.a in deps/ — preserve non-Go files
          nativeBuildInputs = [ pkgs.pkg-config ];
          buildInputs = [ pkgs.arrow-cpp ];
        };

        # ── Path A: build from source ────────────────────────────────────────
        # Single binary with embedded frontend. Built with -tags release.
        # Update vendorHash if go.sum changes; update frontend npmDepsHash if
        # package-lock.json changes.
        symbiont = pkgs.buildGoModule (goBuildBase // {
          pname = "symbiont";
          subPackages = [ "cmd/symbiont" ];
          tags = [ "release" ];
          # Copy the compiled frontend assets into the source tree before the
          # Go build so //go:embed all:dist can find them.
          preBuild = ''
            cp -r ${frontend} frontend/dist
          '';
          meta = {
            description = "Symbiont Neptune Apex dashboard — single binary";
            mainProgram = "symbiont";
          };
        });

        # ── Path B: pre-built binary from GitHub Releases ───────────────────
        # Faster for production installs — no local compilation needed.
        # To upgrade: bump version + hash after tagging a new release.
        #
        # Get the new hash with:
        #   gh release download vX.Y.Z --pattern "symbiont-linux-amd64.tar.gz" --dir /tmp
        #   nix-prefetch-url file:///tmp/symbiont-linux-amd64.tar.gz
        #   nix hash convert --hash-algo sha256 --to sri <base32-hash>
        symbiont-bin = pkgs.stdenv.mkDerivation {
          pname = "symbiont-bin";
          version = "0.1.0"; # <── bump on upgrade
          src = pkgs.fetchurl {
            url = "https://github.com/kjaebker/Symbiont/releases/download/v0.1.0/symbiont-linux-amd64.tar.gz";
            hash = "sha256-ufKv5+uK60l81z7QV5zu42Tarefs75os8JPqUgiu/f0="; # <── update on upgrade
          };
          sourceRoot = ".";
          # Patch the binary's interpreter and rpath to use Nix store paths.
          # The binary is dynamically linked (CGO/DuckDB) so it won't run on
          # NixOS without this.
          nativeBuildInputs = [ pkgs.autoPatchelfHook ];
          buildInputs = [
            pkgs.stdenv.cc.cc.lib  # libstdc++
            pkgs.zlib
          ];
          installPhase = ''
            runHook preInstall
            install -Dm755 symbiont $out/bin/symbiont
            runHook postInstall
          '';
          meta = {
            description = "Symbiont Neptune Apex dashboard (pre-built)";
            mainProgram = "symbiont";
          };
        };

      in
      {
        # ── Packages ─────────────────────────────────────────────────────────
        packages = {
          inherit symbiont symbiont-bin frontend;
          default = symbiont;
        };

        # ── Dev shell ────────────────────────────────────────────────────────
        devShells.default = pkgs.mkShell {
          name = "symbiont";

          buildInputs = with pkgs; [
            go_1_24
            duckdb
            sqlite
            gopls
            delve
            golangci-lint
            nodejs_22
            nodePackages.typescript
            gotools
            jq
            curl
          ];

          shellHook = ''
            echo "Symbiont dev shell ready."
            echo "Go: $(go version)"
            echo "DuckDB: $(duckdb --version 2>/dev/null || echo 'not found')"
            echo "Node: $(node --version 2>/dev/null || echo 'not found')"
            echo ""
            echo "Dev:     go run ./cmd/api (requires .env)"
            echo "Release: make build  →  ./symbiont serve"
          '';
        };

        # ── NixOS module ─────────────────────────────────────────────────────
        # Single systemd service running `symbiont serve`.
        # Works with either the source-built or pre-built package.
        #
        # In configuration.nix:
        #
        #   inputs.symbiont.url = "github:kjaebker/Symbiont";
        #
        #   { inputs, pkgs, ... }: {
        #     imports = [ inputs.symbiont.nixosModules.${pkgs.system}.default ];
        #     services.symbiont = {
        #       enable = true;
        #       # package = inputs.symbiont.packages.${pkgs.system}.symbiont-bin; # pre-built
        #     };
        #   }
        nixosModules.default = { config, lib, pkgs, ... }:
          let
            cfg = config.services.symbiont;
          in
          {
            options.services.symbiont = {
              enable = lib.mkEnableOption "Symbiont Neptune Apex dashboard";

              package = lib.mkOption {
                type = lib.types.package;
                default = symbiont;
                description = "The symbiont binary package (source-built or pre-built).";
              };

              envFile = lib.mkOption {
                type = lib.types.str;
                default = "/etc/symbiont/env";
                description = "Path to environment file containing secrets (SYMBIONT_APEX_URL, etc.).";
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

              systemd.services.symbiont = {
                description = "Symbiont Neptune Apex Dashboard";
                after = [ "network.target" ];
                wantedBy = [ "multi-user.target" ];

                serviceConfig = {
                  ExecStart = "${cfg.package}/bin/symbiont serve";
                  EnvironmentFile = cfg.envFile;
                  WorkingDirectory = cfg.dataDir;
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
