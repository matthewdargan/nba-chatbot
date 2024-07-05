{
  inputs = {
    dream2nix = {
      inputs.nixpkgs.follows = "nixpkgs";
      url = "github:nix-community/dream2nix";
    };
    nixpkgs.url = "nixpkgs/nixos-unstable";
    parts.url = "github:hercules-ci/flake-parts";
    pre-commit-hooks = {
      inputs.nixpkgs.follows = "nixpkgs";
      url = "github:cachix/pre-commit-hooks.nix";
    };
  };
  outputs = inputs:
    inputs.parts.lib.mkFlake {inherit inputs;} {
      imports = [inputs.pre-commit-hooks.flakeModule];
      perSystem = {
        config,
        pkgs,
        ...
      }: let
        package = inputs.dream2nix.lib.evalModules {
          packageSets.nixpkgs = pkgs;
          modules = [
            ./default.nix
            {
              paths.package = ./.;
              paths.projectRoot = ./.;
            }
          ];
        };
      in {
        devShells.default = pkgs.mkShell {
          inputsFrom = [package.devShell];
          packages = [pkgs.ollama];
          shellHook = "${config.pre-commit.installationScript}";
        };
        packages.nba-chatbot = package;
        pre-commit = {
          settings = {
            hooks = {
              alejandra.enable = true;
              deadnix.enable = true;
              ruff.enable = true;
              statix.enable = true;
            };
            src = ./.;
          };
        };
      };
      systems = ["aarch64-darwin" "aarch64-linux" "x86_64-darwin" "x86_64-linux"];
    };
}
