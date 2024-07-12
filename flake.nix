{
  inputs = {
    nix-go = {
      inputs.nixpkgs.follows = "nixpkgs";
      url = "github:matthewdargan/nix-go";
    };
    nixpkgs.url = "nixpkgs/nixos-unstable";
    parts.url = "github:hercules-ci/flake-parts";
    pre-commit-hooks = {
      inputs.nixpkgs.follows = "nixpkgs";
      url = "github:cachix/pre-commit-hooks.nix";
    };
    process-compose.url = "github:Platonic-Systems/process-compose-flake";
    services-flake.url = "github:juspay/services-flake";
  };
  outputs = inputs:
    inputs.parts.lib.mkFlake {inherit inputs;} {
      imports = [
        inputs.pre-commit-hooks.flakeModule
        inputs.process-compose.flakeModule
      ];
      perSystem = {
        config,
        inputs',
        lib,
        pkgs,
        ...
      }: {
        devShells.default = pkgs.mkShell {
          packages = [
            inputs'.nix-go.packages.go
            inputs'.nix-go.packages.golangci-lint
            pkgs.pgcli
          ];
          shellHook = "${config.pre-commit.installationScript}";
        };
        packages.nba-chatbot = inputs'.nix-go.legacyPackages.buildGoModule {
          meta = with lib; {
            description = "NBA RAG Chatbot";
            homepage = "https://github.com/matthewdargan/nba-chatbot";
            license = licenses.bsd3;
            maintainers = with maintainers; [matthewdargan];
          };
          pname = "nba-chatbot";
          src = ./.;
          vendorHash = "sha256-tAsNY+uT0rMoh1geIJymHWOjKmX63Ss5b0VPGgdYw8Y=";
          version = "0.5.0";
        };
        pre-commit = {
          check.enable = false;
          settings = {
            hooks = {
              alejandra.enable = true;
              deadnix.enable = true;
              golangci-lint = {
                enable = true;
                package = inputs'.nix-go.packages.golangci-lint;
              };
              gotest = {
                enable = true;
                package = inputs'.nix-go.packages.go;
              };
              statix.enable = true;
            };
            src = ./.;
          };
        };
        process-compose."services" = {
          imports = [inputs.services-flake.processComposeModules.default];
          services.postgres."pg1" = {
            enable = true;
            initialDatabases = [
              {
                name = "chatbot";
                schemas = [./sql/create-player.sql];
              }
            ];
            package = pkgs.postgresql_16.withPackages (p: [p.pgvector]);
          };
        };
      };
      systems = ["aarch64-darwin" "aarch64-linux" "x86_64-darwin" "x86_64-linux"];
    };
}
